package orchestrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	"github.com/coderoycc/ai-go-api/internal/domain/ports/mocks"
	regexIntent "github.com/coderoycc/ai-go-api/internal/infrastructure/intent/regex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTool implementa ports.Tool para pruebas unitarias de manera desacoplada.
type MockTool struct {
	mock.Mock
	ToolName      string
	ReqPermission models.Permission
}

func (m *MockTool) Name() string               { return m.ToolName }
func (m *MockTool) Description() string        { return "Mock tool for testing" }
func (m *MockTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (m *MockTool) RequiredPermission() models.Permission {
	if m.ReqPermission != "" {
		return m.ReqPermission
	}
	return models.PermissionRead
}
func (m *MockTool) Execute(ctx context.Context, arguments string) (any, error) {
	args := m.Called(ctx, arguments)
	return args.Get(0), args.Error(1)
}

func setupTestOrchestrator(mockLLM *mocks.MockLLM, mockMemory *mocks.MockMemory, tools ...ports.Tool) *orchestrator.Orchestrator {
	intentDetector := regexIntent.NewDetector()
	contextManager := appctx.NewManager(mockMemory)
	policyEngine := policies.NewEngine()
	formatter := response.NewFormatter()
	return orchestrator.NewOrchestrator(mockLLM, contextManager, intentDetector, policyEngine, formatter)
}

func (m *MockTool) toolSet() []ports.Tool {
	return []ports.Tool{m}
}

// Caso A: La politica bloquea la ejecucion de una herramienta si el usuario carece de permisos.
func TestOrchestrator_CaseA_PolicyBlocksToolExecution(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockTool := new(MockTool)
	mockTool.ToolName = "cancel_sale"
	mockTool.ReqPermission = models.PermissionWrite

	mockMemory.On("Load", mock.Anything, "session-block").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-block", mock.Anything).Return(nil)

	llmResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{ID: "call_cancel_1", Name: "cancel_sale", Arguments: `{"sale_id": "123"}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 10},
	}
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory)
	input := orchestrator.ChatInput{
		SessionID:  "session-block",
		Message:    "Quiero cancelar la venta #123",
		Permission: models.PermissionRead,
	}

	res := orc.HandleChat(context.Background(), input, mockTool.toolSet())

	assert.Contains(t, res.Message, models.ErrPermissionDenied.Error())
	mockLLM.AssertNumberOfCalls(t, "Chat", 1)
}

// Caso B: El LLM responde directamente en lenguaje natural sin requerir herramientas.
func TestOrchestrator_CaseB_LLMDirectNaturalResponse(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)

	mockMemory.On("Load", mock.Anything, "session-direct").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-direct", mock.Anything).Return(nil)

	expectedLLMResp := models.ChatResponse{
		Content:   "Hola! Soy tu asistente de ventas. En que puedo ayudarte hoy?",
		Model:     "gpt-4o",
		CreatedAt: time.Now(),
		Usage:     &models.TokenUsage{PromptTokens: 10, CompletionTokens: 15, TotalTokens: 25},
	}
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(expectedLLMResp, nil)

	orc := setupTestOrchestrator(mockLLM, mockMemory)
	input := orchestrator.ChatInput{
		SessionID: "session-direct",
		Message:   "Hola, buenas tardes",
	}

	res := orc.HandleChat(context.Background(), input, nil)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Equal(t, "Hola! Soy tu asistente de ventas. En que puedo ayudarte hoy?", res.Message)
	assert.Equal(t, 25, res.TokensUsed)
	mockLLM.AssertNumberOfCalls(t, "Chat", 1)
}

// Caso C: El LLM retorna Tool Calling, la tool se ejecuta, el LLM recibe el resultado
// y genera una respuesta en lenguaje natural (loop de 2 iteraciones).
func TestOrchestrator_CaseC_ToolCallingLoopAndNaturalResponse(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockTool := new(MockTool)
	mockTool.ToolName = "product_advanced_search"

	mockMemory.On("Load", mock.Anything, "session-tool").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-tool", mock.Anything).Return(nil)

	mockTool.On("Execute", mock.Anything, `{"codigo": "PROD-99"}`).Return(
		map[string]any{
			"total": 1,
			"productos": []any{
				map[string]any{"codigo": "PROD-99", "nombre": "Laptop HP", "precio": 1200, "stock": 15},
			},
		},
		nil,
	)

	// Iteracion 1: el LLM solicita la tool
	llmRespWithTool := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{ID: "call_abc123", Name: "product_advanced_search", Arguments: `{"codigo": "PROD-99"}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 30},
	}
	// Iteracion 2: el LLM recibe el resultado y responde en lenguaje natural
	llmRespNatural := models.ChatResponse{
		Content:   "Encontre el producto PROD-99: Laptop HP con precio $1200 y 15 unidades en stock.",
		ToolCalls: nil,
		Usage:     &models.TokenUsage{TotalTokens: 45},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmRespWithTool, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmRespNatural, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory)
	input := orchestrator.ChatInput{
		SessionID:  "session-tool",
		Message:    "Cuantas unidades quedan del producto PROD-99?",
		Permission: models.PermissionRead,
	}

	res := orc.HandleChat(context.Background(), input, mockTool.toolSet())

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Contains(t, res.Message, "Laptop HP")
	assert.Equal(t, 75, res.TokensUsed) // 30 + 45 acumulados
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
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(nil, errors.New("api no disponible"))

	llmResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{ID: "call_err_123", Name: "product_advanced_search", Arguments: `{"codigo": "PROD-ERR"}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 15},
	}
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory)
	input := orchestrator.ChatInput{
		SessionID:  "session-err",
		Message:    "Asisteme por favor verificando con tus herramientas la disponibilidad del producto PROD-ERR",
		Permission: models.PermissionRead,
	}

	res := orc.HandleChat(context.Background(), input, mockTool.toolSet())

	assert.Contains(t, res.Message, models.ErrToolExecutionFailed.Error())
	mockTool.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
}

// Caso E: Loop de 2 iteraciones simulando el flujo real marca -> producto.
//
// Iter 1: LLM solicita get_brands()       -> devuelve [{id:5, nombre:"Monopol"}, ...]
// Iter 2: LLM solicita search_products(5) -> devuelve productos filtrados por ID de marca
// Iter 3: LLM responde en lenguaje natural con el resumen final.
func TestOrchestrator_CaseE_MultiToolLoop_BrandToProduct(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)

	mockBrandTool := &MockTool{ToolName: "get_brands"}
	mockSearchTool := &MockTool{ToolName: "product_advanced_search"}
	tools := []ports.Tool{mockBrandTool, mockSearchTool}

	mockMemory.On("Load", mock.Anything, "session-multi").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-multi", mock.Anything).Return(nil)

	// Iteracion 1: LLM pide el listado de marcas
	llmResp1 := models.ChatResponse{
		ToolCalls: []models.ToolCall{
			{ID: "call_brands_1", Name: "get_brands", Arguments: `{}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 20},
	}
	mockBrandTool.On("Execute", mock.Anything, `{}`).Return(
		[]any{
			map[string]any{"id": 5, "nombre": "Monopol"},
			map[string]any{"id": 8, "nombre": "Bayer"},
		},
		nil,
	)

	// Iteracion 2: LLM usa el ID de Monopol para buscar sus productos
	llmResp2 := models.ChatResponse{
		ToolCalls: []models.ToolCall{
			{ID: "call_search_1", Name: "product_advanced_search", Arguments: `{"marca_id": 5}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 35},
	}
	mockSearchTool.On("Execute", mock.Anything, `{"marca_id": 5}`).Return(
		map[string]any{
			"total":     3,
			"productos": []any{"Pintura Monopol 1L", "Pintura Monopol 4L", "Barniz Monopol"},
		},
		nil,
	)

	// Iteracion 3: LLM responde en lenguaje natural
	llmResp3 := models.ChatResponse{
		Content: "Encontre 3 productos de la marca Monopol: Pintura 1L, Pintura 4L y Barniz.",
		Usage:   &models.TokenUsage{TotalTokens: 50},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp1, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp2, nil).Once()
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp3, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory)
	input := orchestrator.ChatInput{
		SessionID:  "session-multi",
		Message:    "Que productos de la marca Monopol tienen disponibles?",
		Permission: models.PermissionRead,
	}

	res := orc.HandleChat(context.Background(), input, tools)

	assert.Equal(t, response.ModeNatural, res.Mode)
	assert.Contains(t, res.Message, "Monopol")
	assert.Equal(t, 105, res.TokensUsed) // 20 + 35 + 50 tokens acumulados
	mockBrandTool.AssertCalled(t, "Execute", mock.Anything, `{}`)
	mockSearchTool.AssertCalled(t, "Execute", mock.Anything, `{"marca_id": 5}`)
	mockLLM.AssertNumberOfCalls(t, "Chat", 3)
}
