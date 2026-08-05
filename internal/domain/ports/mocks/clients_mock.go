package mocks

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/ports"
	"github.com/stretchr/testify/mock"
)

// MockProductClient implementa ports.ProductClient para pruebas.
type MockProductClient struct {
	mock.Mock
}

func (m *MockProductClient) SearchProducts(ctx context.Context, query string) ([]ports.Product, error) {
	args := m.Called(ctx, query)
	if products, ok := args.Get(0).([]ports.Product); ok {
		return products, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductClient) GetProductByID(ctx context.Context, id string) (*ports.Product, error) {
	args := m.Called(ctx, id)
	if product, ok := args.Get(0).(*ports.Product); ok {
		return product, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductClient) CheckStock(ctx context.Context, productID string) (int, error) {
	args := m.Called(ctx, productID)
	return args.Int(0), args.Error(1)
}

// MockSalesClient implementa ports.SalesClient para pruebas.
type MockSalesClient struct {
	mock.Mock
}

func (m *MockSalesClient) CreateSale(ctx context.Context, req ports.CreateSaleRequest) (*ports.Sale, error) {
	args := m.Called(ctx, req)
	if sale, ok := args.Get(0).(*ports.Sale); ok {
		return sale, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSalesClient) GetSaleByID(ctx context.Context, id string) (*ports.Sale, error) {
	args := m.Called(ctx, id)
	if sale, ok := args.Get(0).(*ports.Sale); ok {
		return sale, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSalesClient) CancelSale(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
