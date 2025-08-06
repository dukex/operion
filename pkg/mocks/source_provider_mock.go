package mocks

import (
	"context"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/stretchr/testify/mock"
)

// MockSourceProvider is a mock implementation of protocol.SourceProvider interface.
type MockSourceProvider struct {
	mock.Mock
}

func (m *MockSourceProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	args := m.Called(ctx, callback)

	return args.Error(0)
}

func (m *MockSourceProvider) Stop(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockSourceProvider) Validate() error {
	args := m.Called()

	return args.Error(0)
}
