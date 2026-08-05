# AI Orchestrator Engine 🚀

Orquestador de IA Empresarial (AI Gateway) de alto rendimiento escrito en Go. Agnóstico a proveedores de IA (OpenAI, Gemini, Claude, DeepSeek, Ollama), seguro y optimizado con Arquitectura Hexagonal y DDD.

---

## 📋 Requisitos Previos

- **Go**: v1.22+
- **Docker & Docker Compose** (opcional, para ejecución contenedorizada)
- **Redis** (opcional, si usas almacenamiento de sesión externo)
- **Ollama** (opcional, para modelos locales offline sin costo)

---

## 🛠️ Dependencias Principales

```
- github.com/gin-gonic/gin                  → Framework Web HTTP
- github.com/redis/go-redis/v9              → Persistencia de memoria conversacional
- github.com/sashabaranov/go-openai         → SDK OpenAI / DeepSeek / Ollama
- google.golang.org/genai                   → SDK oficial Google Gemini
- github.com/anthropics/anthropic-sdk-go    → SDK oficial Anthropic Claude
- github.com/joho/godotenv                  → Carga de variables de entorno .env
- github.com/stretchr/testify               → Mocks y Suite de pruebas unitarias
```

---

## ⚙️ Configuración (.env)

Copia la plantilla `.env.example` a `.env`:

```bash
cp .env.example .env
```

### Configuración Mínima (`.env`):

```ini
PORT=8080
APP_ENV=dev

# Selección de proveedor: ollama | openai | gemini | claude | deepseek
LLM_PROVIDER=ollama
LLM_MODEL=llama3.1

# API Key del proveedor activo (no requerida para Ollama)
OPENAI_API_KEY=sk-...
GEMINI_API_KEY=AIzaSy...
ANTHROPIC_API_KEY=sk-ant-...

# Redis y Microservicios
REDIS_HOST=localhost
REDIS_PORT=6379
PRODUCTS_SERVICE_URL=http://localhost:8081
SALES_SERVICE_URL=http://localhost:8082
```

---

## 🚀 Inicio Rápido

### Opción A: Ejecución Local en Go

```bash
# 1. Instalar dependencias
go mod download

# 2. Iniciar el servidor
go run cmd/server/main.go
```

### Opción B: Ejecución con Docker Compose (Recomendado)

Levanta la API en Go, Redis y Ollama local con un solo comando:

```bash
docker-compose up -d --build
```

---

## 🧪 Pruebas Unitarias

Ejecutar la suite completa de tests (con mocks):

```bash
go test -v ./...
```

---

## 📡 Ejemplo de Petición API

**Endpoint**: `POST /api/v1/chat`

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "sesion-usuario-123",
    "message": "Quiero consultar el stock de laptops"
  }'
```

**Respuesta de Ejemplo**:

```json
{
  "session_id": "sesion-usuario-123",
  "message": "Actualmente tenemos 15 unidades de Laptops HP disponibles.",
  "mode": "natural",
  "intent": "check_stock",
  "tokens_used": 42,
  "timestamp": "2026-08-05T21:00:00Z"
}
```

---

## 🏛️ Arquitectura del Proyecto

```
cmd/server/             → Punto de entrada e Inyección de Dependencias (Wiring)
internal/
  ├── api/              → Handlers HTTP Gin, DTOs y Middlewares
  ├── application/      → Orchestrator, Policy Engine, Intent Detector, Context & Tools
  ├── domain/           → Modelos del Dominio y Puertos (Interfaces Hexagonales)
  ├── infrastructure/   → Adaptadores LLM (5 proveedores), Redis Client, HTTP Clients & Tools
  └── shared/config/    → Configuración centralizada y validación Fail-Fast
```
