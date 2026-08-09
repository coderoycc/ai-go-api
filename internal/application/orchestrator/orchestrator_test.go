package orchestrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/intent"
	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
	apptools "github.com/coderoycc/ai-go-api/internal/application/tools"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	"github.com/coderoycc/ai-go-api/internal/domain/ports/mocks"
	infratools "github.com/coderoycc/ai-go-api/internal/infrastructure/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestOrchestrator(mockLLM *mocks.MockLLM, mockMemory *mocks.MockMemory, mockProduct *mocks.MockProductClient) *orchestrator.Orchestrator {
	// Reglas de política personalizadas para las pruebas
	policyEngine := policies.NewEngine([]models.PolicyRule{
		{
			Name:           "allow_general_and_stock",
			AllowedIntents: []models.IntentType{models.IntentGeneral, models.IntentCheckStock, models.IntentUnknown},
			AllowedTools:   []string{"search_products"},
		},
		{
			Name:          "block_sales",
			DeniedIntents: []models.IntentType{models.IntentCreateSale, models.IntentCancelSale},
			DeniedTools:   []string{"cancel_sale"},
		},
	})

	intentDetector := intent.NewDetector()
	contextManager := appctx.NewManager(mockMemory)

	toolRegistry := apptools.NewRegistry()
	if mockProduct != nil {
		_ = toolRegistry.Register(infratools.NewProductSearchTool(mockProduct))
	}

	toolExecutor := apptools.NewExecutor(toolRegistry, policyEngine)
	formatter := response.NewFormatter()

	return orchestrator.NewOrchestrator(
		mockLLM,
		contextManager,
		intentDetector,
		policyEngine,
		toolRegistry,
		toolExecutor,
		formatter,
	)
}

// Caso A: La política bloquea la intención antes de llamar al LLM (no consume tokens).
func TestOrchestrator_CaseA_PolicyBlocksIntent(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)

	// Mock de memoria retorna nil (sesión nueva) y se puede guardar
	mockMemory.On("Load", mock.Anything, "session-block").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-block", mock.Anything).Return(nil)

	orc := setupTestOrchestrator(mockLLM, mockMemory, nil)

	// Mensaje que gatilla una intención bloqueada (IntentCancelSale)
	input := orchestrator.ChatInput{
		SessionID: "session-block",
		Message:   "Quiero cancelar venta #123",
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Contains(t, res.Message, "no está permitida")
	assert.Equal(t, 0, res.TokensUsed)
	// Verificar que el LLM NUNCA fue llamado
	mockLLM.AssertNotCalled(t, "Chat", mock.Anything, mock.Anything)
}

// Caso B: El LLM responde directamente en lenguaje natural sin requerir herramientas.
func TestOrchestrator_CaseB_LLMDirectNaturalResponse(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)

	mockMemory.On("Load", mock.Anything, "session-direct").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-direct", mock.Anything).Return(nil)

	expectedLLMResp := models.ChatResponse{
		Content:   "Hola! Soy tu asistente de ventas. ¿En qué puedo ayudarte hoy?",
		Model:     "gpt-4o",
		CreatedAt: time.Now(),
		Usage:     &models.TokenUsage{PromptTokens: 10, CompletionTokens: 15, TotalTokens: 25},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(expectedLLMResp, nil)

	orc := setupTestOrchestrator(mockLLM, mockMemory, nil)

	input := orchestrator.ChatInput{
		SessionID: "session-direct",
		Message:   "Hola, buenas tardes",
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Equal(t, "Hola! Soy tu asistente de ventas. ¿En qué puedo ayudarte hoy?", res.Message)
	assert.Equal(t, 25, res.TokensUsed)
	mockLLM.AssertNumberOfCalls(t, "Chat", 1)
}

// Caso C: El LLM retorna Tool Calling (check_stock), el executor la procesa y re-envía la respuesta al LLM.
func TestOrchestrator_CaseC_ToolCallingExecution(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockProduct := new(mocks.MockProductClient)

	mockMemory.On("Load", mock.Anything, "session-tool").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-tool", mock.Anything).Return(nil)

	// Mock de la API externa
	mockProduct.On("SearchProducts", mock.Anything, ports.SearchProductsRequest{Codigo: "PROD-99"}).Return(
		&ports.ProductSearchResponse{
			Total: 1,
			Productos: []ports.Product{
				{Codigo: "PROD-99", Nombre: "Laptop HP", Precio: 1200, Stock: 15},
			},
		},
		nil,
	)

	// Primera llamada al LLM: Retorna ToolCall para search_products
	firstLLMResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{
				ID:        "call_abc123",
				Name:      "search_products",
				Arguments: `{"codigo": "PROD-99"}`,
			},
		},
		Usage: &models.TokenUsage{TotalTokens: 30},
	}

	// Segunda llamada al LLM (después de ejecutar el tool): Retorna respuesta final
	secondLLMResp := models.ChatResponse{
		Content: "El producto PROD-99 cuenta con 15 unidades disponibles en inventario.",
		Usage:   &models.TokenUsage{TotalTokens: 20},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(firstLLMResp, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(secondLLMResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockProduct)

	input := orchestrator.ChatInput{
		SessionID: "session-tool",
		Message:   "¿Cuántas unidades quedan del producto PROD-99?",
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Contains(t, res.Message, "15 unidades disponibles")
	assert.Equal(t, 50, res.TokensUsed) // 30 + 20 tokens acumulados

	mockProduct.AssertCalled(t, "SearchProducts", mock.Anything, ports.SearchProductsRequest{Codigo: "PROD-99"})
	mockLLM.AssertNumberOfCalls(t, "Chat", 2)
}

// Caso D: Fallo en la llamada a la herramienta o respuesta de error de los microservicios.
func TestOrchestrator_CaseD_ToolExecutionFailure(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockProduct := new(mocks.MockProductClient)

	mockMemory.On("Load", mock.Anything, "session-err").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-err", mock.Anything).Return(nil)

	// La API externa responde con error 404/500
	mockProduct.On("SearchProducts", mock.Anything, mock.Anything).Return(nil, errors.New("api no disponible"))

	firstLLMResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{
				ID:        "call_err_123",
				Name:      "search_products",
				Arguments: `{"codigo": "PROD-ERR"}`,
			},
		},
		Usage: &models.TokenUsage{TotalTokens: 15},
	}

	// El LLM recibe el mensaje de error de la herramienta y redacta una respuesta amable
	secondLLMResp := models.ChatResponse{
		Content: "Lo siento, no pude verificar el stock en este momento debido a un problema con el sistema de inventario.",
		Usage:   &models.TokenUsage{TotalTokens: 25},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(firstLLMResp, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(secondLLMResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockProduct)

	input := orchestrator.ChatInput{
		SessionID: "session-err",
		Message:   "Asísteme por favor verificando con tus herramientas la disponibilidad del producto PROD-ERR",
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Contains(t, res.Message, "problema con el sistema de inventario")
	mockProduct.AssertCalled(t, "SearchProducts", mock.Anything, mock.Anything)
}
