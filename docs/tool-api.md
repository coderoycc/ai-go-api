# Tool API — Búsqueda de Productos (`search_products`)

Documentación de la tool registrada en el AI Orchestrator Engine. Esta tool expone
al LLM un endpoint de la API externa para que el modelo pueda consultar datos reales.

## Flujo de la tool

```
POST /api/v1/chat
   │
   ▼
Handler (internal/api/handler.go) ──► Orchestrator (orchestrator.go)
   │
   ├── Context Manager (context_manager.go)   → carga/crea sesión en Redis
   ├── Intent Detector (intent_detector.go)   → clasifica la intención (regex, sin LLM)
   ├── Policy Engine (policy_engine.go)       → valida que la acción esté permitida
   │
   └── Ruta: si la intención mapea 1:1 a la tool → RUTA DIRECTA (sin LLM)
        └── ProductSearchTool (tools.go) ──► ProductClient (client.go) ──► TU API
            (lo que el LLM      (ejecuta el HTTP real:
             ve como definición)  POST {PRODUCTS_SERVICE_URL}/api/productos/buscar)
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

| #   | Archivo                                              | Qué contiene                                                                                                                             |
| --- | ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `internal/domain/ports/product.go`                   | Los **structs**: `SearchProductsRequest` (body), `ProductSearchResponse` (wrapper) y `Product` (item). Define el puerto `ProductClient`. |
| 2   | `internal/infrastructure/clients/products/client.go` | El **HTTP real**: método, path (`/api/productos/buscar`), headers y decode de la respuesta.                                              |
| 3   | `internal/infrastructure/tools/tools.go`             | Lo que **ve el LLM**: `Description()` y `Parameters()` (schema JSON con los 12 campos) + `Execute()` que une todo.                       |

**Regla clave:** los nombres JSON de los structs (1) deben coincidir con los nombres
de `properties` en `Parameters()` (3), porque el LLM genera `{"campo": valor}` y ese
mismo JSON se envía como body a la API.

## Registrar la tool

La tool se registra una sola vez en el arranque:

`cmd/server/main.go`

```go
registry := toolsApp.NewRegistry()
_ = registry.Register(toolsInfra.NewProductSearchTool(prodClient))
```

- `prodClient := productClient.NewClient(cfg.Clients.ProductsURL, cfg.Clients.Timeout)`
- La política que la permite: `internal/application/policies/policy_engine.go` → `AllowedTools: ["search_products"]`.

## Prueba de la ruta directa (sin LLM)

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"s1","message":"buscar laptops"}'
```

Si el intent detector reconoce la búsqueda, responde en modo `raw` con el JSON de la
API sin consumir tokens del LLM.

## Cómo agregar otra API/tool

1. Define los structs de request/response en `internal/domain/ports/` (o un puerto nuevo).
2. Crea/ajusta el cliente HTTP en `internal/infrastructure/clients/` con el método y path.
3. Crea la tool en `internal/infrastructure/tools/` implementando `ports.Tool` (`Name`, `Description`, `Parameters`, `Execute`).
4. Regístrala en `cmd/server/main.go` y añádela a `AllowedTools` en `policy_engine.go`.
5. Si quieres que una intención la dispare en ruta directa, agrega el mapeo en `internal/application/intent/intent_detector.go` (`MapIntentToTool`).

## Lecciones Aprendidas y Plan de Refactorización

Durante el desarrollo e integración de herramientas (tools) para Tool Calling, se identificaron las siguientes falencias y áreas de mejora que se abordarán en futuras refactorizaciones:

1. **Mapeo de Tipos Array en Esquemas (JSON Schema vs. SDKs de LLM)**:
   - _Problema encontrado_: Al traducir esquemas genéricos a proveedores como Gemini, las propiedades de tipo `array` requerían obligatoriamente la propiedad `items` (ej. definir que el array contiene strings). Sin esto, la API devolvía un error `400 Invalid Argument` (_missing field_).
   - _Mejora futura_: Centralizar y robustecer el `SchemaMapper` (`internal/application/tools/schema.go`) para validar automáticamente tipos complejos y arrays anidados.

2. **Selección y Filtrado Dinámico de Propiedades de Respuesta (`AddPropResponse`)**:
   - _Problema encontrado_: El JSON devuelto por los microservicios externos a veces contiene metadatos excesivos o estructuras anidadas que saturan el contexto del LLM o devuelven más información de la necesaria al frontend.
   - _Mejora futura_: Implementar un mecanismo fluido o encadenado al definir la tool (ej. `AddPropResponse("total")`, `AddPropResponse("productos")`) para que, tras la ejecución de `Execute()`, la respuesta se filtre y se envíe únicamente la propiedad o subconjunto de propiedades seleccionadas del JSON de respuesta.
