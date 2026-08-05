package ports

import "context"

// Product representa un producto del microservicio de Productos.
type Product struct {
	// ID es el identificador único del producto.
	ID string `json:"id"`
	// Name es el nombre del producto.
	Name string `json:"name"`
	// Description es la descripción detallada del producto.
	Description string `json:"description,omitempty"`
	// Price es el precio unitario del producto.
	Price float64 `json:"price"`
	// Stock es la cantidad disponible en inventario.
	Stock int `json:"stock"`
	// Category es la categoría del producto.
	Category string `json:"category,omitempty"`
}

// ProductClient define el contrato para comunicarse con el microservicio de Productos.
// Los adaptadores implementan esta interfaz para conectarse via HTTP, gRPC u otro protocolo.
type ProductClient interface {
	// SearchProducts busca productos que coincidan con el término de búsqueda.
	// Retorna una lista de productos y error si la consulta falla.
	SearchProducts(ctx context.Context, query string) ([]Product, error)

	// GetProductByID obtiene un producto específico por su identificador.
	// Retorna error si el producto no existe.
	GetProductByID(ctx context.Context, id string) (*Product, error)

	// CheckStock verifica la disponibilidad de stock de un producto.
	// Retorna la cantidad disponible y error si la consulta falla.
	CheckStock(ctx context.Context, productID string) (int, error)
}
