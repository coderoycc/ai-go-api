package orchestrator_test

import (
	"context"
	"encoding/json"
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
	"github.com/coderoycc/ai-go-api/internal/domain/ports/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTool implementa ports.Tool para pruebas unitarias de manera desacoplada.
type MockTool struct {
	mock.Mock
	ToolName string
}

func (m *MockTool) Name() string                     { return m.ToolName }
func (m *MockTool) Description() string              { return "Mock tool for testing" }
func (m *MockTool) Method() string                   { return "POST" }
func (m *MockTool) EndpointURL() string              { return "http://localhost/mock" }
func (m *MockTool) Parameters() map[string]any       { return map[string]any{"type": "object"} }
func (m *MockTool) ResponseSchema() map[string]any   { return map[string]any{"type": "object"} }
func (m *MockTool) ExcludedFields() []string         { return nil }
func (m *MockTool) MapResponse(rawBody []byte) (any, error) {
	var data any
	_ = json.Unmarshal(rawBody, &data)
	return data, nil
}
func (m *MockTool) Execute(ctx context.Context, arguments string) (any, error) {
	args := m.Called(ctx, arguments)
	return args.Get(0), args.Error(1)
}

func setupTestOrchestrator(mockLLM *mocks.MockLLM, mockMemory *mocks.MockMemory, mockTool *MockTool) *orchestrator.Orchestrator {
	// Reglas de política personalizadas para las pruebas
	policyEngine := policies.NewEngine([]models.PolicyRule{
		{
			Name:           "allow_general_and_stock",
			AllowedIntents: []models.IntentType{models.IntentGeneral, models.IntentCheckStock, models.IntentUnknown, models.IntentSearchProduct},
			AllowedTools:   []string{"product_advanced_search", "product_stock_check", "search_products"},
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
	if mockTool != nil {
		_ = toolRegistry.Register(mockTool)
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

// Caso C: El LLM retorna Tool Calling (product_advanced_search), el executor la procesa y re-envía la respuesta al LLM.
func TestOrchestrator_CaseC_ToolCallingExecution(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockTool := new(MockTool)
	mockTool.ToolName = "product_advanced_search"

	mockMemory.On("Load", mock.Anything, "session-tool").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-tool", mock.Anything).Return(nil)

	// Mock de la tool ejecutando
	mockTool.On("Execute", mock.Anything, `{"codigo": "PROD-99"}`).Return(
		map[string]any{
			"total": 1,
			"productos": []any{
				map[string]any{"codigo": "PROD-99", "nombre": "Laptop HP", "precio": 1200, "stock": 15},
			},
		},
		nil,
	)

	// Primera llamada al LLM: Retorna ToolCall para product_advanced_search
	firstLLMResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{
				ID:        "call_abc123",
				Name:      "product_advanced_search",
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

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockTool)

	input := orchestrator.ChatInput{
		SessionID: "session-tool",
		Message:   "¿Cuántas unidades quedan del producto PROD-99?",
	}

	resResult := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeNatural, resResult.Mode)
	assert.Contains(t, resResult.Message, "15 unidades disponibles")
	assert.Equal(t, 50, resResult.TokensUsed) // 30 + 20 tokens acumulados

	mockTool.AssertCalled(t, "Execute", mock.Anything, `{"codigo": "PROD-99"}`)
	mockLLM.AssertNumberOfCalls(t, "Chat", 2)
}

// Caso D: Fallo en la llamada a la herramienta.
func TestOrchestrator_CaseD_ToolExecutionFailure(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockTool := new(MockTool)
	mockTool.ToolName = "product_advanced_search"

	mockMemory.On("Load", mock.Anything, "session-err").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-err", mock.Anything).Return(nil)

	// La tool responde con error
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(nil, errors.New("api no disponible"))

	firstLLMResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{
				ID:        "call_err_123",
				Name:      "product_advanced_search",
				Arguments: `{"codigo": "PROD-ERR"}`,
			},
		},
		Usage: &models.TokenUsage{TotalTokens: 15},
	}

	secondLLMResp := models.ChatResponse{
		Content: "Lo siento, no pude verificar el stock en este momento debido a un problema con el sistema de inventario.",
		Usage:   &models.TokenUsage{TotalTokens: 25},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(firstLLMResp, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(secondLLMResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockTool)

	input := orchestrator.ChatInput{
		SessionID: "session-err",
		Message:   "Asísteme por favor verificando con tus herramientas la disponibilidad del producto PROD-ERR",
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Contains(t, res.Message, "problema con el sistema de inventario")
	mockTool.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
}
