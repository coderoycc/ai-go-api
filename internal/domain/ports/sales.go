package ports

import (
	"context"
	"time"
)

// Sale representa una venta del microservicio de Ventas.
type Sale struct {
	// ID es el identificador único de la venta.
	ID string `json:"id"`
	// ProductID es el identificador del producto vendido.
	ProductID string `json:"product_id"`
	// Quantity es la cantidad de unidades vendidas.
	Quantity int `json:"quantity"`
	// TotalPrice es el precio total de la venta.
	TotalPrice float64 `json:"total_price"`
	// CustomerName es el nombre del cliente.
	CustomerName string `json:"customer_name,omitempty"`
	// Status es el estado actual de la venta (pending, completed, cancelled).
	Status string `json:"status"`
	// CreatedAt marca cuándo se registró la venta.
	CreatedAt time.Time `json:"created_at"`
}

// CreateSaleRequest encapsula los datos necesarios para crear una nueva venta.
type CreateSaleRequest struct {
	// ProductID es el identificador del producto a vender.
	ProductID string `json:"product_id"`
	// Quantity es la cantidad de unidades a vender.
	Quantity int `json:"quantity"`
	// CustomerName es el nombre del cliente (opcional).
	CustomerName string `json:"customer_name,omitempty"`
}

// SalesClient define el contrato para comunicarse con el microservicio de Ventas.
// Los adaptadores implementan esta interfaz para conectarse via HTTP, gRPC u otro protocolo.
type SalesClient interface {
	// CreateSale registra una nueva venta en el sistema.
	// Retorna la venta creada y error si la operación falla.
	CreateSale(ctx context.Context, req CreateSaleRequest) (*Sale, error)

	// GetSaleByID consulta una venta existente por su identificador.
	// Retorna error si la venta no existe.
	GetSaleByID(ctx context.Context, id string) (*Sale, error)

	// CancelSale cancela una venta existente por su identificador.
	// Retorna error si la venta no existe o no puede cancelarse.
	CancelSale(ctx context.Context, id string) error
}
