package ports

import "context"

// SearchProductsRequest encapsula los filtros combinables del body del endpoint
// POST /api/productos/buscar. Todos los campos son opcionales.
type SearchProductsRequest struct {
	Codigo        string   `json:"codigo,omitempty"`
	Nombre        string   `json:"nombre,omitempty"`
	PalabrasClave []string `json:"palabras_clave,omitempty"`
	Etiquetas     []string `json:"etiquetas,omitempty"`
	Marca         []string `json:"marca,omitempty"`
	Tipo          []string `json:"tipo,omitempty"`
	Categoria     []string `json:"categoria,omitempty"`
	PrecioMin     float64  `json:"precio_min,omitempty"`
	PrecioMax     float64  `json:"precio_max,omitempty"`
	Orden         string   `json:"orden,omitempty"`
	Pagina        int      `json:"pagina,omitempty"`
	Limite        int      `json:"limite,omitempty"`
}

// ProductSearchResponse es el wrapper que devuelve el endpoint de búsqueda.
type ProductSearchResponse struct {
	Total            int            `json:"total"`
	Pagina           int            `json:"pagina"`
	Limite           int            `json:"limite"`
	FiltrosAplicados map[string]any `json:"filtros_aplicados"`
	Productos        []Product      `json:"productos"`
}

// Product representa un item del array "productos" devuelto por la API.
type Product struct {
	Codigo       string   `json:"codigo"`
	Nombre       string   `json:"nombre"`
	Marca        string   `json:"marca"`
	Categoria    string   `json:"categoria"`
	Precio       float64  `json:"precio"`
	Tipo         string   `json:"tipo"`
	Stock        int      `json:"stock"`
	Descripcion  string   `json:"descripcion"`
	Etiquetas    []string `json:"etiquetas"`
	PalabrasClave []string `json:"palabras_clave"`
	Imagen       string   `json:"imagen"`
}

// ProductClient define el contrato para comunicarse con la API externa.
// Los adaptadores implementan esta interfaz para conectarse via HTTP, gRPC u otro protocolo.
type ProductClient interface {
	// SearchProducts consulta la API de búsqueda de productos con los filtros
	// dados. Retorna el wrapper de resultados y error si la consulta falla.
	SearchProducts(ctx context.Context, req SearchProductsRequest) (*ProductSearchResponse, error)
}
