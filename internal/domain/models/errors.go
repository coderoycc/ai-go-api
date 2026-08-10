package models

import "errors"

var (
	// ErrInvalidIntent indica que la intención detectada es desconocida o no válida.
	ErrInvalidIntent = errors.New("intención de mensaje no válida o no reconocida")

	// ErrPermissionDenied indica que el permiso del usuario (read/write) no satisface el permiso de la herramienta.
	ErrPermissionDenied = errors.New("permiso insuficiente para ejecutar la operación solicitada")

	// ErrToolNotFound indica que no se identificó ni encontró una herramienta válida para ejecutar.
	ErrToolNotFound = errors.New("herramienta no identificada o no registrada")

	// ErrToolExecutionFailed indica un fallo durante la ejecución de la herramienta en el microservicio.
	ErrToolExecutionFailed = errors.New("error al ejecutar la herramienta en el servicio externo")
)
