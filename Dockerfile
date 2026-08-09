# ==========================================
# STAGE 1: Builder
# ==========================================
FROM golang:1.26-alpine AS builder

# Instalar ca-certificates y tzdata
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar archivos de dependencias e instalar primero para aprovechar la caché de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiar todo el código fuente
COPY . .

# Compilar binario estático optimizado (-s -w para reducir tamaño)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/server \
    ./cmd/server

# ==========================================
# STAGE 2: Final Minimal Image
# ==========================================
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar binario compilado desde el builder
COPY --from=builder /app/server /app/server

# Copiar .env si existe (opcional)
COPY --from=builder /app/.env.example /app/.env.example

# Exponer el puerto por defecto de la API
EXPOSE 8080

# Usuario no-root por seguridad
RUN adduser -D -u 1000 appuser
USER appuser

# Comando de inicio del motor orquestador
ENTRYPOINT ["/app/server"]
