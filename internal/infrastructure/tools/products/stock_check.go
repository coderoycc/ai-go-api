package products

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// StockCheckTool implementa ports.Tool para consultar el stock de productos.
type StockCheckTool struct {
	baseURL    string
	httpClient *http.Client
}

// NewStockCheckTool crea una nueva instancia de la tool de verificación de stock.
func NewStockCheckTool(baseURL string, timeout time.Duration) *StockCheckTool {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &StockCheckTool{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
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

func (t *StockCheckTool) ResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codigo":     map[string]any{"type": "string"},
			"stock":      map[string]any{"type": "integer"},
			"tienda":     map[string]any{"type": "string"},
			"disponible": map[string]any{"type": "boolean"},
		},
	}
}

func (t *StockCheckTool) ExcludedFields() []string {
	return []string{"debug_info", "internal_trace_id"}
}

// MapResponse parsea el body crudo, excluye campos no deseados y retorna un objeto estructurado.
func (t *StockCheckTool) MapResponse(rawBody []byte) (any, error) {
	var data map[string]any
	if err := json.Unmarshal(rawBody, &data); err != nil {
		var rawList []any
		if errList := json.Unmarshal(rawBody, &rawList); errList == nil {
			cleanedList := make([]any, 0, len(rawList))
			for _, item := range rawList {
				if itemMap, ok := item.(map[string]any); ok {
					for _, field := range t.ExcludedFields() {
						delete(itemMap, field)
					}
					cleanedList = append(cleanedList, itemMap)
				} else {
					cleanedList = append(cleanedList, item)
				}
			}
			return cleanedList, nil
		}
		return nil, fmt.Errorf("stock_check: error al parsear respuesta JSON de la API: %w", err)
	}

	for _, field := range t.ExcludedFields() {
		delete(data, field)
	}

	return data, nil
}

func (t *StockCheckTool) Execute(ctx context.Context, arguments string) (any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(arguments), &raw); err != nil {
		raw = map[string]any{"codigo": strings.Trim(arguments, "\"")}
	}

	body, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("stock_check: error serializando argumentos: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, t.Method(), t.EndpointURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("stock_check: error creando request HTTP: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stock_check: error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stock_check: error leyendo respuesta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stock_check: error HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return t.MapResponse(respBody)
}
