package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	openailib "github.com/sashabaranov/go-openai"
)

// Adapter implementa ports.LLM usando el SDK de OpenAI.
// Encapsula completamente la comunicación con la API de OpenAI,
// mapeando entre los modelos de dominio y el formato nativo del SDK.
type Adapter struct {
	client *openailib.Client
	model  string
}

// NewAdapter crea un nuevo adaptador de OpenAI con la API key y modelo especificados.
func NewAdapter(apiKey, model string) *Adapter {
	if model == "" {
		model = openailib.GPT4o
	}
	return &Adapter{
		client: openailib.NewClient(apiKey),
		model:  model,
	}
}

// Chat envía una solicitud conversacional a OpenAI y retorna la respuesta
// mapeada a los modelos de dominio. Soporta Tool Calling nativo.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	chatReq := openailib.ChatCompletionRequest{
		Model:    a.resolveModel(req.Model),
		Messages: mapMessagesToOpenAI(req.Messages),
	}

	if req.Temperature > 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = mapToolsToOpenAI(req.Tools)
	}

	resp, err := a.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("openai: error en chat completion: %w", err)
	}

	return mapOpenAIResponse(resp), nil
}

// resolveModel retorna el modelo solicitado o el configurado por defecto.
func (a *Adapter) resolveModel(requested string) string {
	if requested != "" {
		return requested
	}
	return a.model
}

// mapMessagesToOpenAI convierte mensajes del dominio al formato del SDK de OpenAI.
func mapMessagesToOpenAI(messages []models.Message) []openailib.ChatCompletionMessage {
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

// mapToolsToOpenAI convierte definiciones de herramientas del dominio al formato OpenAI.
func mapToolsToOpenAI(tools []models.ToolDefinition) []openailib.Tool {
	result := make([]openailib.Tool, 0, len(tools))
	for _, t := range tools {
		// Serializar los parámetros como JSON para FunctionDefinition.Parameters
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

// mapOpenAIResponse convierte la respuesta del SDK de OpenAI a modelos de dominio.
func mapOpenAIResponse(resp openailib.ChatCompletionResponse) models.ChatResponse {
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
