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

func (m *MockProductClient) SearchProducts(ctx context.Context, req ports.SearchProductsRequest) (*ports.ProductSearchResponse, error) {
	args := m.Called(ctx, req)
	if res, ok := args.Get(0).(*ports.ProductSearchResponse); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}
