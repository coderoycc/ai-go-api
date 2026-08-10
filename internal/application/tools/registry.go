package tools

import (
	"fmt"
	"sync"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// Registry implementa ports.ToolRegistry manteniendo un mapa thread-safe
// de herramientas disponibles para el orquestador.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ports.Tool
}

// NewRegistry crea un nuevo registro de herramientas vacío.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ports.Tool),
	}
}

// Register registra una nueva herramienta (ports.Tool) o un grupo de herramientas (ports.ToolGroup) en el registry.
// Retorna error si ya existe una herramienta con el mismo nombre o si el tipo no es soportado.
func (r *Registry) Register(item interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch t := item.(type) {
	case ports.Tool:
		if _, exists := r.tools[t.Name()]; exists {
			return fmt.Errorf("tool_registry: herramienta '%s' ya registrada", t.Name())
		}
		r.tools[t.Name()] = t
		return nil

	case ports.ToolGroup:
		for _, tool := range t.Tools() {
			if _, exists := r.tools[tool.Name()]; exists {
				return fmt.Errorf("tool_registry: herramienta '%s' ya registrada", tool.Name())
			}
			r.tools[tool.Name()] = tool
		}
		return nil

	default:
		return fmt.Errorf("tool_registry: tipo de ítem no soportado para registro (debe implementar ports.Tool o ports.ToolGroup)")
	}
}

// Get obtiene una herramienta por su nombre.
// Retorna nil y false si la herramienta no está registrada.
func (r *Registry) Get(name string) (ports.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List retorna todas las herramientas registradas.
func (r *Registry) List() []ports.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// Definitions retorna las definiciones de todas las herramientas registradas,
// formateadas para enviar al LLM como herramientas disponibles.
func (r *Registry) Definitions() []models.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]models.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, models.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return defs
}
