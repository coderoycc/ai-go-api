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
	apptools "github.com/coderoycc/ai-go-api/internal/application/tools"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
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

func (m *MockTool) Name() string                     { return m.ToolName }
func (m *MockTool) Description() string              { return "Mock tool for testing" }
func (m *MockTool) Parameters() map[string]any       { return map[string]any{"type": "object"} }
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

func setupTestOrchestrator(mockLLM *mocks.MockLLM, mockMemory *mocks.MockMemory, mockTool *MockTool) *orchestrator.Orchestrator {
	intentDetector := regexIntent.NewDetector()
	contextManager := appctx.NewManager(mockMemory)

	toolRegistry := apptools.NewRegistry()
	if mockTool != nil {
		_ = toolRegistry.Register(mockTool)
	}

	policyEngine := policies.NewEngine(toolRegistry)
	formatter := response.NewFormatter()

	return orchestrator.NewOrchestrator(
		mockLLM,
		contextManager,
		intentDetector,
		policyEngine,
		toolRegistry,
		formatter,
	)
}

// Caso A: La política (PolicyEngine) bloquea la ejecución de una herramienta si el usuario carece de permisos.
func TestOrchestrator_CaseA_PolicyBlocksToolExecution(t *testing.T) {
	mockLLM := new(mocks.MockLLM)
	mockMemory := new(mocks.MockMemory)
	mockTool := new(MockTool)
	mockTool.ToolName = "cancel_sale"
	mockTool.ReqPermission = models.PermissionWrite

	mockMemory.On("Load", mock.Anything, "session-block").Return((*models.SessionContext)(nil), nil)
	mockMemory.On("Save", mock.Anything, "session-block", mock.Anything).Return(nil)

	// El LLM solicita ejecutar cancel_sale
	llmResp := models.ChatResponse{
		Content: "",
		ToolCalls: []models.ToolCall{
			{ID: "call_cancel_1", Name: "cancel_sale", Arguments: `{"sale_id": "123"}`},
		},
		Usage: &models.TokenUsage{TotalTokens: 10},
	}

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockTool)

	input := orchestrator.ChatInput{
		SessionID:  "session-block",
		Message:    "Quiero cancelar la venta #123",
		Permission: models.PermissionRead, // solo lectura, pero cancel_sale requiere write
	}

	res := orc.HandleChat(context.Background(), input)

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

// Caso C: El LLM retorna Tool Calling (product_advanced_search), se valida permiso, se ejecuta la herramienta y se responde con datos mapeados.
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

	llmResp := models.ChatResponse{
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

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockTool)

	input := orchestrator.ChatInput{
		SessionID:  "session-tool",
		Message:    "¿Cuántas unidades quedan del producto PROD-99?",
		Permission: models.PermissionRead,
	}

	resResult := orc.HandleChat(context.Background(), input)

	assert.Equal(t, response.ModeRaw, resResult.Mode)
	assert.NotNil(t, resResult.Data)
	mockTool.AssertCalled(t, "Execute", mock.Anything, `{"codigo": "PROD-99"}`)
	mockLLM.AssertNumberOfCalls(t, "Chat", 1)
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

	llmResp := models.ChatResponse{
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

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return(llmResp, nil).Once()

	orc := setupTestOrchestrator(mockLLM, mockMemory, mockTool)

	input := orchestrator.ChatInput{
		SessionID:  "session-err",
		Message:    "Asísteme por favor verificando con tus herramientas la disponibilidad del producto PROD-ERR",
		Permission: models.PermissionRead,
	}

	res := orc.HandleChat(context.Background(), input)

	assert.Contains(t, res.Message, models.ErrToolExecutionFailed.Error())
	mockTool.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
}
