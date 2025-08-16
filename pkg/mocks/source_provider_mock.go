package mocks

import (
	"context"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of protocol.Provider interface.
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	args := m.Called(ctx, callback)

	return args.Error(0)
}

func (m *MockProvider) Stop(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockProvider) Validate() error {
	args := m.Called()

	return args.Error(0)
}
