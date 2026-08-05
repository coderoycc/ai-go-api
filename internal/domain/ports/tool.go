package ports

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Tool define el contrato para una herramienta ejecutable por el orquestador.
// Cada herramienta se registra en el ToolRegistry y puede ser invocada
// directamente (ruta directa) o por el LLM (via Tool Calling).
type Tool interface {
	// Name retorna el identificador único de la herramienta.
	Name() string

	// Description retorna una descripción legible de lo que hace la herramienta,
	// utilizada por el LLM para decidir cuándo invocarla.
	Description() string

	// Parameters retorna el esquema JSON de los parámetros que acepta la herramienta.
	// Este esquema se envía al LLM como parte de la definición de herramientas disponibles.
	Parameters() map[string]interface{}

	// Execute ejecuta la herramienta con los argumentos proporcionados (JSON string).
	// Retorna el resultado como string (serializado) para inyectar de vuelta al LLM.
	Execute(ctx context.Context, arguments string) (string, error)
}

// ToolRegistry gestiona el registro y la búsqueda de herramientas disponibles
// para el orquestador. Permite registrar herramientas al inicio de la aplicación
// y buscarlas por nombre durante la ejecución.
type ToolRegistry interface {
	// Register registra una nueva herramienta en el registry.
	// Retorna error si ya existe una herramienta con el mismo nombre.
	Register(tool Tool) error

	// Get obtiene una herramienta por su nombre.
	// Retorna nil y false si la herramienta no está registrada.
	Get(name string) (Tool, bool)

	// List retorna todas las herramientas registradas.
	List() []Tool

	// Definitions retorna las definiciones de todas las herramientas registradas,
	// formateadas para enviar al LLM como herramientas disponibles.
	Definitions() []models.ToolDefinition
}
