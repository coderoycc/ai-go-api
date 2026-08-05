package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	openailib "github.com/sashabaranov/go-openai"
)

// Adapter implementa ports.LLM usando la API compatible con OpenAI de Ollama.
// Permite ejecutar modelos de IA localmente sin depender de servicios cloud,
// ideal para desarrollo, pruebas o entornos sin acceso a internet.
type Adapter struct {
	client *openailib.Client
	model  string
}

const (
	// defaultBaseURL es el endpoint base de Ollama con compatibilidad OpenAI.
	defaultBaseURL = "http://localhost:11434/v1"
	// defaultModel es el modelo local por defecto.
	defaultModel = "llama3.1"
)

// NewAdapter crea un nuevo adaptador de Ollama con el modelo y host especificados.
// No requiere API key ya que Ollama se ejecuta localmente.
func NewAdapter(model, baseURL string) *Adapter {
	if model == "" {
		model = defaultModel
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	config := openailib.DefaultConfig("ollama")
	config.BaseURL = baseURL

	return &Adapter{
		client: openailib.NewClientWithConfig(config),
		model:  model,
	}
}

// Chat envía una solicitud conversacional al modelo local de Ollama y retorna
// la respuesta mapeada a los modelos de dominio. Soporta Tool Calling en
// modelos compatibles (ej. llama3.1, mistral).
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	chatReq := openailib.ChatCompletionRequest{
		Model:    a.resolveModel(req.Model),
		Messages: mapMessages(req.Messages),
	}

	if req.Temperature > 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = mapTools(req.Tools)
	}

	resp, err := a.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("ollama: error en chat completion: %w", err)
	}

	return mapResponse(resp), nil
}

func (a *Adapter) resolveModel(requested string) string {
	if requested != "" {
		return requested
	}
	return a.model
}

// mapMessages convierte mensajes del dominio al formato OpenAI-compatible.
func mapMessages(messages []models.Message) []openailib.ChatCompletionMessage {
	result := make([]openailib.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		m := openailib.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
		if msg.ToolCallID != "" {
			m.ToolCallID = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]openailib.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				m.ToolCalls = append(m.ToolCalls, openailib.ToolCall{
					ID:   tc.ID,
					Type: openailib.ToolTypeFunction,
					Function: openailib.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
		}
		result = append(result, m)
	}
	return result
}

// mapTools convierte definiciones de herramientas del dominio al formato OpenAI-compatible.
func mapTools(tools []models.ToolDefinition) []openailib.Tool {
	result := make([]openailib.Tool, 0, len(tools))
	for _, t := range tools {
		params, _ := json.Marshal(t.Parameters)
		result = append(result, openailib.Tool{
			Type: openailib.ToolTypeFunction,
			Function: &openailib.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(params),
			},
		})
	}
	return result
}

// mapResponse convierte la respuesta OpenAI-compatible a modelos de dominio.
func mapResponse(resp openailib.ChatCompletionResponse) models.ChatResponse {
	result := models.ChatResponse{
		Model:     resp.Model,
		CreatedAt: time.Now(),
		Usage: &models.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Content = choice.Message.Content
		if len(choice.Message.ToolCalls) > 0 {
			result.ToolCalls = make([]models.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				result.ToolCalls = append(result.ToolCalls, models.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}
	}

	return result
}
