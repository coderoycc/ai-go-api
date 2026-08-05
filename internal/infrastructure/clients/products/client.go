package products

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// Client implementa ports.ProductClient conectándose al microservicio de Productos via HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient crea un nuevo cliente HTTP para el microservicio de Productos.
// Configura timeouts explícitos para evitar bloqueos indefinidos.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SearchProducts busca productos que coincidan con el término de búsqueda
// llamando al endpoint GET /products?q={query} del microservicio.
func (c *Client) SearchProducts(ctx context.Context, query string) ([]ports.Product, error) {
	endpoint := fmt.Sprintf("%s/products?q=%s", c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("products: error al crear request de búsqueda: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("products: error al buscar productos: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "buscar productos"); err != nil {
		return nil, err
	}

	var products []ports.Product
	if err := decodeJSON(resp.Body, &products); err != nil {
		return nil, fmt.Errorf("products: error al decodificar respuesta de búsqueda: %w", err)
	}

	return products, nil
}

// GetProductByID obtiene un producto específico por su ID
// llamando al endpoint GET /products/{id} del microservicio.
func (c *Client) GetProductByID(ctx context.Context, id string) (*ports.Product, error) {
	endpoint := fmt.Sprintf("%s/products/%s", c.baseURL, url.PathEscape(id))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("products: error al crear request por ID: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("products: error al obtener producto %s: %w", id, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "obtener producto"); err != nil {
		return nil, err
	}

	var product ports.Product
	if err := decodeJSON(resp.Body, &product); err != nil {
		return nil, fmt.Errorf("products: error al decodificar producto %s: %w", id, err)
	}

	return &product, nil
}

// CheckStock verifica la disponibilidad de stock de un producto
// llamando al endpoint GET /products/{id}/stock del microservicio.
func (c *Client) CheckStock(ctx context.Context, productID string) (int, error) {
	endpoint := fmt.Sprintf("%s/products/%s/stock", c.baseURL, url.PathEscape(productID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("products: error al crear request de stock: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("products: error al verificar stock de %s: %w", productID, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "verificar stock"); err != nil {
		return 0, err
	}

	var result struct {
		Stock int `json:"stock"`
	}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return 0, fmt.Errorf("products: error al decodificar stock de %s: %w", productID, err)
	}

	return result.Stock, nil
}

// checkHTTPError verifica el código de estado HTTP y retorna un error descriptivo
// si la respuesta indica un fallo (4xx o 5xx).
func checkHTTPError(resp *http.Response, operation string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("products: recurso no encontrado al %s (404)", operation)
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("products: no autorizado al %s (401)", operation)
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("products: acceso denegado al %s (403)", operation)
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return fmt.Errorf("products: error del cliente al %s (%d): %s", operation, resp.StatusCode, string(body))
	case resp.StatusCode >= 500:
		return fmt.Errorf("products: error del servidor al %s (%d): %s", operation, resp.StatusCode, string(body))
	default:
		return fmt.Errorf("products: respuesta inesperada al %s (%d): %s", operation, resp.StatusCode, string(body))
	}
}

// decodeJSON decodifica el body de una respuesta HTTP a la estructura destino.
func decodeJSON(body io.Reader, dest interface{}) error {
	return json.NewDecoder(body).Decode(dest)
}
