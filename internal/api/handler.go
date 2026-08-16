package api

import (
	"net/http"

	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/application/services"
	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/gin-gonic/gin"
)

// ProductChatHandler expone el endpoint HTTP de chat de productos.
type ProductChatHandler struct {
	service *services.ProductChatService
}

// NewProductChatHandler instancia un nuevo controlador HTTP de chat de productos.
func NewProductChatHandler(service *services.ProductChatService) *ProductChatHandler {
	return &ProductChatHandler{service: service}
}

// HandleChat procesa las solicitudes POST /api/v1/products/chat.
// Extrae session_id y message del body y los envía al servicio de productos.
func (h *ProductChatHandler) HandleChat(c *gin.Context) {
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

	response := h.service.HandleChat(c.Request.Context(), input)

	c.JSON(http.StatusOK, response)
}