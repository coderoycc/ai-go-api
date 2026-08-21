package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/gin-gonic/gin"
)

// LoggerMiddleware realiza un registro estructurado básico de las peticiones HTTP.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[HTTP] %d | %13v | %s | %-7s %s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// CORSMiddleware habilita políticas de CORS permisivas para comunicación con Frontend/Mobile.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RecoveryMiddleware captura panics imprevistos y responde con HTTP 500.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERY] error en handler: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Error interno del servidor",
				})
			}
		}()
		c.Next()
	}
}

// AuthMiddleware valida la API Key del servicio (si está habilitada) y extrae
// los headers HTTP de autorización del cliente: X-User-ID y X-User-Permission.
//
// Cuando AUTH_ENABLED=false: no se valida ninguna clave y se otorga el permiso
// supremo (PermissionWrite) a todos, ya que el sistema opera en modo abierto.
// Cuando AUTH_ENABLED=true: se valida la API Key y se extrae el permiso del header
// X-User-Permission ("read" o "write"). Si el header está ausente, se asigna "read".
func AuthMiddleware(apiKey string, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userPerm models.Permission

		if !enabled {
			// Sin autenticación: permiso supremo para todos.
			userPerm = models.PermissionWrite
		} else {
			reqKey := c.GetHeader("X-API-Key")
			if reqKey == "" {
				reqKey = c.GetHeader("Authorization")
				reqKey = strings.TrimPrefix(reqKey, "Bearer ")
			}
			if reqKey != apiKey {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API Key de servicio no válida o no provista"})
				return
			}
			userPermStr := strings.ToLower(strings.TrimSpace(c.GetHeader("X-User-Permission")))
			if userPermStr == "write" {
				userPerm = models.PermissionWrite
			} else {
				userPerm = models.PermissionRead
			}
		}

		c.Set("authenticated", true)
		c.Set("user_id", c.GetHeader("X-User-ID"))
		c.Set("permission", userPerm)
		c.Next()
	}
}
