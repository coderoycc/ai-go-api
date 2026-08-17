package engine

import (
	"net/http"

	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	"github.com/gin-gonic/gin"
)

// FeatureRecipe define la receta completa para desplegar una nueva funcionalidad impulsada por IA y Tool Calling.
type FeatureRecipe struct {
	Path         string       // Ruta del endpoint (ej: "/products/chat")
	SystemPrompt string       // Prompt del sistema especializado para este dominio
	Tools        []ports.Tool // Herramientas (APIs) disponibles para este dominio
}

// FeatureEngine gestiona y registra recetas de funcionalidades en el router HTTP de forma unificada.
type FeatureEngine struct {
	orchestrator *orchestrator.Orchestrator
}

// NewFeatureEngine crea una nueva instancia del motor de recetas de funcionalidades.
func NewFeatureEngine(orch *orchestrator.Orchestrator) *FeatureEngine {
	return &FeatureEngine{orchestrator: orch}
}

// Register procesa una receta y registra automáticamente el endpoint POST en el grupo Gin,
// unificando la extracción de sesión, validación de permisos, ejecución del orquestador y formateo de respuesta.
func (e *FeatureEngine) Register(group *gin.RouterGroup, recipe FeatureRecipe) {
	group.POST(recipe.Path, func(c *gin.Context) {
		var req struct {
			SessionID string `json:"session_id" binding:"required"`
			Message   string `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Formato de petición inválido",
				"details": err.Error(),
			})
			return
		}

		userPerm := models.PermissionRead
		if permVal, exists := c.Get("permission"); exists {
			if p, ok := permVal.(models.Permission); ok {
				userPerm = p
			}
		}

		input := orchestrator.ChatInput{
			SessionID:  req.SessionID,
			Message:    req.Message,
			Permission: userPerm,
		}

		apiResp := e.orchestrator.HandleChatWithPrompt(c.Request.Context(), input, recipe.Tools, recipe.SystemPrompt)
		c.JSON(http.StatusOK, apiResp)
	})
}
