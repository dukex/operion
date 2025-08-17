// Package eventbus provides specialized event bus for source events.
package eventbus

import (
	"context"

	"github.com/dukex/operion/pkg/events"
)

// SourceEventHandler is called when a source event is received.
type SourceEventHandler func(ctx context.Context, sourceEvent *events.SourceEvent) error

// SourceEventPublisher publishes source events.
type SourceEventPublisher interface {
	PublishSourceEvent(ctx context.Context, sourceEvent *events.SourceEvent) error
}

// SourceEventSubscriber subscribes to source events.
type SourceEventSubscriber interface {
	HandleSourceEvents(handler SourceEventHandler) error
	SubscribeToSourceEvents(ctx context.Context) error
}

// SourceEventBus combines publishing and subscribing for source events.
type SourceEventBus interface {
	SourceEventPublisher
	SourceEventSubscriber
	Close() error
}
