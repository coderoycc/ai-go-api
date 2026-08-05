package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/redis/go-redis/v9"
)

const (
	// defaultTTL es el tiempo de vida por defecto de una sesión si no se especifica.
	defaultTTL = 30 * time.Minute
	// keyPrefix es el prefijo usado para las claves de sesión en Redis.
	keyPrefix = "session:"
)

// MemoryStore implementa ports.Memory usando Redis como backend de almacenamiento.
// Serializa/deserializa SessionContext como JSON y soporta TTL configurable.
type MemoryStore struct {
	client *redis.Client
}

// NewMemoryStore crea una nueva instancia de MemoryStore conectada a Redis.
// Valida la conexión con un PING antes de retornar.
func NewMemoryStore(addr, password string, db int) (*MemoryStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: no se pudo conectar a %s: %w", addr, err)
	}

	return &MemoryStore{client: client}, nil
}

// Load recupera el contexto de sesión desde Redis para el sessionID dado.
// Retorna nil sin error si la sesión no existe (key expirada o inexistente).
func (m *MemoryStore) Load(ctx context.Context, sessionID string) (*models.SessionContext, error) {
	key := keyPrefix + sessionID

	data, err := m.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Sesión no encontrada, no es error
		}
		return nil, fmt.Errorf("redis: error al cargar sesión %s: %w", sessionID, err)
	}

	var session models.SessionContext
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("redis: error al deserializar sesión %s: %w", sessionID, err)
	}

	return &session, nil
}

// Save persiste el contexto de sesión en Redis con serialización JSON.
// Respeta el TTL definido en SessionContext; si es 0, usa el TTL por defecto.
func (m *MemoryStore) Save(ctx context.Context, sessionID string, session *models.SessionContext) error {
	key := keyPrefix + sessionID

	session.UpdatedAt = time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("redis: error al serializar sesión %s: %w", sessionID, err)
	}

	ttl := defaultTTL
	if session.TTL > 0 {
		ttl = time.Duration(session.TTL) * time.Second
	}

	if err := m.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis: error al guardar sesión %s: %w", sessionID, err)
	}

	return nil
}

// Delete elimina completamente el contexto de sesión de Redis.
func (m *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	key := keyPrefix + sessionID

	if err := m.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: error al eliminar sesión %s: %w", sessionID, err)
	}

	return nil
}

// Close cierra la conexión con Redis de forma limpia.
func (m *MemoryStore) Close() error {
	return m.client.Close()
}
