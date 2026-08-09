package policies

import (
	"strings"
	"sync"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Engine es el motor de políticas que valida si una intención o herramienta
// está permitida antes de consumir tokens del LLM. Actúa como guardián de
// seguridad desacoplado del proveedor de IA.
type Engine struct {
	mu    sync.RWMutex
	rules []models.PolicyRule
}

// NewEngine crea un nuevo motor de políticas con las reglas proporcionadas.
func NewEngine(rules []models.PolicyRule) *Engine {
	return &Engine{rules: rules}
}

// DefaultEngine crea un motor de políticas con reglas permisivas por defecto
// que permite todas las intenciones de negocio y bloquea las desconocidas.
func DefaultEngine() *Engine {
	return &Engine{
		rules: []models.PolicyRule{
			{
				Name:        "allow_business_intents",
				Description: "Permite intenciones de negocio estándar",
				AllowedIntents: []models.IntentType{
					models.IntentSearchProduct,
					models.IntentGetProduct,
					models.IntentCheckStock,
					models.IntentCreateSale,
					models.IntentGetSale,
					models.IntentCancelSale,
					models.IntentGeneral,
				},
				AllowedTools: []string{
					"search_products",
				},
			},
			{
				Name:           "block_unknown",
				Description:    "Bloquea intenciones no reconocidas",
				DeniedIntents:  []models.IntentType{models.IntentUnknown},
				RequiresAuth:   false,
			},
		},
	}
}

// EvaluateIntent evalúa si una intención está permitida según las políticas configuradas.
// Retorna PolicyResult indicando si la acción es permitida y la razón.
func (e *Engine) EvaluateIntent(req models.PolicyEvalRequest) models.PolicyResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Primero verificar reglas de denegación explícita
	for _, rule := range e.rules {
		if containsIntent(rule.DeniedIntents, req.Intent) {
			return models.PolicyResult{
				Allowed:    false,
				Reason:     "Intención bloqueada por política: " + rule.Name,
				PolicyName: rule.Name,
			}
		}
	}

	// Verificar si requiere autenticación
	for _, rule := range e.rules {
		if rule.RequiresAuth && !req.Authenticated {
			if containsIntent(rule.AllowedIntents, req.Intent) {
				return models.PolicyResult{
					Allowed:    false,
					Reason:     "La intención requiere autenticación",
					PolicyName: rule.Name,
				}
			}
		}
	}

	// Verificar si está explícitamente permitida
	for _, rule := range e.rules {
		if containsIntent(rule.AllowedIntents, req.Intent) {
			return models.PolicyResult{
				Allowed:    true,
				Reason:     "Intención permitida por política: " + rule.Name,
				PolicyName: rule.Name,
			}
		}
	}

	// Si no hay regla explícita, denegar por defecto
	return models.PolicyResult{
		Allowed:    false,
		Reason:     "Intención no contemplada en ninguna política",
		PolicyName: "default_deny",
	}
}

// EvaluateTool evalúa si una herramienta específica puede ejecutarse.
func (e *Engine) EvaluateTool(req models.PolicyEvalRequest) models.PolicyResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Verificar denegación explícita de herramienta
	for _, rule := range e.rules {
		if containsString(rule.DeniedTools, req.ToolName) {
			return models.PolicyResult{
				Allowed:    false,
				Reason:     "Herramienta bloqueada por política: " + rule.Name,
				PolicyName: rule.Name,
			}
		}
	}

	// Verificar permisos explícitos
	for _, rule := range e.rules {
		if containsString(rule.AllowedTools, req.ToolName) {
			return models.PolicyResult{
				Allowed:    true,
				Reason:     "Herramienta permitida por política: " + rule.Name,
				PolicyName: rule.Name,
			}
		}
	}

	return models.PolicyResult{
		Allowed:    false,
		Reason:     "Herramienta '" + req.ToolName + "' no contemplada en políticas",
		PolicyName: "default_deny",
	}
}

// AddRule agrega una nueva regla al motor de políticas en tiempo de ejecución.
func (e *Engine) AddRule(rule models.PolicyRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// IsDirectResolvable determina si una intención puede resolverse sin usar el LLM.
// Las intenciones directas son aquellas que mapean 1:1 a una herramienta.
func (e *Engine) IsDirectResolvable(intent models.IntentType) bool {
	directIntents := map[models.IntentType]bool{
		models.IntentSearchProduct: true,
		models.IntentGetProduct:    true,
		models.IntentCheckStock:    true,
		models.IntentCreateSale:    true,
		models.IntentGetSale:       true,
		models.IntentCancelSale:    true,
	}
	return directIntents[intent]
}

func containsIntent(list []models.IntentType, item models.IntentType) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func containsString(list []string, item string) bool {
	for _, v := range list {
		if strings.EqualFold(v, item) {
			return true
		}
	}
	return false
}
