# Tool API — Búsqueda de Productos (`product_advanced_search`)

Documentación de las tools expuestas por el endpoint de productos en el AI
Orchestrator Engine. Estas tools exponen al LLM endpoints de la API externa para
que el modelo pueda consultar datos reales.

## Flujo de la tool

```
POST /api/v1/products/chat
   │
   ▼
ProductChatHandler (internal/api/handler.go) ──► ProductChatService (application/services)
   │                                                  │  Arma las tools de productos (products.ProductTool)
   │                                                  ▼
   └── Orchestrator (orchestrator.go)  ← recibe []ports.Tool explícito por endpoint
         │
         ├── Context Manager (context_manager.go)   → carga/crea sesión en Redis
         ├── Intent Detector (intent_detector.go)   → clasifica la intención (regex, sin LLM)
         ├── Policy Engine (policy_engine.go)       → valida el permiso de la tool elegida
         │
         └── Ejecución de la tool seleccionada ──► ProductTool.Tools() ──► TU API
```

El resultado de la API se devuelve como JSON en la respuesta (modo `raw` cuando va
por ruta directa, o se le pasa al LLM para que redacte en lenguaje natural).

## Configuración de la URL

La URL base se define en `.env` (variable `PRODUCTS_SERVICE_URL`). El path del
endpoint está hardcodeado en el cliente HTTP.

```
PRODUCTS_SERVICE_URL=https://tu-api.com
```

→ La petición real será: `POST https://tu-api.com/api/productos/buscar`

## Endpoint expuesto

**`POST {PRODUCTS_SERVICE_URL}/api/productos/buscar`**

### Body (todos los campos son opcionales y combinables)

```json
{
  "codigo": "P002",
  "nombre": "smartphone",
  "palabras_clave": ["bluetooth", "gamer"],
  "etiquetas": ["niños", "juguetes"],
  "marca": ["Samsung", "Sony"],
  "tipo": ["oferta", "liquidacion"],
  "categoria": ["Audio"],
  "precio_min": 100,
  "precio_max": 2000,
  "orden": "asc",
  "pagina": 1,
  "limite": 10
}
```

### Respuesta (wrapper)

```json
{
  "total": 16,
  "pagina": 1,
  "limite": 16,
  "filtros_aplicados": {},
  "productos": [
    {
      "codigo": "P006",
      "nombre": "Teclado Mecánico Redragon K552",
      "marca": "Redragon",
      "categoria": "Perifericos",
      "precio": 220,
      "tipo": "oferta",
      "stock": 20,
      "descripcion": "Teclado mecánico compacto retroiluminado",
      "etiquetas": ["tecnologia", "gaming"],
      "palabras_clave": ["teclado", "redragon"],
      "imagen": "https://via.placeholder.com/300?text=Redragon+K552"
    }
  ]
}
```

## Dónde se configura cada cosa (los 3 lugares)

Una tool se compone de 3 piezas que deben hablar el mismo idioma (mismos nombres
de campos JSON):

| #   | Archivo                                                            | Qué contiene                                                                                                        |
| --- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| 1   | `internal/domain/models` (y `ports.Tool`)                          | El contrato `ports.Tool` que toda tool debe implementar.                                                             |
| 2   | `internal/infrastructure/tools/base.go` (`BaseHTTPTool`)           | El **HTTP real**: método, path, headers y decode de la respuesta.                                                    |
| 3   | `internal/infrastructure/tools/products/*_tool.go`                 | Lo que **ve el LLM**: `Description()` y `Parameters()` (schema JSON) + `Name()` y `RequiredPermission()`.            |

**Regla clave:** los nombres JSON de los structs deben coincidir con los nombres
de `properties` en `Parameters()`, porque el LLM genera `{"campo": valor}` y ese
mismo JSON se envía como body a la API.

## Ensamblar el endpoint

Cada endpoint se arma explícitamente en un servicio de aplicación que agrupa sus
propias tools y las pasa al orquestador. Para productos:

`internal/application/services/product_chat_service.go`

```go
service := services.NewProductChatService(
    llmAdapter,
    contextManager,
    intentDetector,
    policyEngine,
    formatter,
    cfg.Clients.ProductsURL,
    cfg.Clients.Timeout,
)
```

- `ProductChatService` crea las tools de productos (`ProductTool.Tools()`) y se
  las pasa al orquestador en cada petición.
- La política de permisos la evalúa el `PolicyEngine` con la tool ya seleccionada
  por el LLM.

## Prueba del endpoint

```bash
curl -X POST http://localhost:8080/api/v1/products/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"s1","message":"buscar laptops"}'
```

## Cómo agregar otra API/tool

1. Implementa `ports.Tool` (`Name`, `Description`, `Parameters`, `RequiredPermission`, `Execute`) en `internal/infrastructure/tools/`.
2. Si es un grupo de tools, agrégalas al método `Tools()` de su grupo.
3. Crea un servicio por endpoint en `internal/application/services/` que arme esas tools y llame al orquestador.
4. Expón la ruta explícita en `internal/api/router.go`.

## Lecciones Aprendidas y Plan de Refactorización

Durante el desarrollo e integración de herramientas (tools) para Tool Calling, se identificaron las siguientes falencias y áreas de mejora que se abordarán en futuras refactorizaciones:

1. **Mapeo de Tipos Array en Esquemas (JSON Schema vs. SDKs de LLM)**:
   - _Problema encontrado_: Al traducir esquemas genéricos a proveedores como Gemini, las propiedades de tipo `array` requerían obligatoriamente la propiedad `items` (ej. definir que el array contiene strings). Sin esto, la API devolvía un error `400 Invalid Argument` (_missing field_).
   - _Mejora futura_: Centralizar y robustecer el mapeo de esquemas para validar automáticamente tipos complejos y arrays anidados.

2. **Selección y Filtrado Dinámico de Propiedades de Respuesta (`AddPropResponse`)**:
   - _Problema encontrado_: El JSON devuelto por los microservicios externos a veces contiene metadatos excesivos o estructuras anidadas que saturan el contexto del LLM o devuelven más información de la necesaria al frontend.
   - _Mejora futura_: Implementar un mecanismo fluido o encadenado al definir la tool (ej. `AddPropResponse("total")`, `AddPropResponse("productos")`) para que, tras la ejecución de `Execute()`, la respuesta se filtre y se envíe únicamente la propiedad o subconjunto de propiedades seleccionadas del JSON de respuesta.
