package models

import "time"

// Role representa el rol del emisor de un mensaje en la conversación.
type Role string

const (
	// RoleUser indica un mensaje enviado por el usuario final.
	RoleUser Role = "user"
	// RoleSystem indica un mensaje de configuración/sistema para el LLM.
	RoleSystem Role = "system"
	// RoleAssistant indica un mensaje generado por el asistente (LLM).
	RoleAssistant Role = "assistant"
	// RoleTool indica un mensaje que es el resultado de la ejecución de una herramienta.
	RoleTool Role = "tool"
)


// Message representa un mensaje individual dentro de la conversación.
type Message struct {
	// Role indica quién emitió el mensaje (user, system, assistant, tool).
	Role Role `json:"role"`
	// Content es el contenido textual del mensaje.
	Content string `json:"content"`
	// ToolCallID identifica la llamada a herramienta asociada (solo para Role=tool).
	ToolCallID string `json:"tool_call_id,omitempty"`
	// ToolCalls contiene las llamadas a herramientas solicitadas por el LLM (solo para Role=assistant).
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall representa una solicitud del LLM para ejecutar una herramienta externa.
type ToolCall struct {
	// ID es el identificador único de la llamada a herramienta asignado por el LLM.
	ID string `json:"id"`
	// Name es el nombre de la herramienta a invocar (ej. "search_products").
	Name string `json:"name"`
	// Arguments contiene los argumentos serializados como JSON string para la herramienta.
	Arguments string `json:"arguments"`
	// ThoughtSignature almacena la firma de razonamiento opcional requerida por Gemini en tool calling.
	ThoughtSignature []byte `json:"thought_signature,omitempty"`
}

// ChatRequest encapsula la petición que se envía al puerto LLM.
type ChatRequest struct {
	// SessionID identifica la sesión conversacional del usuario.
	SessionID string `json:"session_id"`
	// Messages es el historial de mensajes a enviar al LLM.
	Messages []Message `json:"messages"`
	// Tools define las herramientas disponibles para que el LLM pueda invocar.
	Tools []ToolDefinition `json:"tools,omitempty"`
	// Model permite especificar un modelo particular (opcional, usa el configurado por defecto).
	Model string `json:"model,omitempty"`
	// Temperature controla la creatividad de la respuesta (0.0 a 1.0).
	Temperature float64 `json:"temperature,omitempty"`
	// MaxTokens limita la longitud máxima de la respuesta del LLM.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// ToolDefinition describe una herramienta disponible para el LLM en formato estándar.
type ToolDefinition struct {
	// Name es el identificador de la herramienta.
	Name string `json:"name"`
	// Description explica al LLM cuándo y cómo usar esta herramienta.
	Description string `json:"description"`
	// Parameters define el esquema JSON de los parámetros que acepta la herramienta.
	Parameters map[string]interface{} `json:"parameters"`
}

// ChatResponse encapsula la respuesta obtenida del puerto LLM.
type ChatResponse struct {
	// Content es el texto de respuesta generado por el LLM (puede estar vacío si hay ToolCalls).
	Content string `json:"content"`
	// ToolCalls contiene las herramientas que el LLM solicita ejecutar.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// Model indica el modelo que procesó la petición.
	Model string `json:"model,omitempty"`
	// Usage contiene métricas de consumo de tokens.
	Usage *TokenUsage `json:"usage,omitempty"`
	// CreatedAt marca el momento de generación de la respuesta.
	CreatedAt time.Time `json:"created_at"`
}

// TokenUsage representa las métricas de consumo de tokens del LLM.
type TokenUsage struct {
	// PromptTokens cantidad de tokens consumidos por el prompt de entrada.
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens cantidad de tokens generados en la respuesta.
	CompletionTokens int `json:"completion_tokens"`
	// TotalTokens suma total de tokens consumidos.
	TotalTokens int `json:"total_tokens"`
}
