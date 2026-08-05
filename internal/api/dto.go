package api

// ChatHTTPRequest representa la carga útil recibida desde el frontend o app móvil.
type ChatHTTPRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

// HealthResponse representa el estado de salud del servicio.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}
