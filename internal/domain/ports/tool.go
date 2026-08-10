package ports

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Tool define el contrato DDD obligatorio para cualquier API convertida en herramienta.
type Tool interface {
	// Name retorna el identificador único de la herramienta.
	Name() string

	// Description retorna una descripción legible de lo que hace la herramienta.
	Description() string

	// Method retorna el método HTTP que utiliza la herramienta (ej. POST, GET).
	Method() string

	// EndpointURL retorna la URL completa del endpoint de la API externa.
	EndpointURL() string

	// Parameters retorna el esquema JSON de los parámetros que acepta la herramienta.
	Parameters() map[string]any

	// ResponseSchema retorna el esquema JSON de la respuesta que devuelve la API externa.
	ResponseSchema() map[string]any

	// ExcludedFields retorna la lista de campos que deben ser excluidos del JSON de respuesta de la API.
	ExcludedFields() []string

	// MapResponse toma el cuerpo crudo de la respuesta HTTP, lo parsea, aplica exclusiones
	// y retorna un objeto estructurado (map/slice) listo para ser usado por el sistema o frontend.
	MapResponse(rawBody []byte) (any, error)

	// Execute ejecuta la herramienta con los argumentos proporcionados (JSON string).
	// Retorna un objeto estructurado (any) con los datos de la respuesta.
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
