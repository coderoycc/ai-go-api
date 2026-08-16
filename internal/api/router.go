package api

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/application/services"
	"github.com/gin-gonic/gin"
)

// SetupRouter configura el enrutador Gin con sus middlewares y rutas explícitas.
func SetupRouter(productChatService *services.ProductChatService, authAPIKey string, authEnabled bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Registrar middlewares
	router.Use(RecoveryMiddleware())
	router.Use(LoggerMiddleware())
	router.Use(CORSMiddleware())

	// Ruta de HealthCheck
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, HealthResponse{
			Status:    "UP",
			Service:   "AI Orchestrator Engine",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})

	// Grupo API v1
	v1 := router.Group("/api/v1")
	v1.Use(AuthMiddleware(authAPIKey, authEnabled))
	{
		// Endpoint explícito de chat para productos
		productHandler := NewProductChatHandler(productChatService)
		v1.POST("/products/chat", productHandler.HandleChat)
	}

	return router
}