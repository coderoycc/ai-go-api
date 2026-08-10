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

// AdvancedSearchTool implementa ports.Tool para la búsqueda avanzada de productos.
type AdvancedSearchTool struct {
	baseURL    string
	httpClient *http.Client
}

// NewAdvancedSearchTool crea una nueva instancia de la tool de búsqueda avanzada.
func NewAdvancedSearchTool(baseURL string, timeout time.Duration) *AdvancedSearchTool {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &AdvancedSearchTool{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
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

// MapResponse parsea el body crudo, excluye campos no deseados y retorna un objeto estructurado.
func (t *AdvancedSearchTool) MapResponse(rawBody []byte) (any, error) {
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
		return nil, fmt.Errorf("advanced_search: error al parsear respuesta JSON de la API: %w", err)
	}

	for _, field := range t.ExcludedFields() {
		delete(data, field)
	}

	return data, nil
}

func (t *AdvancedSearchTool) Execute(ctx context.Context, arguments string) (any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(arguments), &raw); err != nil {
		raw = map[string]any{"nombre": strings.Trim(arguments, "\"")}
	}

	body, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("advanced_search: error serializando argumentos: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, t.Method(), t.EndpointURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("advanced_search: error creando request HTTP: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("advanced_search: error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("advanced_search: error leyendo respuesta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("advanced_search: error HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return t.MapResponse(respBody)
}
