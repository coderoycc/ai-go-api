package ports

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Tool define el contrato de dominio para cualquier capacidad o herramienta disponible para el motor de IA.
type Tool interface {
	// Name retorna el identificador único de la herramienta.
	Name() string

	// Description retorna una descripción legible de lo que hace la herramienta para el LLM.
	Description() string

	// Parameters retorna el esquema JSON de los parámetros que acepta la herramienta.
	Parameters() map[string]any

	// RequiredPermission declara el nivel de acceso mínimo que necesita el usuario
	// para poder ejecutar esta herramienta (models.PermissionRead o models.PermissionWrite).
	RequiredPermission() models.Permission

	// Execute ejecuta la herramienta con los argumentos proporcionados (JSON string).
	Execute(ctx context.Context, arguments string) (any, error)
}

// ToolGroup agrupa múltiples endpoints/herramientas pertenecientes a un mismo dominio de API.
type ToolGroup interface {
	Tools() []Tool
}

// ToolRegistry gestiona el registro y la búsqueda de herramientas disponibles
// para el orquestador. Permite registrar tanto herramientas individuales (Tool)
// como grupos completos de herramientas (ToolGroup).
type ToolRegistry interface {
	// Register registra una nueva herramienta o grupo de herramientas en el registry.
	Register(item interface{}) error

	// Get obtiene una herramienta por su nombre.
	Get(name string) (Tool, bool)

	// List retorna todas las herramientas registradas.
	List() []Tool

	// Definitions retorna las definiciones de todas las herramientas registradas.
	Definitions() []models.ToolDefinition
}
