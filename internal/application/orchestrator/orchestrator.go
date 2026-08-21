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
	// defaultSystemPrompt es el prompt de respaldo del orquestador cuando ningún endpoint
	// inyecta un SystemPrompt personalizado vía HandleChatWithPrompt.
	// Debe ser genérico y agnóstico al dominio: la lógica de negocio pertenece
	// a cada FeatureRecipe definida en la capa de infraestructura (router/engine).
	defaultSystemPrompt = `Eres un asistente inteligente. Responde de forma concisa y profesional en el mismo idioma que el usuario. Cuando necesites datos externos, usa las herramientas disponibles. No inventes información.`

	// maxToolCallIterations limita las iteraciones del loop agente para prevenir ciclos infinitos.
	// Con 5 iteraciones se soportan flujos como: listar marcas -> buscar por ID -> consultar stock -> responder.
	maxToolCallIterations = 5
)

// ChatInput encapsula la entrada de una solicitud de chat del usuario.
type ChatInput struct {
	SessionID  string            `json:"session_id"`
	Message    string            `json:"message"`
	UserID     string            `json:"user_id,omitempty"`
	Permission models.Permission `json:"permission,omitempty"`
}

// Orchestrator es el nucleo coordinador del AI Engine. Ensambla la secuencia completa:
// Intent Detector -> Context Manager -> LLM Agentic Loop (Tool Calling) -> Policy Engine -> Response Formatter.
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

// HandleChat procesa una solicitud de chat usando el prompt de respaldo generico del orquestador.
// Para funcionalidades de dominio especifico, usar HandleChatWithPrompt con un prompt dedicado.
func (o *Orchestrator) HandleChat(ctx context.Context, input ChatInput, tools []ports.Tool) response.APIResponse {
	return o.HandleChatWithPrompt(ctx, input, tools, defaultSystemPrompt)
}

// HandleChatWithPrompt procesa una solicitud de chat ejecutando el pipeline completo:
//  1. Carga o crea la sesion del usuario.
//  2. Agrega el mensaje del usuario al historial.
//  3. Valida la intencion del mensaje via IntentDetector.
//  4. Ejecuta el loop agente: llama al LLM, ejecuta tools si las solicita, repite hasta
//     que el LLM genere una respuesta natural o se alcance maxToolCallIterations.
func (o *Orchestrator) HandleChatWithPrompt(ctx context.Context, input ChatInput, tools []ports.Tool, customSystemPrompt string) response.APIResponse {
	if customSystemPrompt == "" {
		customSystemPrompt = defaultSystemPrompt
	}

	// 1. CONTEXT MANAGER - Cargar o crear sesion
	session, err := o.contextManager.LoadOrCreate(ctx, input.SessionID)
	if err != nil {
		log.Printf("[orchestrator] error cargando sesion: %v", err)
		return o.formatter.FormatError(input.SessionID, "Error interno al cargar la sesion.")
	}

	// 2. Agregar el mensaje del usuario al historial
	o.contextManager.AddUserMessage(session, input.Message)

	// 3. Validar la intencion del mensaje
	isValid, actionType, err := o.intentDetector.Validate(ctx, input.Message)
	if err != nil || !isValid {
		log.Printf("[orchestrator] intencion no valida para mensaje: %q (err: %v)", input.Message, err)
		o.saveSession(ctx, session)
		return o.formatter.FormatError(input.SessionID, models.ErrInvalidIntent.Error())
	}

	// 4. Ejecutar el loop agente
	return o.runAgentLoop(ctx, input, session, tools, customSystemPrompt, actionType)
}

