package api

import (
	"time"

	"github.com/coderoycc/ai-go-api/internal/application/engine"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	"github.com/gin-gonic/gin"
)

// SetupRouter configura el enrutador Gin utilizando FeatureEngine y recetas declarativas.
func SetupRouter(featureEngine *engine.FeatureEngine, productToolGroup ports.ToolGroup, authAPIKey string, authEnabled bool) *gin.Engine {
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
		// Receta para el chat de productos (añadir nuevas funcionalidades es tan simple como registrar otra receta aquí)
		featureEngine.Register(v1, engine.FeatureRecipe{
			Path: "/products/chat",
			SystemPrompt: `Eres un asistente de ventas inteligente. Tu rol es ayudar a los usuarios con:
- Buscar productos y consultar disponibilidad.
- Verificar stock de productos.

Responde de forma concisa, profesional y en el mismo idioma que el usuario.
Cuando necesites datos, usa las herramientas disponibles. No inventes información.`,
			Tools: productToolGroup.Tools(),
		})
	}

	return router
}