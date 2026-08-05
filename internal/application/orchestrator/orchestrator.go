package orchestrator

import (
	"context"
	"fmt"
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
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// Orchestrator es el núcleo coordinador del AI Engine. Ensambla la secuencia completa:
// Policy Engine → Intent Detector → Context Manager → Evaluador de Ruta → Tool Calling → Response Formatter.
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

// HandleChat procesa una solicitud de chat del usuario ejecutando el pipeline completo.
func (o *Orchestrator) HandleChat(ctx context.Context, input ChatInput) response.APIResponse {
	// 1. CONTEXT MANAGER — Cargar o crear sesión
	session, err := o.contextManager.LoadOrCreate(ctx, input.SessionID)
	if err != nil {
		log.Printf("[orchestrator] error cargando sesión: %v", err)
		return o.formatter.FormatError(input.SessionID, "Error interno al cargar la sesión.")
	}

	// Agregar mensaje del usuario al historial
	o.contextManager.AddUserMessage(ctx, session, input.Message)

	// 2. INTENT DETECTOR — Clasificar intención sin consumir tokens
	detectedIntent, confident := o.intentDetector.DetectWithConfidence(input.Message)
	o.contextManager.UpdateIntent(session, detectedIntent)

	// 3. POLICY ENGINE — ¿La intención está permitida?
	if confident {
		policyResult := o.policyEngine.EvaluateIntent(models.PolicyEvalRequest{
			SessionID: input.SessionID,
			Intent:    detectedIntent,
		})

		if !policyResult.Allowed {
			log.Printf("[orchestrator] intención bloqueada: %s — %s", detectedIntent, policyResult.Reason)
			o.saveSession(ctx, session)
			return o.formatter.FormatError(input.SessionID, "Lo siento, esa acción no está permitida: "+policyResult.Reason)
		}
	}

	// 4. EVALUADOR DE RUTA — ¿Directa o vía LLM?
	if confident && o.policyEngine.IsDirectResolvable(detectedIntent) {
		return o.handleDirectRoute(ctx, input, session, detectedIntent)
	}

	// 5. RUTA VÍA LLM — Consultar al modelo de IA
	return o.handleLLMRoute(ctx, input, session, detectedIntent)
}

// handleDirectRoute resuelve la intención directamente llamando a la herramienta
// correspondiente sin consumir tokens del LLM.
func (o *Orchestrator) handleDirectRoute(ctx context.Context, input ChatInput, session *models.SessionContext, detectedIntent models.IntentType) response.APIResponse {
	toolName := intent.MapIntentToTool(detectedIntent)
	if toolName == "" {
		// Fallback a ruta LLM si no hay herramienta mapeada
		return o.handleLLMRoute(ctx, input, session, detectedIntent)
	}

	tool, exists := o.toolRegistry.Get(toolName)
	if !exists {
		return o.handleLLMRoute(ctx, input, session, detectedIntent)
	}

	// Ejecutar herramienta directamente con el mensaje como argumento
	result, err := tool.Execute(ctx, fmt.Sprintf(`{"query": "%s"}`, input.Message))
	if err != nil {
		log.Printf("[orchestrator] error en ruta directa: %v", err)
		return o.handleLLMRoute(ctx, input, session, detectedIntent)
	}

	// Agregar respuesta al historial
	o.contextManager.AddAssistantMessage(ctx, session, result)
	o.saveSession(ctx, session)

	return o.formatter.FormatRaw(input.SessionID, string(detectedIntent), result)
}

// handleLLMRoute consulta al LLM con el contexto conversacional y herramientas disponibles.
// Implementa el loop de Tool Calling si el LLM solicita ejecutar herramientas.
func (o *Orchestrator) handleLLMRoute(ctx context.Context, input ChatInput, session *models.SessionContext, detectedIntent models.IntentType) response.APIResponse {
	// Construir mensajes con contexto filtrado por intención
	messages := o.contextManager.BuildContextMessages(session, systemPrompt, detectedIntent)

	req := models.ChatRequest{
		SessionID:   input.SessionID,
		Messages:    messages,
		Tools:       o.toolRegistry.Definitions(),
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	totalTokens := 0

	// Loop de Tool Calling (el LLM puede solicitar múltiples herramientas)
	for i := 0; i < maxToolCallIterations; i++ {
		resp, err := o.llm.Chat(ctx, req)
		if err != nil {
			log.Printf("[orchestrator] error en LLM: %v", err)
			o.saveSession(ctx, session)
			return o.formatter.FormatError(input.SessionID, "Error al comunicarse con el asistente de IA.")
		}

		if resp.Usage != nil {
			totalTokens += resp.Usage.TotalTokens
		}

		// Si no hay Tool Calls, tenemos la respuesta final
		if len(resp.ToolCalls) == 0 {
			o.contextManager.AddAssistantMessage(ctx, session, resp.Content)
			o.saveSession(ctx, session)
			return o.formatter.FormatNatural(input.SessionID, resp.Content, string(detectedIntent), totalTokens)
		}

		// Ejecutar las herramientas solicitadas por el LLM
		// Agregar el mensaje del asistente con los ToolCalls al request
		req.Messages = append(req.Messages, models.Message{
			Role:      models.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		toolResults, err := o.toolExecutor.ExecuteAll(ctx, resp.ToolCalls, input.SessionID)
		if err != nil {
			log.Printf("[orchestrator] error ejecutando tools: %v", err)
			o.saveSession(ctx, session)
			return o.formatter.FormatError(input.SessionID, "Error al ejecutar las operaciones solicitadas.")
		}

		// Agregar resultados de herramientas al request para la siguiente iteración
		req.Messages = append(req.Messages, toolResults...)
	}

	// Si llegamos al límite de iteraciones, retornar lo que tengamos
	log.Printf("[orchestrator] límite de iteraciones de Tool Calling alcanzado para sesión %s", input.SessionID)
	o.saveSession(ctx, session)
	return o.formatter.FormatError(input.SessionID, "Se alcanzó el límite de procesamiento. Por favor, intenta con una consulta más simple.")
}

// saveSession persiste la sesión de forma segura (loguea el error sin propagarlo).
func (o *Orchestrator) saveSession(ctx context.Context, session *models.SessionContext) {
	if err := o.contextManager.Save(ctx, session); err != nil {
		log.Printf("[orchestrator] error al guardar sesión %s: %v", session.SessionID, err)
	}
}