// runAgentLoop implementa el ciclo agente completo de Tool Calling.
//
// En cada iteracion:
//   - Construye el contexto de mensajes desde la sesion (incluye resultados de tools anteriores).
//   - Llama al LLM con las herramientas disponibles.
//   - Si el LLM responde en lenguaje natural: retorna la respuesta formateada.
//   - Si el LLM solicita tools: valida permisos, ejecuta cada tool, persiste los resultados
//     en el historial con RoleTool + ToolCallID y continua al siguiente ciclo.
//
// Ejemplo de flujo de 2 iteraciones (marca -> producto):
//
//	Iter 1: LLM -> tool get_brands()       -> [{"id":5,"nombre":"Monopol"}]
//	Iter 2: LLM -> tool search_products(5) -> [{"codigo":"P1","nombre":"Pintura Monopol"}]
//	Iter 3: LLM -> "Encontre 3 productos de la marca Monopol: ..."  OK
func (o *Orchestrator) runAgentLoop(
	ctx context.Context,
	input ChatInput,
	session *models.SessionContext,
	tools []ports.Tool,
	customSystemPrompt string,
	actionType string,
) response.APIResponse {
	toolDefs := definitions(tools)
	totalTokens := 0

	for iteration := 0; iteration < maxToolCallIterations; iteration++ {
		log.Printf("[orchestrator] loop agente - iteracion %d/%d (sesion: %s)", iteration+1, maxToolCallIterations, input.SessionID)

		// Construir mensajes desde el historial actualizado (incluye resultados de tools anteriores)
		messages := o.contextManager.BuildContextMessages(session, customSystemPrompt)
		req := models.ChatRequest{
			SessionID:   input.SessionID,
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.7,
			MaxTokens:   1024,
		}

		resp, err := o.llm.Chat(ctx, req)
		if err != nil {
			log.Printf("[orchestrator] error al consultar LLM (iter %d): %v", iteration+1, err)
			o.saveSession(ctx, session)
			return o.formatter.FormatError(input.SessionID, "Error al comunicarse con el asistente de IA.")
		}

		if resp.Usage != nil {
			totalTokens += resp.Usage.TotalTokens
		}

		// El LLM respondio en lenguaje natural -> fin del loop
		if len(resp.ToolCalls) == 0 {
			log.Printf("[orchestrator] respuesta natural del LLM en iteracion %d (tokens totales: %d)", iteration+1, totalTokens)
			o.contextManager.AddAssistantMessage(session, resp.Content)
			o.saveSession(ctx, session)
			return o.formatter.FormatNatural(input.SessionID, resp.Content, actionType, totalTokens)
		}

		log.Printf("[orchestrator] LLM solicita %d herramienta(s) en iteracion %d", len(resp.ToolCalls), iteration+1)

		// Persistir el mensaje del asistente con sus solicitudes de tools ANTES de los resultados.
		// Esto es requerido por todos los proveedores LLM para reconstruir el historial correctamente.
		o.contextManager.AddAssistantToolCallMessage(session, resp.Content, resp.ToolCalls)

		// Ejecutar TODAS las tools solicitadas en esta iteracion
		for _, toolCall := range resp.ToolCalls {
			log.Printf("[orchestrator] ejecutando tool %q con args: %s", toolCall.Name, toolCall.Arguments)

			// Buscar la herramienta en las disponibles para este endpoint
			tool, exists := findTool(tools, toolCall.Name)
			if !exists {
				log.Printf("[orchestrator] herramienta no encontrada: %s", toolCall.Name)
				o.saveSession(ctx, session)
				return o.formatter.FormatError(input.SessionID, models.ErrToolNotFound.Error())
			}

			// Verificar permiso del usuario contra el requerido por la tool
			policyResult := o.policyEngine.EvaluateTool(tool, input.Permission)
			if !policyResult.Allowed {
				log.Printf("[orchestrator] permiso insuficiente para tool %q: %s", toolCall.Name, policyResult.Reason)
				o.saveSession(ctx, session)
				return o.formatter.FormatError(input.SessionID, models.ErrPermissionDenied.Error())
			}

			// Ejecutar la herramienta
			result, err := tool.Execute(ctx, toolCall.Arguments)
			if err != nil {
				log.Printf("[orchestrator] error ejecutando tool %q: %v", toolCall.Name, err)
				o.saveSession(ctx, session)
				return o.formatter.FormatError(input.SessionID, models.ErrToolExecutionFailed.Error())
			}

			// Serializar resultado para el historial
			resultBytes, err := json.Marshal(result)
			if err != nil {
				log.Printf("[orchestrator] error serializando resultado de tool %q: %v", toolCall.Name, err)
				o.saveSession(ctx, session)
				return o.formatter.FormatError(input.SessionID, "Error al procesar el resultado de la herramienta.")
			}

			log.Printf("[orchestrator] tool %q completada - resultado: %d bytes", toolCall.Name, len(resultBytes))

			// Registrar el resultado en la sesion con RoleTool (rol correcto para todos los LLMs)
			o.contextManager.UpdateLastTool(session, toolCall.Name)
			o.contextManager.AddToolResultMessage(session, toolCall.ID, toolCall.Name, string(resultBytes))
		}

		// Continuar al siguiente ciclo con el historial enriquecido
	}

	// Se alcanzo el limite de iteraciones sin que el LLM generara una respuesta final
	log.Printf("[orchestrator] limite de iteraciones alcanzado (%d) para sesion %s", maxToolCallIterations, input.SessionID)
	o.saveSession(ctx, session)
	return o.formatter.FormatError(input.SessionID, "El asistente requirio demasiados pasos para responder. Intenta reformular tu pregunta.")
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

// saveSession persiste la sesion de forma segura (loguea el error sin propagarlo).
func (o *Orchestrator) saveSession(ctx context.Context, session *models.SessionContext) {
	if err := o.contextManager.Save(ctx, session); err != nil {
		log.Printf("[orchestrator] error al guardar sesion %s: %v", session.SessionID, err)
	}
}
