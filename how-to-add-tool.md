# Cómo implementar una nueva Tool

Para añadir una nueva funcionalidad (herramienta) que el LLM pueda invocar, sigue estos pasos.

### 1. Crear la estructura de la Tool
Crea un nuevo archivo en `internal/infrastructure/tools/<nombre_modulo>/<tool_name>.go`.

### 2. Implementar `ports.Tool`
Embebe `toolsinfra.BaseHTTPTool` para aprovechar la ejecución HTTP automática.

```go
type MyNewTool struct {
    toolsinfra.BaseHTTPTool
    baseURL string
}
```

### 3. Definición de Parámetros y Descripciones (Crítico para el LLM)
El LLM utiliza el esquema JSON devuelto por `Parameters()` para decidir cómo llamar a tu API.

*   **Descripción precisa:** No basta con decir "código". Usa: *"Código único del producto (ej: PROD-123). Necesario para buscar stock."*
*   **Enumeraciones:** Si un campo tiene valores fijos, usa `enum` en el JSON Schema.
*   **Ejemplos:** Si es posible, incluye descripciones que guíen al LLM sobre el formato.

```go
func (t *MyNewTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "sucursal": map[string]any{
                "type": "string", 
                "description": "Nombre o ID de la sucursal (ej. 'Calacoto', 'CAL-01'). Si el usuario provee un nombre, la tool lo resolverá internamente."
            },
            "fecha": map[string]any{
                "type": "string",
                "description": "Fecha en formato YYYY-MM-DD. Si no se especifica, asumir el mes actual."
            },
        },
        "required": []string{"sucursal"},
    }
}
```

### 4. Opción B: Tool Inteligente (Mapping Interno)
Si tu API técnica requiere IDs (ej. `sucursal_id=123`) pero el usuario usa lenguaje natural ("Calacoto"), **no obligues al LLM a saber el ID**. Implementa la lógica de mapeo dentro del método `Execute` de tu tool.

```go
func (t *MyNewTool) Execute(ctx context.Context, arguments string) (any, error) {
    // 1. Parsear argumentos crudos recibidos del LLM
    var args struct { Sucursal string `json:"sucursal"` }
    json.Unmarshal([]byte(arguments), &args)

    // 2. Mapeo inteligente (puedes consultar una caché, db local, o mapeo estático)
    sucursalID := resolveSucursalNameToID(args.Sucursal) 

    // 3. Preparar argumentos finales para la API real
    newArguments := fmt.Sprintf(`{"sucursal_id": "%s"}`, sucursalID)
    
    // 4. Llamar a la API real utilizando la lógica de BaseHTTPTool embebida
    return t.BaseHTTPTool.Execute(ctx, newArguments)
}
```

### 5. Incluirla en un `ToolGroup`
Asegúrate de que la herramienta sea retornada en el método `Tools()` de tu grupo de herramientas (`ToolGroup`).
```go
func (p *ProductTool) Tools() []ports.Tool {
    return []ports.Tool{
        NewMyNewTool(p.baseURL, p.timeout),
        // ... otras tools
    }
}
```
