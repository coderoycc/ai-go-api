# Módulo LLM — Adaptadores Multi-Proveedor

Adaptadores que implementan la interfaz `ports.LLM`, permitiendo intercambiar proveedores de IA sin modificar la lógica del orquestador.

```go
type LLM interface {
    Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error)
}
```

---

## Proveedores Disponibles

### OpenAI
```go
import llmOpenAI "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/openai"

adapter := llmOpenAI.NewAdapter(apiKey, model)
// model: "gpt-4o", "gpt-4o-mini", "gpt-4-turbo", etc. (default: "gpt-4o")
```

| Variable de entorno | Descripción |
|---------------------|-------------|
| `OPENAI_API_KEY` | API key de OpenAI |

### Gemini
```go
import llmGemini "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/gemini"

adapter, err := llmGemini.NewAdapter(ctx, apiKey, model)
// model: "gemini-2.5-flash", "gemini-2.5-pro", etc. (default: "gemini-2.5-flash")
// Nota: requiere context en la creación del cliente.
```

| Variable de entorno | Descripción |
|---------------------|-------------|
| `GEMINI_API_KEY` | API key de Google AI Studio |

### Claude
```go
import llmClaude "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/claude"

adapter := llmClaude.NewAdapter(apiKey, model)
// model: "claude-sonnet-4-20250514", "claude-opus-4-20250514", etc.
```

| Variable de entorno | Descripción |
|---------------------|-------------|
| `ANTHROPIC_API_KEY` | API key de Anthropic |

### DeepSeek
```go
import llmDeepSeek "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/deepseek"

adapter := llmDeepSeek.NewAdapter(apiKey, model, baseURL)
// model: "deepseek-chat", "deepseek-coder" (default: "deepseek-chat")
// baseURL: "" usa "https://api.deepseek.com" por defecto
```

| Variable de entorno | Descripción |
|---------------------|-------------|
| `DEEPSEEK_API_KEY` | API key de DeepSeek |
| `DEEPSEEK_BASE_URL` | URL base (opcional, para instancias privadas) |

### Ollama (Local)
```go
import llmOllama "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/ollama"

adapter := llmOllama.NewAdapter(model, baseURL)
// model: "llama3.1", "mistral", "codellama", etc. (default: "llama3.1")
// baseURL: "" usa "http://localhost:11434/v1" por defecto
// No requiere API key.
```

| Requisito | Descripción |
|-----------|-------------|
| Ollama instalado | `ollama serve` corriendo en la máquina local |
| Modelo descargado | `ollama pull llama3.1` antes de usar |

---

## Uso General

Todos los adaptadores se usan exactamente igual gracias a la interfaz `ports.LLM`:

```go
// 1. Crear el adaptador (cualquier proveedor)
var llm ports.LLM = llmOpenAI.NewAdapter(os.Getenv("OPENAI_API_KEY"), "gpt-4o")

// 2. Construir la petición
req := models.ChatRequest{
    SessionID: "session-123",
    Messages: []models.Message{
        {Role: models.RoleSystem, Content: "Eres un asistente de ventas."},
        {Role: models.RoleUser, Content: "Busca laptops disponibles"},
    },
    Tools:       toolRegistry.Definitions(), // herramientas disponibles
    Temperature: 0.7,
    MaxTokens:   1024,
}

// 3. Invocar
resp, err := llm.Chat(ctx, req)

// 4. Evaluar respuesta
if len(resp.ToolCalls) > 0 {
    // El LLM quiere ejecutar herramientas → pasar al Executor
} else {
    // Respuesta de texto directo → resp.Content
}
```

## Cambiar de Proveedor

Solo cambia la línea de creación del adaptador:

```go
// De OpenAI...
var llm ports.LLM = llmOpenAI.NewAdapter(apiKey, "gpt-4o")

// ...a Gemini
var llm ports.LLM = llmGemini.NewAdapter(ctx, apiKey, "gemini-2.5-flash")

// ...a Claude
var llm ports.LLM = llmClaude.NewAdapter(apiKey, "claude-sonnet-4-20250514")

// ...a DeepSeek
var llm ports.LLM = llmDeepSeek.NewAdapter(apiKey, "deepseek-chat", "")

// ...a Ollama (local, sin costo)
var llm ports.LLM = llmOllama.NewAdapter("llama3.1", "")
```

## Tool Calling

Cada adaptador mapea internamente las herramientas al formato nativo de su proveedor:

| Proveedor | Formato nativo |
|-----------|---------------|
| OpenAI | `Function Calling` (tools array) |
| Gemini | `FunctionDeclaration` / `FunctionCall` |
| Claude | `ToolUse` / `ToolResult` blocks |
| DeepSeek | Compatible con OpenAI Function Calling |
| Ollama | Compatible con OpenAI Function Calling |

## Dependencias

```
github.com/sashabaranov/go-openai    → OpenAI, DeepSeek, Ollama
google.golang.org/genai              → Gemini
github.com/anthropics/anthropic-sdk-go → Claude
```
