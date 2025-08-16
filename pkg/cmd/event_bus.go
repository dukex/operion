package cmd

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/dukex/operion/pkg/channels/kafka"
	"github.com/dukex/operion/pkg/eventbus"
)

func NewEventBus(provider string, logger *slog.Logger) eventbus.EventBus {
	switch provider {
	case "kafka":
		pub, sub, err := kafka.CreateChannel(watermill.NewSlogLogger(logger), "operion")
		if err != nil {
			panic(fmt.Errorf("failed to create Kafka pub/sub: %w", err))
		}

		return eventbus.NewWatermillEventBus(pub, sub)
	default:
		panic("Unsupported event bus provider: " + provider)
	}
}

// NewSourceEventBus creates a source event bus instance based on the provider.
func NewSourceEventBus(provider string, logger *slog.Logger) eventbus.SourceEventBus {
	switch provider {
	case "kafka":
		sourceEventBus, err := eventbus.NewKafkaSourceEventBus(logger)
		if err != nil {
			panic(fmt.Errorf("failed to create Kafka source event bus: %w", err))
		}

		return sourceEventBus
	default:
		panic("Unsupported source event bus provider: " + provider)
	}
}
