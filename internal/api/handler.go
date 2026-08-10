package api

import (
	"net/http"

	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/gin-gonic/gin"
)

// ChatHandler expone endpoints HTTP para interactuar con el motor orquestador.
type ChatHandler struct {
	orchestrator *orchestrator.Orchestrator
}

// NewChatHandler instancia un nuevo controlador HTTP de Chat.
func NewChatHandler(orc *orchestrator.Orchestrator) *ChatHandler {
	return &ChatHandler{
		orchestrator: orc,
	}
}

// HandleChat procesa las solicitudes POST /api/v1/chat.
// Extrae session_id y message del body limpio y los envía al orquestador.
func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req ChatHTTPRequest
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

	userID := ""
	if uidVal, exists := c.Get("user_id"); exists {
		if uid, ok := uidVal.(string); ok {
			userID = uid
		}
	}

	input := orchestrator.ChatInput{
		SessionID:  req.SessionID,
		Message:    req.Message,
		UserID:     userID,
		Permission: userPerm,
	}

	response := h.orchestrator.HandleChat(c.Request.Context(), input)

	c.JSON(http.StatusOK, response)
}
