package config

import (
	"fmt"

	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config agrupa toda la configuración centralizada del sistema.
type Config struct {
	Server  ServerConfig
	LLM     LLMConfig
	Redis   RedisConfig
	Clients ClientsConfig
	Auth    AuthConfig
}

// AuthConfig define los parámetros para la verificación de autenticación/autorización.
type AuthConfig struct {
	APIKey  string
	Enabled bool
}

// ServerConfig define los parámetros del servidor HTTP.
type ServerConfig struct {
	Port         string
	Env          string // dev, prod, test
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// LLMConfig define los parámetros para el proveedor de Inteligencia Artificial.
type LLMConfig struct {
	Provider    string  // openai, gemini, claude, deepseek, ollama
	Model       string  // gpt-4o, gemini-2.5-flash, claude-sonnet-4-20250514, deepseek-chat, llama3.1
	APIKey      string  // API Key requerida (excepto en Ollama)
	BaseURL     string  // URL Base personalizada (para DeepSeek/Ollama)
	Temperature float64 // Creatividad (0.0 a 1.0)
	MaxTokens   int     // Límite de tokens en la respuesta
	Timeout     time.Duration
}

// RedisConfig define los parámetros de conexión a la memoria Redis.
type RedisConfig struct {
	Host       string
	Port       string
	Password   string
	DB         int
	SessionTTL time.Duration
}

// Addr retorna el string de conexión "host:port".
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// ClientsConfig define la URL base y timeout para la API externa.
type ClientsConfig struct {
	ProductsURL string
	Timeout     time.Duration
}

// Load lee el archivo .env (si existe) y carga la configuración desde las variables de entorno.
// Valida que la configuración cargada sea consistente y segura para iniciar la aplicación.
func Load() (*Config, error) {
	// Intentar cargar el archivo .env si está presente (no falla si no existe)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Env:          getEnv("APP_ENV", "dev"),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		LLM: LLMConfig{
			Provider:    strings.ToLower(getEnv("LLM_PROVIDER", "ollama")),
			Model:       getEnv("LLM_MODEL", ""),
			APIKey:      getEnv("LLM_API_KEY", ""),
			BaseURL:     getEnv("LLM_BASE_URL", ""),
			Temperature: getEnvFloat("LLM_TEMPERATURE", 0.7),
			MaxTokens:   getEnvInt("LLM_MAX_TOKENS", 1024),
			Timeout:     getEnvDuration("LLM_TIMEOUT", 30*time.Second),
		},
		Redis: RedisConfig{
			Host:       getEnv("REDIS_HOST", "localhost"),
			Port:       getEnv("REDIS_PORT", "6379"),
			Password:   getEnv("REDIS_PASSWORD", ""),
			DB:         getEnvInt("REDIS_DB", 0),
			SessionTTL: getEnvDuration("REDIS_SESSION_TTL", 30*time.Minute),
		},
		Clients: ClientsConfig{
			ProductsURL: getEnv("PRODUCTS_SERVICE_URL", "http://localhost:8081"),
			Timeout:     getEnvDuration("CLIENTS_TIMEOUT", 5*time.Second),
		},
		Auth: AuthConfig{
			APIKey:  getEnv("AI_ENGINE_API_KEY", "default_secret_key_change_in_prod"),
			Enabled: getEnvBool("AUTH_ENABLED", false),
		},
	}

	// Asignar modelo por defecto si no se especificó según el proveedor
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = getDefaultModel(cfg.LLM.Provider)
	}

	// Asignar API Key desde variable específica del proveedor si LLM_API_KEY no está configurada
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = getProviderAPIKey(cfg.LLM.Provider)
	}

	// Validar configuración crítica (Fail-Fast)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuración inválida: %w", err)
	}

	return cfg, nil
}

// Validate realiza la validación estricta de la configuración cargada.
func (c *Config) Validate() error {
	validProviders := map[string]bool{
		"openai":   true,
		"gemini":   true,
		"claude":   true,
		"deepseek": true,
		"ollama":   true,
	}

	if !validProviders[c.LLM.Provider] {
		return fmt.Errorf("proveedor LLM '%s' no soportado. Opciones válidas: openai, gemini, claude, deepseek, ollama", c.LLM.Provider)
	}

	// Ollama no requiere API Key
	if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		return fmt.Errorf("el proveedor LLM '%s' requiere una API Key configurada (usa LLM_API_KEY o %s)",
			c.LLM.Provider, getProviderEnvName(c.LLM.Provider))
	}

	// Validar que la autenticación no use la clave secreta por defecto fuera de entorno dev
	if c.Auth.Enabled && c.Auth.APIKey == "default_secret_key_change_in_prod" && c.Server.Env != "dev" {
		return fmt.Errorf("autenticación habilitada en entorno '%s' requiere una AI_ENGINE_API_KEY explícita y segura", c.Server.Env)
	}

	return nil
}

func getDefaultModel(provider string) string {
	switch provider {
	case "openai":
		return "gpt-4o"
	case "gemini":
		return "gemini-2.5-flash"
	case "claude":
		return "claude-sonnet-4-20250514"
	case "deepseek":
		return "deepseek-chat"
	case "ollama":
		return "llama3.1"
	default:
		return "llama3.1"
	}
}

func getProviderAPIKey(provider string) string {
	switch provider {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "gemini":
		return os.Getenv("GEMINI_API_KEY")
	case "claude":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY")
	default:
		return ""
	}
}

func getProviderEnvName(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	default:
		return "LLM_API_KEY"
	}
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultValue
}
