package regex

import (
	"context"
	"regexp"
	"strings"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// actionRule asocia un patrón regex con una categoría genérica de acción.
type actionRule struct {
	pattern *regexp.Regexp
	action  string
}

// RegexIntentDetector implementa ports.IntentDetector utilizando expresiones regulares deterministas
// para validar la veracidad y coherencia de la intención en el mensaje del usuario.
type RegexIntentDetector struct {
	rules []actionRule
}

// NewRegexIntentDetector crea un nuevo validador de intenciones basado en expresiones regulares.
func NewRegexIntentDetector() ports.IntentDetector {
	return &RegexIntentDetector{
		rules: []actionRule{
			// Listado / Búsqueda / Consulta
			{
				pattern: regexp.MustCompile(`(?i)\b(busca|buscar|encuentra|encontrar|muestra|mostrar|listar?|ver|detalle|detalles|info|información|consultar?|obtener|search|find|show|list|get|check|verificar?|cuántos?|cuántas?|cuantos?|cuantas?|stock|inventario|disponibilidad)\b`),
				action:  "query",
			},
			// Creación / Adición / Registro / Compra
			{
				pattern: regexp.MustCompile(`(?i)\b(crear?|agregar?|añadir?|meter|comprar|compra|adquirir|ordenar|pedir|quiero|necesito|registrar?|create?|buy|purchase|order|add)\b`),
				action:  "create",
			},
			// Actualización / Edición / Modificación
			{
				pattern: regexp.MustCompile(`(?i)\b(actualizar?|modificar?|editar?|cambiar?|update|edit|modify|change)\b`),
				action:  "update",
			},
			// Eliminación / Cancelación
			{
				pattern: regexp.MustCompile(`(?i)\b(cancelar?|anular?|devolver?|revertir?|eliminar?|borrar?|cancel|revoke|return|delete|remove)\b`),
				action:  "delete",
			},
			// Conversación General / Saludos / Ayuda
			{
				pattern: regexp.MustCompile(`(?i)\b(hola|hey|buenas?|hi|hello|ayuda|help|menú|menu|gracias|thanks?)\b`),
				action:  "general",
			},
		},
	}
}

// NewDetector retorna una instancia de ports.IntentDetector basada en expresiones regulares.
func NewDetector() ports.IntentDetector {
	return NewRegexIntentDetector()
}

// Validate evalúa si el mensaje tiene una intención de acción válida o coherente.
func (d *RegexIntentDetector) Validate(ctx context.Context, message string) (bool, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(message))
	normalized = strings.Trim(normalized, "¿?¡!")

	if normalized == "" {
		return false, "unknown", nil
	}

	for _, rule := range d.rules {
		if rule.pattern.MatchString(normalized) {
			return true, rule.action, nil
		}
	}

	// Si contiene palabras legibles de al menos 2 caracteres, se considera una intención válida general.
	words := strings.Fields(normalized)
	if len(words) > 0 {
		return true, "general", nil
	}

	return false, "unknown", nil
}
