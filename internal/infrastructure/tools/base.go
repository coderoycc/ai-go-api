package toolsinfra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// HTTPToolDescriptor define los métodos adicionales requeridos por BaseHTTPTool
// para ejecutar la petición HTTP y aplicar el filtrado de campos de respuesta.
type HTTPToolDescriptor interface {
	ports.Tool
	Method() string
	EndpointURL() string
	ExcludedFields() []string
	FallbackArgKey() string
}

// BaseHTTPTool provee la implementación reutilizable de Execute para cualquier herramienta HTTP.
// Al embeber BaseHTTPTool en un struct de herramienta concreta (como StockCheckTool),
// la herramienta hereda la ejecución HTTP completa y el filtrado automático de ExcludedFields
// sin necesidad de implementar ni Execute ni MapResponse.
type BaseHTTPTool struct {
	tool       HTTPToolDescriptor
	httpClient *http.Client
}

// NewBaseHTTPTool inicializa BaseHTTPTool asociando la referencia a la herramienta concreta
// y configurando el cliente HTTP con el timeout indicado.
func NewBaseHTTPTool(tool HTTPToolDescriptor, timeout time.Duration) BaseHTTPTool {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return BaseHTTPTool{
		tool:       tool,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Execute realiza el flujo estructurado de ejecución HTTP:
//  1. Parsea los argumentos JSON (o usa FallbackArgKey si el LLM envió un string plano).
//  2. Realiza la petición HTTP al endpoint del microservicio.
//  3. Valida el código de estado de la respuesta.
//  4. Parsea el cuerpo JSON y elimina automáticamente los campos definidos en ExcludedFields.
func (b *BaseHTTPTool) Execute(ctx context.Context, arguments string) (any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(arguments), &raw); err != nil {
		key := b.tool.FallbackArgKey()
		if key == "" {
			key = "input"
		}
		raw = map[string]any{key: strings.Trim(arguments, "\"")}
	}

	body, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: error serializando argumentos: %w", b.tool.Name(), err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, b.tool.Method(), b.tool.EndpointURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: error creando request HTTP: %w", b.tool.Name(), err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: error en petición HTTP: %w", b.tool.Name(), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: error leyendo respuesta: %w", b.tool.Name(), err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: error HTTP %d: %s", b.tool.Name(), resp.StatusCode, string(respBody))
	}

	var data map[string]any
	if err := json.Unmarshal(respBody, &data); err != nil {
		var rawList []any
		if errList := json.Unmarshal(respBody, &rawList); errList == nil {
			cleanedList := make([]any, 0, len(rawList))
			excluded := b.tool.ExcludedFields()
			for _, item := range rawList {
				if itemMap, ok := item.(map[string]any); ok {
					for _, field := range excluded {
						delete(itemMap, field)
					}
					cleanedList = append(cleanedList, itemMap)
				} else {
					cleanedList = append(cleanedList, item)
				}
			}
			return cleanedList, nil
		}
		return nil, fmt.Errorf("%s: error al parsear respuesta JSON de la API: %w", b.tool.Name(), err)
	}

	for _, field := range b.tool.ExcludedFields() {
		delete(data, field)
	}

	return data, nil
}
