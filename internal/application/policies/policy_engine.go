package policies

import (
	"fmt"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// Engine es el motor de políticas que valida si un usuario tiene permisos
// para ejecutar una herramienta en particular.
type Engine struct{}

// NewEngine crea un nuevo motor de políticas.
func NewEngine() *Engine {
	return &Engine{}
}

// EvaluateTool evalúa si el usuario tiene permiso para ejecutar la herramienta solicitada.
// Compara el permiso del usuario (userPerm) contra el permiso requerido por la tool (RequiredPermission).
// REGLA: write tiene permiso de lectura implícito automáticamente.
func (e *Engine) EvaluateTool(tool ports.Tool, userPerm models.Permission) models.PolicyResult {
	required := tool.RequiredPermission()
	if models.HasPermission(userPerm, required) {
		return models.PolicyResult{
			Allowed: true,
			Reason:  fmt.Sprintf("permiso suficiente para '%s' (requiere: %s)", tool.Name(), required),
		}
	}

	return models.PolicyResult{
		Allowed: false,
		Reason:  fmt.Sprintf("permiso insuficiente para '%s': se requiere permiso '%s'", tool.Name(), required),
	}
}
