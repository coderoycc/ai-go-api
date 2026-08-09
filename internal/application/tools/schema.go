package tools

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	openailib "github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

// SchemaMapper se encarga de traducir la definición agnóstica de una herramienta
// (models.ToolDefinition) a las estructuras nativas requeridas por cada SDK de LLM.
type SchemaMapper struct{}

// NewSchemaMapper crea una nueva instancia de SchemaMapper.
func NewSchemaMapper() *SchemaMapper {
	return &SchemaMapper{}
}

// ToOpenAITool convierte una ToolDefinition al formato de herramientas de OpenAI / DeepSeek / Ollama.
func (m *SchemaMapper) ToOpenAITool(def models.ToolDefinition) openailib.Tool {
	paramsJSON, _ := json.Marshal(def.Parameters)

	return openailib.Tool{
		Type: openailib.ToolTypeFunction,
		Function: &openailib.FunctionDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  json.RawMessage(paramsJSON),
		},
	}
}

// ToOpenAITools convierte un slice de ToolDefinition al formato de herramientas de OpenAI.
func (m *SchemaMapper) ToOpenAITools(defs []models.ToolDefinition) []openailib.Tool {
	result := make([]openailib.Tool, 0, len(defs))
	for _, d := range defs {
		result = append(result, m.ToOpenAITool(d))
	}
	return result
}

// ToAnthropicTool convierte una ToolDefinition al formato de herramientas de Claude (Anthropic).
func (m *SchemaMapper) ToAnthropicTool(def models.ToolDefinition) anthropic.ToolUnionParam {
	inputSchema := anthropic.ToolInputSchemaParam{
		Type:       "object",
		Properties: def.Parameters["properties"],
	}

	if required, ok := def.Parameters["required"].([]interface{}); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				inputSchema.Required = append(inputSchema.Required, s)
			}
		}
	} else if requiredStr, ok := def.Parameters["required"].([]string); ok {
		inputSchema.Required = requiredStr
	}

	toolParam := anthropic.ToolParam{
		Name:        def.Name,
		Description: param.NewOpt(def.Description),
		InputSchema: inputSchema,
	}

	return anthropic.ToolUnionParam{OfTool: &toolParam}
}

// ToAnthropicTools convierte un slice de ToolDefinition al formato de herramientas de Claude.
func (m *SchemaMapper) ToAnthropicTools(defs []models.ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(defs))
	for _, d := range defs {
		result = append(result, m.ToAnthropicTool(d))
	}
	return result
}

// ToGeminiFunctionDeclaration convierte una ToolDefinition al FunctionDeclaration de Gemini.
func (m *SchemaMapper) ToGeminiFunctionDeclaration(def models.ToolDefinition) *genai.FunctionDeclaration {
	fd := &genai.FunctionDeclaration{
		Name:        def.Name,
		Description: def.Description,
	}

	if def.Parameters != nil {
		fd.Parameters = m.mapToGeminiSchema(def.Parameters)
	}

	return fd
}

// ToGeminiTool envuelve las FunctionDeclarations en la estructura Tool de Gemini.
func (m *SchemaMapper) ToGeminiTool(defs []models.ToolDefinition) []*genai.Tool {
	funcDecls := make([]*genai.FunctionDeclaration, 0, len(defs))
	for _, d := range defs {
		funcDecls = append(funcDecls, m.ToGeminiFunctionDeclaration(d))
	}

	return []*genai.Tool{
		{FunctionDeclarations: funcDecls},
	}
}

// mapToGeminiSchema traduce un esquema de parámetros JSON Schema al genai.Schema de Gemini.
func (m *SchemaMapper) mapToGeminiSchema(params map[string]interface{}) *genai.Schema {
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
						if itemsMap, ok := propMap["items"].(map[string]interface{}); ok {
							if itemType, ok := itemsMap["type"].(string); ok {
								itemSchema := &genai.Schema{}
								switch itemType {
								case "string":
									itemSchema.Type = genai.TypeString
								case "integer":
									itemSchema.Type = genai.TypeInteger
								case "number":
									itemSchema.Type = genai.TypeNumber
								case "boolean":
									itemSchema.Type = genai.TypeBoolean
								default:
									itemSchema.Type = genai.TypeString
								}
								propSchema.Items = itemSchema
							}
						} else {
							propSchema.Items = &genai.Schema{Type: genai.TypeString}
						}
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
	} else if requiredStr, ok := params["required"].([]string); ok {
		schema.Required = requiredStr
	}

	return schema
}
