package cmd

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/dukex/operion/pkg/channels/kafka"
	"github.com/dukex/operion/pkg/event_bus"
)

func NewEventBus(provider string, logger *slog.Logger) event_bus.EventBus {
	switch provider {
	case "kafka":
		pub, sub, err := kafka.CreateChannel(watermill.NewSlogLogger(logger), "operion-trigger")

		if err != nil {
			panic(fmt.Errorf("failed to create Kafka pub/sub: %w", err))
		}

		return event_bus.NewWatermillEventBus(pub, sub)
	default:
		panic("Unsupported event bus provider: " + provider)
	}
}
