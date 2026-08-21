package products

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	toolsinfra "github.com/coderoycc/ai-go-api/internal/infrastructure/tools"
)

// StockCheckTool implementa ports.Tool para consultar el stock de productos.
// Embebe BaseHTTPTool, lo que le permite heredar Execute automáticamente
// y garantiza la exclusión de campos definidos en ExcludedFields sin código repetido.
type StockCheckTool struct {
	toolsinfra.BaseHTTPTool
	baseURL string
}

// NewStockCheckTool crea una nueva instancia de la herramienta de verificación de stock.
func NewStockCheckTool(baseURL string, timeout time.Duration) *StockCheckTool {
	t := &StockCheckTool{baseURL: baseURL}
	t.BaseHTTPTool = toolsinfra.NewBaseHTTPTool(t, timeout)
	return t
}

func (t *StockCheckTool) Name() string {
	return "product_stock_check"
}

func (t *StockCheckTool) Description() string {
	return "Consulta el stock y disponibilidad de un producto en almacén o tienda por código, área, fecha de vencimiento y tienda."
}

func (t *StockCheckTool) Method() string {
	return "POST"
}

func (t *StockCheckTool) EndpointURL() string {
	return t.baseURL + "/productos/stock"
}

func (t *StockCheckTool) RequiredPermission() models.Permission {
	return models.PermissionRead
}

func (t *StockCheckTool) FallbackArgKey() string {
	return "codigo"
}

func (t *StockCheckTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codigo":            map[string]any{"type": "string", "description": "Código del producto."},
			"area":              map[string]any{"type": "string", "description": "Área de negocio."},
			"fecha_vencimiento": map[string]any{"type": "string", "description": "Fecha de vencimiento."},
			"tienda":            map[string]any{"type": "string", "description": "Tienda o almacén."},
		},
		"required": []string{"codigo"},
	}
}

func (t *StockCheckTool) ExcludedFields() []string {
	return []string{"debug_info", "internal_trace_id"}
}

