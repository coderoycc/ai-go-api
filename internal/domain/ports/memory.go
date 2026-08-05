package ports

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
)

// Memory define el contrato para la persistencia del estado conversacional.
// El adaptador principal es Redis, pero la interfaz permite cualquier backend
// de almacenamiento que soporte operaciones CRUD sobre el contexto de sesión.
type Memory interface {
	// Load recupera el contexto de sesión activo para el sessionID dado.
	// Retorna nil sin error si la sesión no existe.
	Load(ctx context.Context, sessionID string) (*models.SessionContext, error)

	// Save persiste el contexto de sesión, creándolo o actualizándolo.
	// Debe respetar el TTL definido en el SessionContext.
	Save(ctx context.Context, sessionID string, session *models.SessionContext) error

	// Delete elimina completamente el contexto de sesión del almacenamiento.
	Delete(ctx context.Context, sessionID string) error
}
