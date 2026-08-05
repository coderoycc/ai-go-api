package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
)

// ProductSearchTool implementa ports.Tool para buscar productos.
type ProductSearchTool struct {
	client ports.ProductClient
}

func NewProductSearchTool(client ports.ProductClient) *ProductSearchTool {
	return &ProductSearchTool{client: client}
}

func (t *ProductSearchTool) Name() string {
	return "search_products"
}

func (t *ProductSearchTool) Description() string {
	return "Busca productos disponibles en el catálogo por nombre, categoría o término clave."
}

func (t *ProductSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "El término de búsqueda o nombre del producto a consultar.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *ProductSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos para search_products: %w", err)
	}

	products, err := t.client.SearchProducts(ctx, args.Query)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(products)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ProductGetTool implementa ports.Tool para obtener un producto por ID.
type ProductGetTool struct {
	client ports.ProductClient
}

func NewProductGetTool(client ports.ProductClient) *ProductGetTool {
	return &ProductGetTool{client: client}
}

func (t *ProductGetTool) Name() string {
	return "get_product"
}

func (t *ProductGetTool) Description() string {
	return "Obtiene el detalle completo de un producto por su ID."
}

func (t *ProductGetTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Identificador único del producto.",
			},
		},
		"required": []string{"id"},
	}
}

func (t *ProductGetTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos para get_product: %w", err)
	}

	product, err := t.client.GetProductByID(ctx, args.ID)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(product)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// StockCheckTool implementa ports.Tool para verificar stock.
type StockCheckTool struct {
	client ports.ProductClient
}

func NewStockCheckTool(client ports.ProductClient) *StockCheckTool {
	return &StockCheckTool{client: client}
}

func (t *StockCheckTool) Name() string {
	return "check_stock"
}

func (t *StockCheckTool) Description() string {
	return "Verifica el stock disponible en inventario para un producto específico."
}

func (t *StockCheckTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"product_id": map[string]interface{}{
				"type":        "string",
				"description": "ID del producto a verificar stock.",
			},
		},
		"required": []string{"product_id"},
	}
}

func (t *StockCheckTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		ProductID string `json:"product_id"`
		Query     string `json:"query"`
		ID        string `json:"id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos para check_stock: %w", err)
	}

	productID := args.ProductID
	if productID == "" {
		productID = args.ID
	}
	if productID == "" {
		productID = args.Query
	}

	stock, err := t.client.CheckStock(ctx, productID)
	if err != nil {
		return "", err
	}

	res := map[string]interface{}{
		"product_id": productID,
		"stock":      stock,
	}
	data, _ := json.Marshal(res)
	return string(data), nil
}

// SaleCreateTool implementa ports.Tool para registrar una venta.
type SaleCreateTool struct {
	client ports.SalesClient
}

func NewSaleCreateTool(client ports.SalesClient) *SaleCreateTool {
	return &SaleCreateTool{client: client}
}

func (t *SaleCreateTool) Name() string {
	return "create_sale"
}

func (t *SaleCreateTool) Description() string {
	return "Crea/Registra una orden de compra o venta en el sistema."
}

func (t *SaleCreateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"product_id": map[string]interface{}{
				"type":        "string",
				"description": "ID del producto a comprar.",
			},
			"quantity": map[string]interface{}{
				"type":        "integer",
				"description": "Cantidad de unidades.",
			},
			"customer_name": map[string]interface{}{
				"type":        "string",
				"description": "Nombre del cliente comprador (opcional).",
			},
		},
		"required": []string{"product_id", "quantity"},
	}
}

func (t *SaleCreateTool) Execute(ctx context.Context, arguments string) (string, error) {
	var req ports.CreateSaleRequest
	if err := json.Unmarshal([]byte(arguments), &req); err != nil {
		return "", fmt.Errorf("argumentos inválidos para create_sale: %w", err)
	}

	sale, err := t.client.CreateSale(ctx, req)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(sale)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaleGetTool implementa ports.Tool para consultar una venta.
type SaleGetTool struct {
	client ports.SalesClient
}

func NewSaleGetTool(client ports.SalesClient) *SaleGetTool {
	return &SaleGetTool{client: client}
}

func (t *SaleGetTool) Name() string {
	return "get_sale"
}

func (t *SaleGetTool) Description() string {
	return "Consulta el estado y detalle de una venta por su ID."
}

func (t *SaleGetTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "ID de la venta.",
			},
		},
		"required": []string{"id"},
	}
}

func (t *SaleGetTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos para get_sale: %w", err)
	}

	sale, err := t.client.GetSaleByID(ctx, args.ID)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(sale)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaleCancelTool implementa ports.Tool para cancelar una venta.
type SaleCancelTool struct {
	client ports.SalesClient
}

func NewSaleCancelTool(client ports.SalesClient) *SaleCancelTool {
	return &SaleCancelTool{client: client}
}

func (t *SaleCancelTool) Name() string {
	return "cancel_sale"
}

func (t *SaleCancelTool) Description() string {
	return "Cancela una venta o pedido existente por su ID."
}

func (t *SaleCancelTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "ID de la venta a cancelar.",
			},
		},
		"required": []string{"id"},
	}
}

func (t *SaleCancelTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos para cancel_sale: %w", err)
	}

	if err := t.client.CancelSale(ctx, args.ID); err != nil {
		return "", err
	}

	res := map[string]interface{}{
		"status":  "cancelled",
		"sale_id": args.ID,
		"message": "Venta cancelada exitosamente",
	}
	data, _ := json.Marshal(res)
	return string(data), nil
}
