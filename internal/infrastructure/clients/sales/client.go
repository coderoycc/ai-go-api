package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// Client implementa ports.SalesClient conectándose al microservicio de Ventas via HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient crea un nuevo cliente HTTP para el microservicio de Ventas.
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

// CreateSale registra una nueva venta en el microservicio
// llamando al endpoint POST /sales con el cuerpo JSON.
func (c *Client) CreateSale(ctx context.Context, req ports.CreateSaleRequest) (*ports.Sale, error) {
	endpoint := fmt.Sprintf("%s/sales", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("sales: error al serializar request de venta: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sales: error al crear request de venta: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sales: error al crear venta: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "crear venta"); err != nil {
		return nil, err
	}

	var sale ports.Sale
	if err := decodeJSON(resp.Body, &sale); err != nil {
		return nil, fmt.Errorf("sales: error al decodificar venta creada: %w", err)
	}

	return &sale, nil
}

// GetSaleByID consulta una venta existente por su ID
// llamando al endpoint GET /sales/{id} del microservicio.
func (c *Client) GetSaleByID(ctx context.Context, id string) (*ports.Sale, error) {
	endpoint := fmt.Sprintf("%s/sales/%s", c.baseURL, url.PathEscape(id))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("sales: error al crear request de consulta: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sales: error al consultar venta %s: %w", id, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "consultar venta"); err != nil {
		return nil, err
	}

	var sale ports.Sale
	if err := decodeJSON(resp.Body, &sale); err != nil {
		return nil, fmt.Errorf("sales: error al decodificar venta %s: %w", id, err)
	}

	return &sale, nil
}

// CancelSale cancela una venta existente por su ID
// llamando al endpoint PUT /sales/{id}/cancel del microservicio.
func (c *Client) CancelSale(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("%s/sales/%s/cancel", c.baseURL, url.PathEscape(id))

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, nil)
	if err != nil {
		return fmt.Errorf("sales: error al crear request de cancelación: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sales: error al cancelar venta %s: %w", id, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPError(resp, "cancelar venta"); err != nil {
		return err
	}

	return nil
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
		return fmt.Errorf("sales: recurso no encontrado al %s (404)", operation)
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("sales: no autorizado al %s (401)", operation)
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("sales: acceso denegado al %s (403)", operation)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("sales: conflicto al %s (409): %s", operation, string(body))
	case resp.StatusCode == http.StatusUnprocessableEntity:
		return fmt.Errorf("sales: datos inválidos al %s (422): %s", operation, string(body))
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return fmt.Errorf("sales: error del cliente al %s (%d): %s", operation, resp.StatusCode, string(body))
	case resp.StatusCode >= 500:
		return fmt.Errorf("sales: error del servidor al %s (%d): %s", operation, resp.StatusCode, string(body))
	default:
		return fmt.Errorf("sales: respuesta inesperada al %s (%d): %s", operation, resp.StatusCode, string(body))
	}
}

// decodeJSON decodifica el body de una respuesta HTTP a la estructura destino.
func decodeJSON(body io.Reader, dest interface{}) error {
	return json.NewDecoder(body).Decode(dest)
}
