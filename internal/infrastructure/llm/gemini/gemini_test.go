package gemini

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	productsTools "github.com/coderoycc/ai-go-api/internal/infrastructure/tools/products"
)

func TestMapMessagesToGemini(t *testing.T) {
	messages := []models.Message{
		{Role: models.RoleSystem, Content: "Eres un asistente útil."},
		{Role: models.RoleUser, Content: "Hola, busca productos de tecnología."},
		{
			Role: models.RoleAssistant,
			ToolCalls: []models.ToolCall{
				{ID: "search_products", Name: "search_products", Arguments: `{"query":"tecnologia"}`},
			},
		},
		{
			Role:       models.RoleTool,
			ToolCallID: "search_products",
			Content:    `[{"id":"1","name":"Laptop"}]`,
		},
	}

	contents, systemInstruction := mapMessagesToGemini(messages)

	if systemInstruction == nil {
		t.Fatal("esperaba systemInstruction no nil")
	}
	if len(contents) != 3 {
		t.Fatalf("esperaba 3 contenidos, obtuve %d", len(contents))
	}
}

func TestMapToolsToGemini(t *testing.T) {
	tools := []models.ToolDefinition{
		{
			Name:        "search_products",
			Description: "Busca productos en el catálogo",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Término de búsqueda",
					},
				},
				"required": []interface{}{"query"},
			},
		},
	}

	geminiTools := mapToolsToGemini(tools)
	if len(geminiTools) != 1 {
		t.Fatalf("esperaba 1 tool, obtuve %d", len(geminiTools))
	}
	if len(geminiTools[0].FunctionDeclarations) != 1 {
		t.Fatalf("esperaba 1 function declaration")
	}
	if geminiTools[0].FunctionDeclarations[0].Name != "search_products" {
		t.Errorf("nombre incorrecto: %s", geminiTools[0].FunctionDeclarations[0].Name)
	}
}

// TestGemini_InteractiveInspection permite inspeccionar con total detalle
// qué entra (mensajes y tools) y qué responde Gemini (texto o tool calls).
// Puedes cambiar el mensaje del usuario para probar diferentes comandos.
// Ejecutar: GEMINI_API_KEY="tu-key" go test -v ./internal/infrastructure/llm/gemini -run TestGemini_InteractiveInspection
func TestGemini_InteractiveInspection(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Saltando prueba de inspección: GEMINI_API_KEY no está definida")
	}

	ctx := context.Background()
	geminiAdapter, err := NewAdapter(ctx, apiKey, "gemini-3.1-flash-lite")
	if err != nil {
		t.Fatalf("error creando adaptador Gemini: %v", err)
	}

	// 🛠️ PUEDES MODIFICAR ESTE MENSAJE PARA PROBAR DIFERENTES COMANDOS
	userMessage := "Quiero saber si existen productos gamers"

	messages := []models.Message{
		{
			Role:    models.RoleSystem,
			Content: "Eres un asistente de ventas inteligente. Usa las herramientas disponibles cuando necesites buscar productos.",
		},
		{
			Role:    models.RoleUser,
			Content: userMessage,
		},
	}

	// Cargar herramientas de productos directamente desde el grupo
	productTool := productsTools.NewProductTool("http://localhost", time.Second)
	toolDefs := make([]models.ToolDefinition, 0, len(productTool.Tools()))
	for _, tool := range productTool.Tools() {
		toolDefs = append(toolDefs, models.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}

	// Construir ChatRequest exacto que se envía al modelo
	req := models.ChatRequest{
		SessionID:   "inspection-session",
		Messages:    messages,
		Tools:       toolDefs,
		Temperature: 0.1,
	}

	// 📤 IMPRIMIR DETALLE DE ENTRADA AL MODELO (REQUEST)
	t.Logf("\n==================================================")
	t.Logf(" 📤 [ENTRADA AL MODELO GEMINI]")
	t.Logf("==================================================")
	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	t.Logf("%s\n", string(reqJSON))

	// Invocar Chat
	resp, err := geminiAdapter.Chat(ctx, req)
	if err != nil {
		t.Skipf("⚠️ Saltando prueba de inspección interactiva: %v", err)
	}

	// 📥 IMPRIMIR DETALLE DE SALIDA DEL MODELO (RESPONSE)
	t.Logf("\n==================================================")
	t.Logf(" 📥 [SALIDA / RESPUESTA DEL MODELO GEMINI]")
	t.Logf("==================================================")
	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	t.Logf("%s\n", string(respJSON))

	// Resumen interpretado para facilitar la lectura
	if len(resp.ToolCalls) > 0 {
		t.Logf("🛠️  DECISIÓN DE GEMINI: Tool Calling detectado")
		for i, tc := range resp.ToolCalls {
			t.Logf("   [%d] Tool Name : %s", i, tc.Name)
			t.Logf("       Arguments : %s", tc.Arguments)
		}
	} else {
		t.Logf("💬 DECISIÓN DE GEMINI: Respuesta en texto plano")
		t.Logf("   Contenido : %q", resp.Content)
	}
}
