package services

import (
	"context"
	"time"

	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	productsTools "github.com/coderoycc/ai-go-api/internal/infrastructure/tools/products"
)

// ProductChatService es el servicio de aplicación del endpoint de chat de productos.
// Arma las herramientas de productos y las envía al orquestador en cada petición.
type ProductChatService struct {
	orchestrator *orchestrator.Orchestrator
	productTools *productsTools.ProductTool
}

// NewProductChatService crea el servicio de chat de productos con sus dependencias.
func NewProductChatService(
	llm ports.LLM,
	contextManager *appctx.Manager,
	intentDetector ports.IntentDetector,
	policyEngine *policies.Engine,
	formatter *response.Formatter,
	productsBaseURL string,
	productsTimeout time.Duration,
) *ProductChatService {
	return &ProductChatService{
		orchestrator: orchestrator.NewOrchestrator(llm, contextManager, intentDetector, policyEngine, formatter),
		productTools: productsTools.NewProductTool(productsBaseURL, productsTimeout),
	}
}

// HandleChat procesa el mensaje usando las herramientas del endpoint de productos.
func (s *ProductChatService) HandleChat(ctx context.Context, input orchestrator.ChatInput) response.APIResponse {
	return s.orchestrator.HandleChat(ctx, input, s.productTools.Tools())
}