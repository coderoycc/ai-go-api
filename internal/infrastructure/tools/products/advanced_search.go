package products

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	toolsinfra "github.com/coderoycc/ai-go-api/internal/infrastructure/tools"
)

// AdvancedSearchTool implementa ports.Tool para la búsqueda avanzada de productos.
// Embebe BaseHTTPTool, lo que le permite heredar Execute automáticamente
// y garantiza la exclusión de campos definidos en ExcludedFields sin código repetido.
type AdvancedSearchTool struct {
	toolsinfra.BaseHTTPTool
	baseURL string
}

// NewAdvancedSearchTool crea una nueva instancia de la herramienta de búsqueda avanzada.
func NewAdvancedSearchTool(baseURL string, timeout time.Duration) *AdvancedSearchTool {
	t := &AdvancedSearchTool{baseURL: baseURL}
	t.BaseHTTPTool = toolsinfra.NewBaseHTTPTool(t, timeout)
	return t
}

func (t *AdvancedSearchTool) Name() string {
	return "product_advanced_search"
}

func (t *AdvancedSearchTool) Description() string {
	return "Busca productos de forma avanzada en el catálogo permitiendo filtrar por código, nombre, categoría, área, precio, marcas, tipos, etiquetas y palabras clave. Usa esta herramienta cuando el usuario quiera buscar productos en el catálogo."
}

func (t *AdvancedSearchTool) Method() string {
	return "POST"
}

func (t *AdvancedSearchTool) EndpointURL() string {
	return t.baseURL + "/productos/buscar"
}

func (t *AdvancedSearchTool) RequiredPermission() models.Permission {
	return models.PermissionRead
}

func (t *AdvancedSearchTool) FallbackArgKey() string {
	return "nombre"
}

func (t *AdvancedSearchTool) Parameters() map[string]any {
	strArray := func(desc string) map[string]any {
		return map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": desc,
		}
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codigo":         map[string]any{"type": "string", "description": "Código exacto del producto."},
			"nombre":         map[string]any{"type": "string", "description": "Nombre o término de búsqueda."},
			"palabras_clave": strArray("Palabras clave."),
			"etiquetas":      strArray("Etiquetas."),
			"marca":          map[string]any{"type": "string", "description": "Marca."},
			"tipo":           map[string]any{"type": "string", "description": "Tipo de producto."},
			"categoria":      map[string]any{"type": "string", "description": "Categoría."},
			"precio_min":     map[string]any{"type": "number", "description": "Precio mínimo."},
			"precio_max":     map[string]any{"type": "number", "description": "Precio máximo."},
			"orden":          map[string]any{"type": "string", "enum": []string{"asc", "desc"}, "description": "Orden por precio."},
			"pagina":         map[string]any{"type": "integer", "description": "Número de página."},
			"limite":         map[string]any{"type": "integer", "description": "Límite por página."},
		},
		"required": []string{},
	}
}

func (t *AdvancedSearchTool) ResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"total":             map[string]any{"type": "integer", "description": "Total de productos encontrados."},
			"pagina":            map[string]any{"type": "integer", "description": "Página actual."},
			"limite":            map[string]any{"type": "integer", "description": "Límite de resultados."},
			"filtros_aplicados": map[string]any{"type": "object", "description": "Filtros aplicados."},
			"productos":         map[string]any{"type": "array", "description": "Lista de productos."},
		},
	}
}

func (t *AdvancedSearchTool) ExcludedFields() []string {
	return []string{"limite", "pagina", "filtros_aplicados"}
}
