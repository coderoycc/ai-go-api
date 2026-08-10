package orchestrator

import (
	"context"
	"encoding/json"
	"log"

	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/intent"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
	"github.com/coderoycc/ai-go-api/internal/application/tools"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

const (
	// systemPrompt es el prompt base del asistente de negocio.
	systemPrompt = `Eres un asistente de ventas inteligente. Tu rol es ayudar a los usuarios con:
- Buscar productos y consultar disponibilidad.
- Verificar stock de productos.
- Crear, consultar y cancelar ventas.

Responde de forma concisa, profesional y en el mismo idioma que el usuario.
Cuando necesites datos, usa las herramientas disponibles. No inventes información.`

	// maxToolCallIterations limita las iteraciones de Tool Calling para prevenir loops infinitos.
	maxToolCallIterations = 5
)

// ChatInput encapsula la entrada de una solicitud de chat del usuario.
type ChatInput struct {
	SessionID  string           `json:"session_id"`
	Message    string           `json:"message"`
	UserID     string           `json:"user_id,omitempty"`
	Permission models.Permission `json:"permission,omitempty"`
}

// Orchestrator es el núcleo coordinador del AI Engine. Ensambla la secuencia completa:
// Intent Detector → Context Manager → LLM Tool Calling → Policy Engine → Response Formatter.
// Solo interactúa con interfaces (puertos), nunca con implementaciones concretas.
type Orchestrator struct {
	llm            ports.LLM
	contextManager *appctx.Manager
	intentDetector *intent.Detector
	policyEngine   *policies.Engine
	toolRegistry   ports.ToolRegistry
	toolExecutor   *tools.Executor
	formatter      *response.Formatter
}

// NewOrchestrator crea una nueva instancia del orquestador con todas sus dependencias inyectadas.
func NewOrchestrator(
	llm ports.LLM,
	contextManager *appctx.Manager,
	intentDetector *intent.Detector,
	policyEngine *policies.Engine,
	toolRegistry ports.ToolRegistry,
	toolExecutor *tools.Executor,
	formatter *response.Formatter,
) *Orchestrator {
	return &Orchestrator{
		llm:            llm,
		contextManager: contextManager,
		intentDetector: intentDetector,
		policyEngine:   policyEngine,
		toolRegistry:   toolRegistry,
		toolExecutor:   toolExecutor,
		formatter:      formatter,
	}
}

// HandleChat procesa una solicitud de chat del usuario ejecutando el pipeline completo de 9 pasos:
// 1. Recepción y permisos (ChatInput desde middleware).
// 2. Definición del grupo de herramientas disponibles (toolRegistry).
// 3. Validación de intención (IntentDetector). Si no es válida -> models.ErrInvalidIntent.
// 4. Envío del mensaje al LLM con las herramientas disponibles.
// 5. Recepción de la herramienta indicada por el LLM.
// 6. Validación de permisos con PolicyEngine (read/write). Si no coincide -> models.ErrPermissionDenied.
// 7. Petición a la herramienta seleccionada. Si no existe o falla -> models.ErrToolNotFound / models.ErrToolExecutionFailed.
// 8. Mapeo de la respuesta desde la herramienta (con ExcludedFields ya aplicados).
// 9. Envío de la respuesta API estructurada.
func (o *Orchestrator) HandleChat(ctx context.Context, input ChatInput) response.APIResponse {
	// 1. CONTEXT MANAGER — Cargar o crear sesión
	session, err := o.contextManager.LoadOrCreate(ctx, input.SessionID)
	if err != nil {
		log.Printf("[orchestrator] error cargando sesión: %v", err)
		return o.formatter.FormatError(input.SessionID, "Error interno al cargar la sesión.")
	}

	o.contextManager.AddUserMessage(ctx, session, input.Message)

	// 2. VALIDAR LA INTENCIÓN DEL MENSAJE
	detectedIntent, confident := o.intentDetector.DetectWithConfidence(input.Message)
	if !confident {
		log.Printf("[orchestrator] intención no válida o desconocida para mensaje: %s", input.Message)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrInvalidIntent.Error())
	}
	o.contextManager.UpdateIntent(session, detectedIntent)

	// 3. EJECUTAR EL MENSAJE CON TODAS LAS HERRAMIENTAS VÍA LLM
	messages := o.contextManager.BuildContextMessages(session, systemPrompt, detectedIntent)
	req := models.ChatRequest{
		SessionID:   input.SessionID,
		Messages:    messages,
		Tools:       o.toolRegistry.Definitions(),
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	resp, err := o.llm.Chat(ctx, req)
	if err != nil {
		log.Printf("[orchestrator] error al consultar LLM: %v", err)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, "Error al comunicarse con el asistente de IA.")
	}

	totalTokens := 0
	if resp.Usage != nil {
		totalTokens = resp.Usage.TotalTokens
	}

	// 4. SI EL LLM NO REQUIERE HERRAMIENTAS, DEVOLVER RESPUESTA NATURAL
	if len(resp.ToolCalls) == 0 {
		o.contextManager.AddAssistantMessage(ctx, session, resp.Content)
		o.saveSession(ctx, session)
		return o.formatter.FormatNatural(input.SessionID, resp.Content, string(detectedIntent), totalTokens)
	}

	// 5. RECEPCIÓN DE LA HERRAMIENTA INDICADA POR EL LLM
	toolCall := resp.ToolCalls[0]

	// 6. VERIFICAR SI EL PERMISO DEL USUARIO (read/write) COINCIDE CON EL DE LA TOOL
	policyResult := o.policyEngine.EvaluateTool(toolCall.Name, input.Permission)
	if !policyResult.Allowed {
		log.Printf("[orchestrator] permiso insuficiente para herramienta %s: %s", toolCall.Name, policyResult.Reason)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrPermissionDenied.Error())
	}

	// 7. PETICIÓN A LA HERRAMIENTA SELECCIONADA
	tool, exists := o.toolRegistry.Get(toolCall.Name)
	if !exists {
		log.Printf("[orchestrator] herramienta no encontrada: %s", toolCall.Name)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrToolNotFound.Error())
	}

	// 8. MAPEO DE RESPUESTA DESDE LA TOOL (Aplica MapResponse y ExcludedFields)
	result, err := tool.Execute(ctx, toolCall.Arguments)
	if err != nil {
		log.Printf("[orchestrator] error ejecutando herramienta %s: %v", toolCall.Name, err)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrToolExecutionFailed.Error())
	}

	// 9. PERSISTIR EN REDIS Y RETORNAR RESPUESTA API ESTRUCTURADA
	resultBytes, _ := json.Marshal(result)
	o.contextManager.AddAssistantMessage(ctx, session, string(resultBytes))
	o.saveSession(ctx, session)
	return o.formatter.FormatRaw(input.SessionID, string(detectedIntent), result)
}

// saveSession persiste la sesión de forma segura (loguea el error sin propagarlo).
func (o *Orchestrator) saveSession(ctx context.Context, session *models.SessionContext) {
	if err := o.contextManager.Save(ctx, session); err != nil {
		log.Printf("[orchestrator] error al guardar sesión %s: %v", session.SessionID, err)
	}
}
