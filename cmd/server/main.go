package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/coderoycc/ai-go-api/internal/api"
	appctx "github.com/coderoycc/ai-go-api/internal/application/context"
	"github.com/coderoycc/ai-go-api/internal/application/intent"
	"github.com/coderoycc/ai-go-api/internal/application/orchestrator"
	"github.com/coderoycc/ai-go-api/internal/application/policies"
	"github.com/coderoycc/ai-go-api/internal/application/response"
	toolsApp "github.com/coderoycc/ai-go-api/internal/application/tools"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	productClient "github.com/coderoycc/ai-go-api/internal/infrastructure/clients/products"
	llmClaude "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/claude"
	llmDeepSeek "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/deepseek"
	llmGemini "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/gemini"
	llmOllama "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/ollama"
	llmOpenAI "github.com/coderoycc/ai-go-api/internal/infrastructure/llm/openai"
	redisStore "github.com/coderoycc/ai-go-api/internal/infrastructure/memory/redis"
	toolsInfra "github.com/coderoycc/ai-go-api/internal/infrastructure/tools"
	config "github.com/coderoycc/ai-go-api/internal/shared/config"
)

func main() {
	// 1. Cargar Configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error crítico al cargar configuración: %v", err)
	}

	log.Printf("Iniciando AI Orchestrator Engine en entorno: %s", cfg.Server.Env)

	// 2. Inicializar Almacenamiento de Memoria (Redis)
	redisMemory, err := redisStore.NewMemoryStore(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Error conectando a Redis en %s: %v", cfg.Redis.Addr(), err)
	}
	defer redisMemory.Close()
	log.Println("Conexión a Redis establecida correctamente")

	// 3. Inicializar Adaptador LLM según proveedor configurado
	ctx := context.Background()
	var llmAdapter ports.LLM

	switch cfg.LLM.Provider {
	case "openai":
		llmAdapter = llmOpenAI.NewAdapter(cfg.LLM.APIKey, cfg.LLM.Model)
	case "gemini":
		llmAdapter, err = llmGemini.NewAdapter(ctx, cfg.LLM.APIKey, cfg.LLM.Model)
		if err != nil {
			log.Fatalf("Error inicializando adaptador Gemini: %v", err)
		}
	case "claude":
		llmAdapter = llmClaude.NewAdapter(cfg.LLM.APIKey, cfg.LLM.Model)
	case "deepseek":
		llmAdapter = llmDeepSeek.NewAdapter(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.BaseURL)
	case "ollama":
		llmAdapter = llmOllama.NewAdapter(cfg.LLM.Model, cfg.LLM.BaseURL)
	default:
		log.Fatalf("Proveedor LLM no soportado: %s", cfg.LLM.Provider)
	}
	log.Printf("Proveedor LLM activo: %s (Modelo: %s)", cfg.LLM.Provider, cfg.LLM.Model)

	// 4. Inicializar Cliente HTTP Externo (API a la que llamará la tool)
	prodClient := productClient.NewClient(cfg.Clients.ProductsURL, cfg.Clients.Timeout)

	// 5. Inicializar Registro y Registrar la Tool
	registry := toolsApp.NewRegistry()

	_ = registry.Register(toolsInfra.NewProductSearchTool(prodClient))
	log.Printf("Herramientas registradas: %d herramientas disponibles", len(registry.List()))

	// 6. Ensamblar Componentes del Dominio y Aplicación
	policyEngine := policies.DefaultEngine()
	toolExecutor := toolsApp.NewExecutor(registry, policyEngine)
	intentDetector := intent.NewDetector()
	contextManager := appctx.NewManager(redisMemory)
	formatter := response.NewFormatter()

	orc := orchestrator.NewOrchestrator(
		llmAdapter,
		contextManager,
		intentDetector,
		policyEngine,
		registry,
		toolExecutor,
		formatter,
	)

	// 7. Configurar Servidor HTTP (Gin)
	router := api.SetupRouter(orc)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Iniciar servidor en segundo plano
	go func() {
		log.Printf("Servidor HTTP escuchando en el puerto %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error en servidor HTTP: %v", err)
		}
	}()

	// 8. Manejo de Cierre Graceful (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Apagando servidor HTTP...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Servidor forzado al apagarse: %v", err)
	}

	log.Println("Servidor detenido correctamente.")
}
