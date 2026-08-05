package ports

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// LLM define el contrato para cualquier proveedor de modelos de lenguaje.
// Los adaptadores (OpenAI, Gemini, Claude, DeepSeek, Ollama) implementan
// esta interfaz, permitiendo cambiar de proveedor sin modificar la lógica central.
type LLM interface {
	// Chat envía una solicitud conversacional al modelo de lenguaje y retorna
	// la respuesta generada. El LLM puede responder con texto plano o con
	// solicitudes de ToolCalling que el orquestador deberá ejecutar.
	Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error)
}
