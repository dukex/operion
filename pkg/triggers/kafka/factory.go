package kafka

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

var (
	ErrConfigNil = errors.New("config cannot be nil")
)

func NewKafkaTriggerFactory() protocol.TriggerFactory {
	return &KafkaTriggerFactory{}
}

type KafkaTriggerFactory struct{}

func (f *KafkaTriggerFactory) ID() string {
	return "kafka"
}

func (f *KafkaTriggerFactory) Name() string {
	return "Kafka"
}

func (f *KafkaTriggerFactory) Description() string {
	return "Trigger workflow execution when messages are received on Kafka topics"
}

func (f *KafkaTriggerFactory) Schema() map[string]any {
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

func (f *KafkaTriggerFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, ErrConfigNil
	}

	trigger, err := NewKafkaTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka trigger: %w", err)
	}

	return trigger, nil
}
