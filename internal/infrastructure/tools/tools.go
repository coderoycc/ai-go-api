package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// ProductSearchTool implementa ports.Tool para consultar la API externa.
type ProductSearchTool struct {
	client ports.ProductClient
}

func NewProductSearchTool(client ports.ProductClient) *ProductSearchTool {
	return &ProductSearchTool{client: client}
}

func (t *ProductSearchTool) Name() string {
	return "search_products"
}

func (t *ProductSearchTool) Description() string {
	return "Busca productos en el catálogo mediante la API externa. Permite filtrar por código, nombre, categorías, marcas, tipo, etiquetas, palabras clave y rango de precios. Usa esta herramienta cuando el usuario pregunte por productos disponibles o quiera buscar/buscar algo en el catálogo."
}

func (t *ProductSearchTool) Parameters() map[string]interface{} {
	strArray := func(desc string) map[string]interface{} {
		return map[string]interface{}{
			"type":        "array",
			"items":       map[string]interface{}{"type": "string"},
			"description": desc,
		}
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"codigo":         map[string]interface{}{"type": "string", "description": "Código exacto del producto a buscar."},
			"nombre":         map[string]interface{}{"type": "string", "description": "Término de búsqueda por nombre del producto."},
			"palabras_clave": strArray("Palabras clave que describen el producto."),
			"etiquetas":      strArray("Etiquetas del producto."),
			"marca":          strArray("Marcas de los productos a filtrar."),
			"tipo":           strArray("Tipos de producto (oferta, liquidacion, nuevo, etc.)."),
			"categoria":      strArray("Categorías de los productos a filtrar."),
			"precio_min":     map[string]interface{}{"type": "number", "description": "Precio mínimo."},
			"precio_max":     map[string]interface{}{"type": "number", "description": "Precio máximo."},
			"orden":          map[string]interface{}{"type": "string", "enum": []string{"asc", "desc"}, "description": "Orden de los resultados por precio."},
			"pagina":         map[string]interface{}{"type": "integer", "description": "Número de página de resultados."},
			"limite":         map[string]interface{}{"type": "integer", "description": "Cantidad de resultados por página."},
		},
		"required": []string{},
	}
}

func (t *ProductSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	var req ports.SearchProductsRequest
	if err := json.Unmarshal([]byte(arguments), &req); err != nil {
		return "", fmt.Errorf("argumentos inválidos para search_products: %w", err)
	}

	result, err := t.client.SearchProducts(ctx, req)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
