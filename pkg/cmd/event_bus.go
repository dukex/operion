package cmd

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/dukex/operion/pkg/channels/kafka"
	"github.com/dukex/operion/pkg/eventbus"
)

func NewEventBus(logger *slog.Logger) eventbus.EventBus {
	pub, sub, err := kafka.CreateChannel(watermill.NewSlogLogger(logger), "operion")

	if err != nil {
		panic(fmt.Errorf("failed to create Kafka pub/sub: %w", err))
	}

	return eventbus.NewWatermillEventBus(pub, sub)
}

// NewSourceEventBus creates a Kafka-based source event bus instance
func NewSourceEventBus(logger *slog.Logger) eventbus.SourceEventBus {
	sourceEventBus, err := eventbus.NewKafkaSourceEventBus(logger)
	if err != nil {
		panic(fmt.Errorf("failed to create Kafka source event bus: %w", err))
	}
	return sourceEventBus
}
