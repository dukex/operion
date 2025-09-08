package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/eventbus/kafka"
)

func NewEventBus(ctx context.Context, provider string, logger *slog.Logger) (eventbus.EventBus, error) {
	switch provider {
	case "kafka":
		return kafka.NewEventBus(ctx, logger)
	default:
		return nil, fmt.Errorf("unsupported event bus provider: %s", provider)
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
