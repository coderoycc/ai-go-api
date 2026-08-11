package context

import (
	"context"
	"fmt"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

const (
	// maxHistory es el número máximo de mensajes a mantener en el historial.
	maxHistory = 20
	// defaultSessionTTL es el TTL por defecto de una sesión en segundos (30 min).
	defaultSessionTTL = 1800
)

// Manager administra la memoria conversacional interactuando con ports.Memory.
// Carga, actualiza y persiste el contexto de sesión, inyectando solo las
// entidades relevantes según la intención detectada.
type Manager struct {
	memory ports.Memory
}

// NewManager crea un nuevo administrador de contexto con el backend de memoria dado.
func NewManager(memory ports.Memory) *Manager {
	return &Manager{memory: memory}
}

// LoadOrCreate carga el contexto de sesión existente o crea uno nuevo si no existe.
func (m *Manager) LoadOrCreate(ctx context.Context, sessionID string) (*models.SessionContext, error) {
	session, err := m.memory.Load(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("context_manager: error al cargar sesión: %w", err)
	}

	if session == nil {
		session = &models.SessionContext{
			SessionID: sessionID,
			Entities:  make(map[string]string),
			History:   make([]models.Message, 0),
			Metadata:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			TTL:       defaultSessionTTL,
		}
	}

	return session, nil
}

// Save persiste el contexto de sesión actualizado en la memoria.
func (m *Manager) Save(ctx context.Context, session *models.SessionContext) error {
	if err := m.memory.Save(ctx, session.SessionID, session); err != nil {
		return fmt.Errorf("context_manager: error al guardar sesión: %w", err)
	}
	return nil
}

// AddUserMessage agrega un mensaje del usuario al historial y persiste.
func (m *Manager) AddUserMessage(ctx context.Context, session *models.SessionContext, content string) {
	session.AddMessage(models.Message{
		Role:    models.RoleUser,
		Content: content,
	}, maxHistory)
}

// AddAssistantMessage agrega la respuesta del asistente al historial.
func (m *Manager) AddAssistantMessage(ctx context.Context, session *models.SessionContext, content string) {
	session.AddMessage(models.Message{
		Role:    models.RoleAssistant,
		Content: content,
	}, maxHistory)
}

// UpdateIntent actualiza la última intención detectada en la sesión.
func (m *Manager) UpdateIntent(session *models.SessionContext, intent models.IntentType) {
	session.LastIntent = intent
	session.UpdatedAt = time.Now()
}

// BuildContextMessages construye la lista de mensajes a enviar al LLM,
// incluyendo el system prompt, el contexto de entidades relevantes
// según la intención, y el historial de conversación.
func (m *Manager) BuildContextMessages(session *models.SessionContext, systemPrompt string, intent models.IntentType) []models.Message {
	messages := make([]models.Message, 0, len(session.History)+2)

	// 1. System prompt base
	fullSystem := systemPrompt

	// 2. Inyectar entidades relevantes según la intención
	entityContext := m.buildEntityContext(session, intent)
	if entityContext != "" {
		fullSystem += "\n\nContexto de la conversación actual:\n" + entityContext
	}

	messages = append(messages, models.Message{
		Role:    models.RoleSystem,
		Content: fullSystem,
	})

	// 3. Historial de conversación (copia independiente para evitar aliasing del arreglo subyacente)
	historyCopy := make([]models.Message, len(session.History))
	copy(historyCopy, session.History)
	messages = append(messages, historyCopy...)

	return messages
}

// buildEntityContext construye un string con las entidades relevantes para la intención.
// Solo inyecta al LLM las entidades que necesita según el tipo de operación.
func (m *Manager) buildEntityContext(session *models.SessionContext, intent models.IntentType) string {
	if len(session.Entities) == 0 {
		return ""
	}

	var relevant []string

	switch intent {
	case models.IntentCreateSale:
		// Para ventas necesita producto, cantidad y cliente
		relevant = []string{"product_id", "product_name", "quantity", "customer_name", "price"}
	case models.IntentGetProduct, models.IntentCheckStock:
		// Para consultas de producto solo necesita el ID/nombre
		relevant = []string{"product_id", "product_name"}
	case models.IntentGetSale, models.IntentCancelSale:
		// Para consultas de venta necesita el ID de venta
		relevant = []string{"sale_id", "customer_name"}
	case models.IntentSearchProduct:
		// Para búsquedas, el término y categoría
		relevant = []string{"search_query", "category"}
	default:
		// General: inyectar todas las entidades
		result := ""
		for k, v := range session.Entities {
			result += fmt.Sprintf("- %s: %s\n", k, v)
		}
		return result
	}

	result := ""
	for _, key := range relevant {
		if val := session.GetEntity(key); val != "" {
			result += fmt.Sprintf("- %s: %s\n", key, val)
		}
	}

	return result
}

// Delete elimina completamente una sesión de la memoria.
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	return m.memory.Delete(ctx, sessionID)
}
