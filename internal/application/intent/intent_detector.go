package intent

import (
	"regexp"
	"strings"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// patternRule asocia un patrón regex con una intención de dominio.
type patternRule struct {
	pattern *regexp.Regexp
	intent  models.IntentType
}

// Detector clasifica la intención del usuario usando reglas deterministas (regex)
// para evitar llamadas al LLM cuando la intención es obvia. Solo recurre al LLM
// si no puede clasificar con certeza.
type Detector struct {
	rules []patternRule
}

// NewDetector crea un nuevo detector de intenciones con patrones predefinidos en español e inglés.
func NewDetector() *Detector {
	d := &Detector{}
	d.initRules()
	return d
}

// Detect analiza el mensaje del usuario y retorna la intención detectada.
// Usa evaluación de patrones regex sin consumir tokens del LLM.
func (d *Detector) Detect(message string) models.IntentType {
	normalized := strings.ToLower(strings.TrimSpace(message))

	for _, rule := range d.rules {
		if rule.pattern.MatchString(normalized) {
			return rule.intent
		}
	}

	return models.IntentUnknown
}

// DetectWithConfidence retorna la intención y si fue detectada con alta confianza.
// Si confidence es false, el orquestador debería delegar al LLM.
func (d *Detector) DetectWithConfidence(message string) (models.IntentType, bool) {
	intent := d.Detect(message)
	confident := intent != models.IntentUnknown && intent != models.IntentGeneral
	return intent, confident
}

// initRules inicializa los patrones regex para detección de intenciones.
// Los patrones están en español e inglés para máxima cobertura.
func (d *Detector) initRules() {
	d.rules = []patternRule{
		// Búsqueda de productos
		{
			pattern: regexp.MustCompile(`(busca|buscar|encuentra|encontrar|muestra|mostrar|listar?|search|find|show|list)\s.*(producto|artículo|item|product)`),
			intent:  models.IntentSearchProduct,
		},
		{
			pattern: regexp.MustCompile(`(qué|que|cuáles|cuales|tienes?|tienen|hay)\s.*(producto|artículo|disponible|product|available)`),
			intent:  models.IntentSearchProduct,
		},
		{
			pattern: regexp.MustCompile(`(busca|buscar|encuentra|encontrar)\s+\w+`),
			intent:  models.IntentSearchProduct,
		},

		// Consulta de producto específico
		{
			pattern: regexp.MustCompile(`(detalle|detalles|info|información|ver)\s.*(producto|artículo|product)`),
			intent:  models.IntentGetProduct,
		},
		{
			pattern: regexp.MustCompile(`producto\s+(id|#|número|num)?\s*\d+`),
			intent:  models.IntentGetProduct,
		},

		// Verificación de stock
		{
			pattern: regexp.MustCompile(`(stock|inventario|disponibilidad|existencia|cuántos?|cuantos?|hay)\s.*(disponible|quedan|tiene|product)`),
			intent:  models.IntentCheckStock,
		},
		{
			pattern: regexp.MustCompile(`(check|verificar?|consultar?)\s*(el\s+)?(stock|inventario|availability)`),
			intent:  models.IntentCheckStock,
		},

		// Crear venta
		{
			pattern: regexp.MustCompile(`(comprar|compra|adquirir|ordenar|pedir|quiero|necesito|crear?\s*venta|create?\s*sale|buy|purchase|order)`),
			intent:  models.IntentCreateSale,
		},
		{
			pattern: regexp.MustCompile(`(agregar?|añadir?|meter)\s.*(carrito|orden|pedido|cart)`),
			intent:  models.IntentCreateSale,
		},

		// Consultar venta
		{
			pattern: regexp.MustCompile(`(estado|status|consultar?|ver)\s.*(venta|orden|pedido|compra|sale|order)`),
			intent:  models.IntentGetSale,
		},
		{
			pattern: regexp.MustCompile(`(mi|mis)\s+(venta|orden|pedido|compra|order)`),
			intent:  models.IntentGetSale,
		},

		// Cancelar venta
		{
			pattern: regexp.MustCompile(`(cancelar?|anular?|devolver?|revertir?|cancel|revoke|return)\s.*(venta|orden|pedido|compra|sale|order)`),
			intent:  models.IntentCancelSale,
		},

		// General (saludos, ayuda, etc.)
		{
			pattern: regexp.MustCompile(`^(hola|hey|buenas?|hi|hello|ayuda|help|menú|menu|qué puedo|que puedo|cómo funciona|como funciona)`),
			intent:  models.IntentGeneral,
		},
	}
}

// MapIntentToTool retorna el nombre de la herramienta asociada a una intención.
// Retorna cadena vacía si la intención no mapea directamente a una herramienta.
func MapIntentToTool(intent models.IntentType) string {
	mapping := map[models.IntentType]string{
		models.IntentSearchProduct: "search_products",
		models.IntentGetProduct:    "get_product",
		models.IntentCheckStock:    "check_stock",
		models.IntentCreateSale:    "create_sale",
		models.IntentGetSale:       "get_sale",
		models.IntentCancelSale:    "cancel_sale",
	}
	return mapping[intent]
}
