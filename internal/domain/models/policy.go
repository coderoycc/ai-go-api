package models

// PolicyResult representa el resultado de evaluar una política de seguridad
// sobre una intención o llamada a herramienta.
type PolicyResult struct {
	// Allowed indica si la acción está permitida por el motor de políticas.
	Allowed bool `json:"allowed"`
	// Reason explica por qué la acción fue permitida o denegada.
	Reason string `json:"reason,omitempty"`
	// PolicyName identifica la política que generó esta decisión.
	PolicyName string `json:"policy_name,omitempty"`
}

// PolicyRule define una regla individual del motor de políticas.
type PolicyRule struct {
	// Name es el identificador único de la regla.
	Name string `json:"name"`
	// Description explica el propósito de la regla.
	Description string `json:"description,omitempty"`
	// AllowedIntents lista las intenciones permitidas por esta regla.
	AllowedIntents []IntentType `json:"allowed_intents,omitempty"`
	// DeniedIntents lista las intenciones explícitamente denegadas.
	DeniedIntents []IntentType `json:"denied_intents,omitempty"`
	// AllowedTools lista los nombres de herramientas permitidas.
	AllowedTools []string `json:"allowed_tools,omitempty"`
	// DeniedTools lista los nombres de herramientas explícitamente denegadas.
	DeniedTools []string `json:"denied_tools,omitempty"`
	// RequiresAuth indica si la acción requiere que el usuario esté autenticado.
	RequiresAuth bool `json:"requires_auth,omitempty"`
	// MaxCallsPerMinute limita la cantidad de invocaciones por minuto (0 = sin límite).
	MaxCallsPerMinute int `json:"max_calls_per_minute,omitempty"`
}

// PolicyEvalRequest encapsula los datos necesarios para evaluar una política.
type PolicyEvalRequest struct {
	// SessionID identifica la sesión que solicita la evaluación.
	SessionID string `json:"session_id"`
	// Intent es la intención detectada que se quiere evaluar.
	Intent IntentType `json:"intent"`
	// ToolName es el nombre de la herramienta que se quiere ejecutar (si aplica).
	ToolName string `json:"tool_name,omitempty"`
	// UserRole es el rol del usuario que realiza la petición.
	UserRole string `json:"user_role,omitempty"`
	// Authenticated indica si el usuario está autenticado.
	Authenticated bool `json:"authenticated"`
}
