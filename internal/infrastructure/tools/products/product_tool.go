package products

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// ProductTool agrupa todas las herramientas (endpoints) de la API de Productos.
type ProductTool struct {
	baseURL string
	timeout time.Duration
}

// NewProductTool crea una nueva instancia del grupo de herramientas de productos.
func NewProductTool(baseURL string, timeout time.Duration) *ProductTool {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &ProductTool{
		baseURL: baseURL,
		timeout: timeout,
	}
}

// Tools retorna todos los endpoints configurados para la API de productos.
func (p *ProductTool) Tools() []ports.Tool {
	return []ports.Tool{
		NewAdvancedSearchTool(p.baseURL, p.timeout),
		NewStockCheckTool(p.baseURL, p.timeout),
	}
}
