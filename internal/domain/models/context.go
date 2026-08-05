package models

import "time"

// SessionContext mantiene el estado conversacional y las entidades clave
// de una sesión activa. Se persiste en memoria (Redis) y se inyecta al LLM
// únicamente con el contexto necesario según la intención detectada.
type SessionContext struct {
	// SessionID es el identificador único de la sesión conversacional.
	SessionID string `json:"session_id"`
	// Entities almacena pares clave-valor de entidades extraídas de la conversación
	// (ej. "product_id": "123", "customer_name": "Juan", "quantity": "5").
	Entities map[string]string `json:"entities,omitempty"`
	// History contiene los últimos mensajes de la conversación para mantener contexto.
	History []Message `json:"history,omitempty"`
	// LastIntent registra la última intención detectada en la conversación.
	LastIntent IntentType `json:"last_intent,omitempty"`
	// Metadata almacena información adicional arbitraria de la sesión.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// CreatedAt marca cuándo se creó la sesión.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt marca la última actualización de la sesión.
	UpdatedAt time.Time `json:"updated_at"`
	// TTL define el tiempo de vida de la sesión en segundos.
	TTL int `json:"ttl,omitempty"`
}

// SetEntity establece o actualiza una entidad en el contexto de sesión.
func (sc *SessionContext) SetEntity(key, value string) {
	if sc.Entities == nil {
		sc.Entities = make(map[string]string)
	}
	sc.Entities[key] = value
	sc.UpdatedAt = time.Now()
}

// GetEntity obtiene el valor de una entidad del contexto. Retorna cadena vacía si no existe.
func (sc *SessionContext) GetEntity(key string) string {
	if sc.Entities == nil {
		return ""
	}
	return sc.Entities[key]
}

// AddMessage agrega un mensaje al historial de la sesión, manteniendo
// un máximo de maxHistory mensajes para no saturar la ventana de contexto.
func (sc *SessionContext) AddMessage(msg Message, maxHistory int) {
	sc.History = append(sc.History, msg)
	if len(sc.History) > maxHistory {
		sc.History = sc.History[len(sc.History)-maxHistory:]
	}
	sc.UpdatedAt = time.Now()
}

// ClearEntities limpia todas las entidades almacenadas en la sesión.
func (sc *SessionContext) ClearEntities() {
	sc.Entities = make(map[string]string)
	sc.UpdatedAt = time.Now()
}
