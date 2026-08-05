package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"google.golang.org/genai"
)

// Adapter implementa ports.LLM usando el SDK oficial de Google GenAI para Gemini.
// Encapsula la comunicación con la API de Gemini, incluyendo el mapeo de
// Tool Calling (FunctionDeclaration/FunctionCall) al formato de dominio.
type Adapter struct {
	client *genai.Client
	model  string
}

// NewAdapter crea un nuevo adaptador de Gemini con la API key y modelo especificados.
// Establece la conexión con la API de Gemini al momento de la creación.
func NewAdapter(ctx context.Context, apiKey, model string) (*Adapter, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: error al crear cliente: %w", err)
	}

	return &Adapter{client: client, model: model}, nil
}

// Chat envía una solicitud conversacional a Gemini y retorna la respuesta
// mapeada a los modelos de dominio. Soporta Tool Calling via FunctionDeclaration.
func (a *Adapter) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	contents, systemInstruction := mapMessagesToGemini(req.Messages)
	config := buildGeminiConfig(req, systemInstruction)

	if len(req.Tools) > 0 {
		config.Tools = mapToolsToGemini(req.Tools)
	}

	modelName := a.model
	if req.Model != "" {
		modelName = req.Model
	}

	result, err := a.client.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("gemini: error en generate content: %w", err)
	}

	return mapGeminiResponse(result, modelName), nil
}

// mapMessagesToGemini convierte mensajes del dominio al formato de contenido de Gemini.
// Separa el mensaje de sistema como SystemInstruction del resto de contenidos.
func mapMessagesToGemini(messages []models.Message) ([]*genai.Content, *genai.Content) {
	var contents []*genai.Content
	var systemInstruction *genai.Content

	for _, msg := range messages {
		switch msg.Role {
		case models.RoleSystem:
			systemInstruction = &genai.Content{
				Parts: []*genai.Part{{Text: msg.Content}},
				Role:  "user",
			}

		case models.RoleUser:
			contents = append(contents, &genai.Content{
				Parts: []*genai.Part{{Text: msg.Content}},
				Role:  "user",
			})

		case models.RoleAssistant:
			content := &genai.Content{Role: "model"}
			if msg.Content != "" {
				content.Parts = append(content.Parts, &genai.Part{Text: msg.Content})
			}
			// Mapear ToolCalls del asistente a FunctionCall de Gemini
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Arguments), &args)
				content.Parts = append(content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
			contents = append(contents, content)

		case models.RoleTool:
			// Mapear resultado de herramienta a FunctionResponse de Gemini
			var result map[string]any
			_ = json.Unmarshal([]byte(msg.Content), &result)
			if result == nil {
				result = map[string]any{"result": msg.Content}
			}
			contents = append(contents, &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							Name:     msg.ToolCallID,
							Response: result,
						},
					},
				},
			})
		}
	}

	return contents, systemInstruction
}

// buildGeminiConfig construye la configuración de generación de Gemini.
func buildGeminiConfig(req models.ChatRequest, systemInstruction *genai.Content) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if systemInstruction != nil {
		config.SystemInstruction = systemInstruction
	}

	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		config.Temperature = &temp
	}

	if req.MaxTokens > 0 {
		config.MaxOutputTokens = int32(req.MaxTokens)
	}

	return config
}

// mapToolsToGemini convierte definiciones de herramientas del dominio a Tools de Gemini.
func mapToolsToGemini(tools []models.ToolDefinition) []*genai.Tool {
	funcDecls := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, t := range tools {
		fd := &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
		}

		if t.Parameters != nil {
			fd.Parameters = mapToGeminiSchema(t.Parameters)
		}

		funcDecls = append(funcDecls, fd)
	}

	return []*genai.Tool{
		{FunctionDeclarations: funcDecls},
	}
}

// mapToGeminiSchema convierte un mapa de parámetros JSON Schema al Schema de Gemini.
func mapToGeminiSchema(params map[string]interface{}) *genai.Schema {
	schema := &genai.Schema{
		Type: genai.TypeObject,
	}

	if props, ok := params["properties"].(map[string]interface{}); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for key, val := range props {
			if propMap, ok := val.(map[string]interface{}); ok {
				propSchema := &genai.Schema{}
				if t, ok := propMap["type"].(string); ok {
					switch t {
					case "string":
						propSchema.Type = genai.TypeString
					case "integer":
						propSchema.Type = genai.TypeInteger
					case "number":
						propSchema.Type = genai.TypeNumber
					case "boolean":
						propSchema.Type = genai.TypeBoolean
					case "array":
						propSchema.Type = genai.TypeArray
					default:
						propSchema.Type = genai.TypeString
					}
				}
				if desc, ok := propMap["description"].(string); ok {
					propSchema.Description = desc
				}
				schema.Properties[key] = propSchema
			}
		}
	}

	if required, ok := params["required"].([]interface{}); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return schema
}

// mapGeminiResponse convierte la respuesta de Gemini a modelos de dominio.
func mapGeminiResponse(result *genai.GenerateContentResponse, modelName string) models.ChatResponse {
	response := models.ChatResponse{
		Model:     modelName,
		CreatedAt: time.Now(),
	}

	if result.UsageMetadata != nil {
		response.Usage = &models.TokenUsage{
			PromptTokens:     int(result.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(result.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(result.UsageMetadata.TotalTokenCount),
		}
	}

	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		for _, part := range result.Candidates[0].Content.Parts {
			if part.Text != "" {
				response.Content += part.Text
			}
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				response.ToolCalls = append(response.ToolCalls, models.ToolCall{
					ID:        part.FunctionCall.Name,
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				})
			}
		}
	}

	return response
}
