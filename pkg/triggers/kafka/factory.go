package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/operion-flow/interfaces"
)

type TriggerFactory struct{}

func NewTriggerFactory() interfaces.TriggerFactory {
	return &TriggerFactory{}
}

func (f *TriggerFactory) ID() string {
	return "kafka"
}

func (f *TriggerFactory) Name() string {
	return "Kafka"
}

func (f *TriggerFactory) Description() string {
	return "Trigger workflow execution when messages are received on Kafka topics"
}

// nolint: lll
func (f *TriggerFactory) Schema() map[string]any {
	return map[string]any{
		"type":        "object",
		"title":       "Kafka Trigger Configuration",
		"description": "Configuration for Kafka topic message triggering",
		"properties": map[string]any{
			"topic": map[string]any{
				"type":        "string",
				"description": "The Kafka topic name to subscribe to",
				"examples": []string{
					"user-events",
					"orders",
					"notifications",
					"system-logs",
				},
			},
			"consumer_group": map[string]any{
				"type":        "string",
				"description": "Kafka consumer group ID (auto-generated if not provided using format: operion-triggers-{trigger_id})",
				"examples": []string{
					"operion-order-processor",
					"operion-user-events",
					"custom-consumer-group",
				},
			},
			"brokers": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of Kafka broker addresses (uses KAFKA_BROKERS env var if not provided)",
				"examples": []string{
					"kafka1:9092,kafka2:9092",
					"localhost:9092",
					"kafka1.example.com:9092,kafka2.example.com:9092",
				},
			},
		},
		"required": []string{"topic"},
		"examples": []map[string]any{
			{
				"topic": "user-events",
			},
			{
				"topic":          "orders",
				"consumer_group": "operion-order-processor",
			},
			{
				"topic":          "notifications",
				"consumer_group": "operion-notification-service",
				"brokers":        "kafka1.example.com:9092,kafka2.example.com:9092",
			},
		},
	}
}

func (f *TriggerFactory) Create(
	ctx context.Context,
	config map[string]any,
	logger *slog.Logger,
) (interfaces.Trigger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	trigger, err := NewTrigger(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka trigger: %w", err)
	}

	return trigger, nil
}
