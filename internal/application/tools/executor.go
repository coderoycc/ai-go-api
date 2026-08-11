package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// Executor recibe ToolCalls del LLM, valida con el PolicyEngine y ejecuta
// la herramienta correspondiente en los clientes de microservicios.
type Executor struct {
	registry     ports.ToolRegistry
	policyEngine *policies.Engine
}

// NewExecutor crea un nuevo ejecutor de herramientas con su registry y motor de políticas.
func NewExecutor(registry ports.ToolRegistry, policyEngine *policies.Engine) *Executor {
	return &Executor{
		registry:     registry,
		policyEngine: policyEngine,
	}
}

// ExecuteToolCall ejecuta una sola llamada a herramienta después de validar la política.
// Retorna el resultado como string para inyectar de vuelta al LLM como mensaje tool.
func (e *Executor) ExecuteToolCall(ctx context.Context, toolCall models.ToolCall, sessionID string, userPerm models.Permission) (string, error) {
	// 1. Validar con PolicyEngine antes de ejecutar
	policyResult := e.policyEngine.EvaluateTool(toolCall.Name, userPerm)

	if !policyResult.Allowed {
		return "", fmt.Errorf("executor: herramienta '%s' bloqueada — %s", toolCall.Name, policyResult.Reason)
	}

	// 2. Buscar la herramienta en el registry
	tool, exists := e.registry.Get(toolCall.Name)
	if !exists {
		return "", fmt.Errorf("executor: herramienta '%s' no registrada", toolCall.Name)
	}

	// 3. Ejecutar la herramienta
	result, err := tool.Execute(ctx, toolCall.Arguments)
	if err != nil {
		return "", fmt.Errorf("executor: error al ejecutar '%s': %w", toolCall.Name, err)
	}

	// 4. Serializar el objeto estructurado devuelto por la tool a JSON string para el LLM
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("executor: error al serializar resultado de '%s': %w", toolCall.Name, err)
	}

	return string(resultBytes), nil
}

// ExecuteAll ejecuta todas las ToolCalls retornadas por el LLM secuencialmente.
// Retorna los mensajes de resultado para inyectar de vuelta a la conversación.
func (e *Executor) ExecuteAll(ctx context.Context, toolCalls []models.ToolCall, sessionID string, userPerm models.Permission) ([]models.Message, error) {
	results := make([]models.Message, 0, len(toolCalls))

	for _, tc := range toolCalls {
		result, err := e.ExecuteToolCall(ctx, tc, sessionID, userPerm)
		if err != nil {
			// En caso de error, inyectar el error como resultado para que el LLM lo maneje
			results = append(results, models.Message{
				Role:       models.RoleTool,
				Content:    fmt.Sprintf(`{"error": "%s"}`, err.Error()),
				ToolCallID: tc.ID,
			})
			continue
		}

		results = append(results, models.Message{
			Role:       models.RoleTool,
			Content:    result,
			ToolCallID: tc.ID,
		})
	}

	return results, nil
}
