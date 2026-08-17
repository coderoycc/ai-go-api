# Cómo implementar un nuevo Endpoint (ej. Sales)

Gracias al `FeatureEngine`, añadir una nueva funcionalidad es sumamente sencillo. No necesitas crear nuevos Servicios ni Handlers.

### 1. Definir el grupo de herramientas
Si aún no existe, crea tu estructura de herramientas (ver `how-to-add-tool.md`) y asegúrate de tener una clase que implemente `ports.ToolGroup`.

### 2. Registrar en el Router
Abre `internal/api/router.go` y añade la nueva receta dentro del grupo v1 utilizando `featureEngine.Register`.

El `SystemPrompt` es crucial: define la personalidad y las limitaciones del asistente para ese dominio específico.

```go
// En SetupRouter:
featureEngine.Register(v1, engine.FeatureRecipe{
    Path: "/sales/chat", // Endpoint final: /api/v1/sales/chat
    
    // SystemPrompt altamente descriptivo para acotar el alcance del LLM
    SystemPrompt: `Eres un asistente experto en ventas. 
Tu rol es gestionar órdenes, consultar ventas por fecha y sucursal.
- Si el usuario no provee sucursal, pregúntala.
- No inventes datos. 
- Usa siempre las herramientas disponibles para obtener datos reales.`,
    
    // Inyecta el grupo de herramientas diseñado para este endpoint
    Tools: salesTools.NewSalesTool(cfg.SalesURL, cfg.Timeout).Tools(),
})
```

### 3. ¡Listo!
El `FeatureEngine` tomará automáticamente esta receta, configurará el orquestador con las herramientas y prompts específicos, y manejará toda la comunicación:
* Validación de JSON request.
* Gestión de contexto conversacional (Redis).
* Invocación del LLM.
* Ejecución de herramientas (aplicando políticas de seguridad).
* Formateo de respuestas estructuradas o naturales.
