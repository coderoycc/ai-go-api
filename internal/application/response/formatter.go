package response

import (
	"encoding/json"
	"time"
)

// Mode define el modo de formateo de la respuesta.
type Mode string

const (
	// ModeNatural indica que la respuesta fue procesada por el LLM en lenguaje natural.
	ModeNatural Mode = "natural"
	// ModeRaw indica que la respuesta es JSON directo del microservicio sin procesamiento LLM.
	ModeRaw Mode = "raw"
)

// APIResponse es la estructura estándar de respuesta que recibe el Frontend.
type APIResponse struct {
	// SessionID identifica la sesión conversacional.
	SessionID string `json:"session_id"`
	// Message es el contenido de la respuesta (texto natural o JSON).
	Message string `json:"message"`
	// Mode indica si la respuesta es "natural" (LLM) o "raw" (JSON directo).
	Mode Mode `json:"mode"`
	// Intent es la intención detectada que originó la respuesta.
	Intent string `json:"intent,omitempty"`
	// Data contiene datos estructurados del microservicio (productos, ventas, etc.).
	Data interface{} `json:"data,omitempty"`
	// TokensUsed indica cuántos tokens se consumieron (0 si fue ruta directa).
	TokensUsed int `json:"tokens_used,omitempty"`
	// Timestamp marca cuándo se generó la respuesta.
	Timestamp time.Time `json:"timestamp"`
}

// Formatter formatea la salida a la estructura esperada por el Frontend.
// Soporta modo Natural (respuesta del LLM) y Raw (JSON directo del microservicio).
type Formatter struct{}

// NewFormatter crea un nuevo formateador de respuestas.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatNatural formatea una respuesta conversacional generada por el LLM.
func (f *Formatter) FormatNatural(sessionID, message, intent string, tokensUsed int) APIResponse {
	return APIResponse{
		SessionID:  sessionID,
		Message:    message,
		Mode:       ModeNatural,
		Intent:     intent,
		TokensUsed: tokensUsed,
		Timestamp:  time.Now(),
	}
}

// FormatRaw formatea una respuesta JSON directa del microservicio sin pasar por el LLM.
// Recibe los datos crudos y los serializa como parte de la respuesta.
func (f *Formatter) FormatRaw(sessionID, intent string, data interface{}) APIResponse {
	// Serializar data a string legible para el campo Message
	dataJSON, _ := json.MarshalIndent(data, "", "  ")

	return APIResponse{
		SessionID:  sessionID,
		Message:    string(dataJSON),
		Mode:       ModeRaw,
		Intent:     intent,
		Data:       data,
		TokensUsed: 0,
		Timestamp:  time.Now(),
	}
}

// FormatError formatea una respuesta de error para el Frontend.
func (f *Formatter) FormatError(sessionID, errorMsg string) APIResponse {
	return APIResponse{
		SessionID: sessionID,
		Message:   errorMsg,
		Mode:      ModeNatural,
		Intent:    "error",
		Timestamp: time.Now(),
	}
}
