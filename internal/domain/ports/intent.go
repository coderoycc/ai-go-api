package ports

import "context"

// IntentDetector define la interfaz para evaluar la validez/veracidad de la intención de un mensaje del usuario.
type IntentDetector interface {
	// Validate evalúa si el mensaje contiene una intención de acción válida o coherente.
	// Retorna isValid (true si es una intención válida), actionType (categoría genérica de acción) y err si ocurre un error.
	Validate(ctx context.Context, message string) (isValid bool, actionType string, err error)
}
