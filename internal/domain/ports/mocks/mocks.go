package mocks

import (
	"context"

	"github.com/coderoycc/ai-go-api/internal/domain/models"
	"github.com/stretchr/testify/mock"
)

// MockLLM es una implementación mock del puerto ports.LLM para pruebas unitarias.
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) Chat(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(models.ChatResponse), args.Error(1)
}

// MockMemory es una implementación mock del puerto ports.Memory.
type MockMemory struct {
	mock.Mock
}

func (m *MockMemory) Load(ctx context.Context, sessionID string) (*models.SessionContext, error) {
	args := m.Called(ctx, sessionID)
	if session, ok := args.Get(0).(*models.SessionContext); ok {
		return session, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMemory) Save(ctx context.Context, sessionID string, session *models.SessionContext) error {
	args := m.Called(ctx, sessionID, session)
	return args.Error(0)
}

func (m *MockMemory) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}
