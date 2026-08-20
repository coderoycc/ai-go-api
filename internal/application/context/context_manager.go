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
// Carga, actualiza y persiste el contexto de sesión e historial de mensajes.
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

// AddAssistantMessage agrega la respuesta de texto del asistente al historial.
// Usar solo cuando el LLM responde en lenguaje natural (sin ToolCalls).
func (m *Manager) AddAssistantMessage(ctx context.Context, session *models.SessionContext, content string) {
	session.AddMessage(models.Message{
		Role:    models.RoleAssistant,
		Content: content,
	}, maxHistory)
}

// AddAssistantToolCallMessage persiste el mensaje del asistente que contiene
// solicitudes de herramientas (ToolCalls). Este mensaje debe guardarse en el historial
// ANTES del resultado de la tool para que los LLMs reconstruyan el contexto correctamente.
func (m *Manager) AddAssistantToolCallMessage(ctx context.Context, session *models.SessionContext, content string, toolCalls []models.ToolCall) {
	session.AddMessage(models.Message{
		Role:      models.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}, maxHistory)
}

// AddToolResultMessage persiste el resultado de la ejecución de una herramienta
// con el rol correcto (RoleTool) y el ToolCallID que lo asocia a la solicitud del LLM.
// Esto permite que el LLM en la siguiente iteración conozca el resultado y tome decisiones.
func (m *Manager) AddToolResultMessage(ctx context.Context, session *models.SessionContext, toolCallID, toolName, result string) {
	session.AddMessage(models.Message{
		Role:       models.RoleTool,
		Content:    result,
		ToolCallID: toolCallID,
	}, maxHistory)
}

// UpdateLastTool actualiza el nombre de la última herramienta ejecutada en la sesión.
func (m *Manager) UpdateLastTool(session *models.SessionContext, toolName string) {
	session.LastTool = toolName
	session.UpdatedAt = time.Now()
}

// BuildContextMessages construye la lista de mensajes a enviar al LLM,
// incluyendo el system prompt base y el historial de conversación.
func (m *Manager) BuildContextMessages(session *models.SessionContext, systemPrompt string) []models.Message {
	messages := make([]models.Message, 0, len(session.History)+1)

	// 1. System prompt base
	messages = append(messages, models.Message{
		Role:    models.RoleSystem,
		Content: systemPrompt,
	})

	// 2. Historial de conversación (copia independiente para evitar aliasing del arreglo subyacente)
	historyCopy := make([]models.Message, len(session.History))
	copy(historyCopy, session.History)
	messages = append(messages, historyCopy...)

	return messages
}

// Delete elimina completamente una sesión de la memoria.
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	return m.memory.Delete(ctx, sessionID)
}
