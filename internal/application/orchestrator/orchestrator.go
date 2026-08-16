package orchestrator

import (
	"context"
	"encoding/json"
	"log"

	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
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
type Orchestrator struct {
	llm            ports.LLM
	contextManager *appctx.Manager
	intentDetector ports.IntentDetector
	policyEngine   *policies.Engine
	formatter      *response.Formatter
}

// NewOrchestrator crea una nueva instancia del orquestador con todas sus dependencias inyectadas.
func NewOrchestrator(
	llm ports.LLM,
	contextManager *appctx.Manager,
	intentDetector ports.IntentDetector,
	policyEngine *policies.Engine,
	formatter *response.Formatter,
) *Orchestrator {
	return &Orchestrator{
		llm:            llm,
		contextManager: contextManager,
		intentDetector: intentDetector,
		policyEngine:   policyEngine,
		formatter:      formatter,
	}
}

// HandleChat procesa una solicitud de chat del usuario ejecutando el pipeline completo:
// 1. Recepción y permisos (ChatInput desde middleware).
// 2. Carga/Creación de sesión y adición de mensaje.
// 3. Validación de la veracidad de intención (IntentDetector). Si no es válida -> models.ErrInvalidIntent.
// 4. Envío del mensaje al LLM con las herramientas disponibles (tools).
// 5. Recepción de la respuesta o herramienta indicada por el LLM.
// 6. Validación de permisos con PolicyEngine (read/write). Si no coincide -> models.ErrPermissionDenied.
// 7. Petición a la herramienta seleccionada. Si no existe o falla -> models.ErrToolNotFound / models.ErrToolExecutionFailed.
// 8. Registro de la respuesta y retorno formateado al cliente API.
func (o *Orchestrator) HandleChat(ctx context.Context, input ChatInput, tools []ports.Tool) response.APIResponse {
	// 1. CONTEXT MANAGER — Cargar o crear sesión
	session, err := o.contextManager.LoadOrCreate(ctx, input.SessionID)
	if err != nil {
		log.Printf("[orchestrator] error cargando sesión: %v", err)
		return o.formatter.FormatError(input.SessionID, "Error interno al cargar la sesión.")
	}

	o.contextManager.AddUserMessage(ctx, session, input.Message)

	// 2. VALIDAR LA VERACIDAD E INTENCIÓN DEL MENSAJE VÍA PORTS.INTENTDETECTOR
	isValid, actionType, err := o.intentDetector.Validate(ctx, input.Message)
	if err != nil || !isValid {
		log.Printf("[orchestrator] intención no válida o rechazada para mensaje: %s (err: %v)", input.Message, err)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrInvalidIntent.Error())
	}

	// 3. EJECUTAR EL MENSAJE CON LAS HERRAMIENTAS DISPONIBLES VÍA LLM
	messages := o.contextManager.BuildContextMessages(session, systemPrompt)
	req := models.ChatRequest{
		SessionID:   input.SessionID,
		Messages:    messages,
		Tools:       definitions(tools),
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
		return o.formatter.FormatNatural(input.SessionID, resp.Content, actionType, totalTokens)
	}

	// 5. RECEPCIÓN DE LA HERRAMIENTA INDICADA POR EL LLM
	toolCall := resp.ToolCalls[0]

	// 6. BUSCAR LA HERRAMIENTA EN LAS DISPONIBLES PARA ESTE ENDPOINT
	tool, exists := findTool(tools, toolCall.Name)
	if !exists {
		log.Printf("[orchestrator] herramienta no encontrada: %s", toolCall.Name)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrToolNotFound.Error())
	}

	// 7. VERIFICAR SI EL PERMISO DEL USUARIO (read/write) COINCIDE CON EL DE LA TOOL
	policyResult := o.policyEngine.EvaluateTool(tool, input.Permission)
	if !policyResult.Allowed {
		log.Printf("[orchestrator] permiso insuficiente para herramienta %s: %s", toolCall.Name, policyResult.Reason)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrPermissionDenied.Error())
	}

	// 8. EJECUCIÓN DE LA HERRAMIENTA (Aplica HTTP y ExcludedFields automáticamente)
	result, err := tool.Execute(ctx, toolCall.Arguments)
	if err != nil {
		log.Printf("[orchestrator] error ejecutando herramienta %s: %v", toolCall.Name, err)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrToolExecutionFailed.Error())
	}

	// 9. ACTUALIZAR ÚLTIMA HERRAMIENTA Y RETORNAR RESPUESTA API ESTRUCTURADA
	o.contextManager.UpdateLastTool(session, toolCall.Name)
	resultBytes, _ := json.Marshal(result)
	o.contextManager.AddAssistantMessage(ctx, session, string(resultBytes))
	o.saveSession(ctx, session)
	return o.formatter.FormatRaw(input.SessionID, toolCall.Name, result)
}

// definitions convierte las herramientas disponibles en el formato que espera el LLM.
func definitions(tools []ports.Tool) []models.ToolDefinition {
	defs := make([]models.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, models.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return defs
}

// findTool busca una herramienta por nombre dentro de las disponibles.
func findTool(tools []ports.Tool, name string) (ports.Tool, bool) {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool, true
		}
	}
	return nil, false
}

// saveSession persiste la sesión de forma segura (loguea el error sin propagarlo).
func (o *Orchestrator) saveSession(ctx context.Context, session *models.SessionContext) {
	if err := o.contextManager.Save(ctx, session); err != nil {
		log.Printf("[orchestrator] error al guardar sesión %s: %v", session.SessionID, err)
	}
}
