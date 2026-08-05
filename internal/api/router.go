package api

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/gin-gonic/gin"
)

// SetupRouter configura el enrutador Gin con sus middlewares y rutas.
func SetupRouter(orc *orchestrator.Orchestrator) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Registrar middlewares
	router.Use(RecoveryMiddleware())
	router.Use(LoggerMiddleware())
	router.Use(CORSMiddleware())

	handler := NewChatHandler(orc)

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
	{
		v1.POST("/chat", handler.HandleChat)
	}

	return router
}
