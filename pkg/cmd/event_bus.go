package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/eventbus/kafka"
)

func NewEventBus(ctx context.Context, logger *slog.Logger, provider string) eventbus.EventBus {
	switch provider {
	case "kafka":
		eventBus, err := kafka.NewEventBus(ctx, logger)
		if err != nil {
			panic(fmt.Errorf("failed to create Kafka pub/sub: %w", err))
		}

		return eventBus
	default:
		panic("Unsupported event bus provider: " + provider)
	}
}
