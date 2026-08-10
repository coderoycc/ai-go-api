package models

// Permission representa el nivel de acceso del usuario extraído de los headers HTTP.
type Permission string

const (
	// PermissionRead permite consultar y buscar información (lectura).
	PermissionRead Permission = "read"
	// PermissionWrite permite crear y modificar registros (escritura).
	// Un usuario con write tiene acceso implícito a todas las operaciones de read.
	PermissionWrite Permission = "write"
)

// HasPermission verifica si el permiso del usuario satisface el permiso requerido.
// REGLA: write tiene permiso de lectura automáticamente.
func HasPermission(userPerm Permission, required Permission) bool {
	if userPerm == PermissionWrite {
		return true // write cubre todo (read y write)
	}
	return userPerm == required
}

// PolicyResult es el resultado de evaluar si una acción está permitida.
type PolicyResult struct {
	// Allowed indica si la acción puede ejecutarse.
	Allowed bool `json:"allowed"`
	// Reason explica el motivo de la decisión.
	Reason string `json:"reason,omitempty"`
}
