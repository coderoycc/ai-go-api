package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Adapter implementa ports.LLM usando el SDK oficial de Anthropic para Claude.
// Encapsula la comunicación con la API de Claude, incluyendo el mapeo de
// Tool Calling (ToolUse/ToolResult) al formato de dominio.
type Adapter struct {
	client *anthropic.Client
	model  string
}

// NewAdapter crea un nuevo adaptador de Claude con la API key y modelo especificados.
func NewAdapter(apiKey, model string) *Adapter {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &Adapter{client: &client, model: model}
}

// Chat envía una solicitud conversacional a Claude y retorna la respuesta
// mapeada a los modelos de dominio. Soporta Tool Calling nativo de Anthropic.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	messages, systemBlocks := mapMessagesToClaude(req.Messages)

	params := anthropic.MessageNewParams{
		Model:    anthropic.Model(a.resolveModel(req.Model)),
		Messages: messages,
	}

	// Configurar tokens máximos
	maxTokens := int64(4096)
	if req.MaxTokens > 0 {
		maxTokens = int64(req.MaxTokens)
	}
	params.MaxTokens = maxTokens

	// Configurar system prompt
	if len(systemBlocks) > 0 {
		params.System = systemBlocks
	}

	// Configurar temperatura
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(req.Temperature)
	}

	// Configurar herramientas
	if len(req.Tools) > 0 {
		params.Tools = mapToolsToClaude(req.Tools)
	}

	message, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("claude: error en message creation: %w", err)
	}

	return mapClaudeResponse(message), nil
}

// resolveModel retorna el modelo solicitado o el configurado por defecto.
func (a *Adapter) resolveModel(requested string) string {
	if requested != "" {
		return requested
	}
	return a.model
}

// mapMessagesToClaude convierte mensajes del dominio al formato de Anthropic.
// Separa los mensajes de sistema (que van en un campo aparte en la API de Claude)
// de los mensajes de usuario/asistente/herramienta.
func mapMessagesToClaude(messages []models.Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam) {
	var claudeMessages []anthropic.MessageParam
	var systemBlocks []anthropic.TextBlockParam

	for _, msg := range messages {
		switch msg.Role {
		case models.RoleSystem:
			systemBlocks = append(systemBlocks, anthropic.TextBlockParam{
				Text: msg.Content,
			})

		case models.RoleUser:
			claudeMessages = append(claudeMessages,
				anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)),
			)

		case models.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			// Mapear ToolCalls a bloques ToolUse de Claude
			for _, tc := range msg.ToolCalls {
				var input map[string]interface{}
				_ = json.Unmarshal([]byte(tc.Arguments), &input)
				if input == nil {
					input = make(map[string]interface{})
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			if len(blocks) > 0 {
				claudeMessages = append(claudeMessages, anthropic.NewAssistantMessage(blocks...))
			}

		case models.RoleTool:
			// Mapear resultado de herramienta a ToolResult de Claude
			claudeMessages = append(claudeMessages,
				anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
				),
			)
		}
	}

	return claudeMessages, systemBlocks
}

// mapToolsToClaude convierte definiciones de herramientas del dominio al formato Anthropic.
func mapToolsToClaude(tools []models.ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, t := range tools {
		// Construir el esquema de input para Claude
		inputSchema := anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: t.Parameters["properties"],
		}

		if required, ok := t.Parameters["required"].([]interface{}); ok {
			for _, r := range required {
				if s, ok := r.(string); ok {
					inputSchema.Required = append(inputSchema.Required, s)
				}
			}
		}

		toolParam := anthropic.ToolParam{
			Name:        t.Name,
			Description: param.NewOpt(t.Description),
			InputSchema: inputSchema,
		}

		result = append(result, anthropic.ToolUnionParam{OfTool: &toolParam})
	}

	return result
}

// mapClaudeResponse convierte la respuesta de Claude a modelos de dominio.
func mapClaudeResponse(message *anthropic.Message) models.ChatResponse {
	response := models.ChatResponse{
		Model:     message.Model,
		CreatedAt: time.Now(),
		Usage: &models.TokenUsage{
			PromptTokens:     int(message.Usage.InputTokens),
			CompletionTokens: int(message.Usage.OutputTokens),
			TotalTokens:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
	}

	// Recorrer los bloques de contenido de la respuesta usando Type field
	for _, block := range message.Content {
		switch block.Type {
		case "text":
			response.Content += block.Text
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			response.ToolCalls = append(response.ToolCalls, models.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
		}
	}

	return response
}
