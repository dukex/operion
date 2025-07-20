// Package eventbus provides event-driven communication infrastructure for workflow orchestration.
package eventbus

import (
	"context"

	"github.com/dukex/operion/pkg/events"
)

type Event interface {
	GetType() events.EventType
}

type EventPublisher interface {
	Publish(ctx context.Context, key string, event Event) error
}

type EventSubscriber interface {
	Handle(eventType events.EventType, handler EventHandler) error
	Subscribe(ctx context.Context) error
}

type EventHandler func(ctx context.Context, event interface{}) error

type EventBus interface {
	EventPublisher
	EventSubscriber
	Close() error
	GenerateID() string
}
