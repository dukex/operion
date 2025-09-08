package mocks

import (
	"context"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/stretchr/testify/mock"
)

// MockEventBus is a mock implementation of eventbus.EventBus interface.
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, key string, event eventbus.Event) error {
	args := m.Called(ctx, key, event)

	return args.Error(0)
}

func (m *MockEventBus) Handle(ctx context.Context, eventType events.EventType, handler eventbus.EventHandler) error {
	args := m.Called(eventType, handler)

	return args.Error(0)
}

func (m *MockEventBus) Subscribe(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockEventBus) Close(ctx context.Context) error {
	args := m.Called()

	return args.Error(0)
}

func (m *MockEventBus) GenerateID(ctx context.Context) string {
	args := m.Called()

	return args.String(0)
}

// MockSourceEventBus is a mock implementation of eventbus.SourceEventBus interface.
type MockSourceEventBus struct {
	mock.Mock
}

func (m *MockSourceEventBus) PublishSourceEvent(ctx context.Context, sourceEvent *events.SourceEvent) error {
	args := m.Called(ctx, sourceEvent)

	return args.Error(0)
}

func (m *MockSourceEventBus) HandleSourceEvents(handler eventbus.SourceEventHandler) error {
	args := m.Called(handler)

	return args.Error(0)
}

func (m *MockSourceEventBus) SubscribeToSourceEvents(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockSourceEventBus) Close() error {
	args := m.Called()

	return args.Error(0)
}
