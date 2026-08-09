package products

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// SearchProducts consulta la API de búsqueda de productos
// enviando los filtros como body JSON al endpoint POST /api/productos/buscar.
func (c *Client) SearchProducts(ctx context.Context, req ports.SearchProductsRequest) (*ports.ProductSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("products: error al serializar request: %w", err)
	}

	endpoint := c.baseURL + "/productos/buscar"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("products: error al crear request de búsqueda: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("products: error al buscar productos: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "buscar productos"); err != nil {
		return nil, err
	}

	var result ports.ProductSearchResponse
	if err := decodeJSON(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("products: error al decodificar respuesta de búsqueda: %w", err)
	}

	return &result, nil
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
