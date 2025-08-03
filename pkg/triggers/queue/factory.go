package queue

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

var (
	ErrConfigNil = errors.New("config cannot be nil")
)

func NewQueueTriggerFactory() protocol.TriggerFactory {
	return &QueueTriggerFactory{}
}

type QueueTriggerFactory struct{}

func (f *QueueTriggerFactory) ID() string {
	return "queue"
}

func (f *QueueTriggerFactory) Name() string {
	return "Queue"
}

func (f *QueueTriggerFactory) Description() string {
	return "Trigger workflow execution based on message queue events"
}

func (f *QueueTriggerFactory) Schema() map[string]any {
	return map[string]any{
		"type":        "object",
		"title":       "Queue Trigger Configuration",
		"description": "Configuration for message queue-based workflow triggering",
		"properties": map[string]any{
			"provider": map[string]any{
				"type":        "string",
				"description": "Queue provider type",
				"enum":        []string{"redis"},
				"default":     "redis",
				"examples":    []string{"redis"},
			},
			"queue": map[string]any{
				"type":        "string",
				"description": "Name of the message queue to monitor",
				"examples":    []string{"orders", "notifications", "background-tasks"},
			},
			"consumer_group": map[string]any{
				"type":        "string",
				"description": "Consumer group for message processing (optional)",
				"examples":    []string{"operion-workers", "order-processors", "notification-handlers"},
			},
			"connection": map[string]any{
				"type":        "object",
				"description": "Connection configuration for the queue provider",
				"properties": map[string]any{
					"addr": map[string]any{
						"type":        "string",
						"description": "Queue server address",
						"default":     "localhost:6379",
						"examples":    []string{"localhost:6379", "redis.example.com:6379", "10.0.0.1:6379"},
					},
					"password": map[string]any{
						"type":        "string",
						"description": "Authentication password (optional)",
						"examples":    []string{"secret-password", ""},
					},
					"db": map[string]any{
						"type":        "string",
						"description": "Database number for Redis (optional)",
						"examples":    []string{"0", "1", "15"},
					},
				},
				"examples": []map[string]any{
					{"addr": "localhost:6379"},
					{"addr": "redis.example.com:6379", "password": "secret", "db": "0"},
				},
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "Whether this queue trigger is active",
				"default":     true,
				"examples":    []bool{true, false},
			},
		},
		"required": []string{"queue"},
		"examples": []map[string]any{
			{
				"provider":   "redis",
				"queue":      "orders",
				"connection": map[string]string{"addr": "localhost:6379"},
			},
			{
				"provider":       "redis",
				"queue":          "notifications",
				"consumer_group": "notification-workers",
				"connection":     map[string]string{"addr": "redis.example.com:6379", "password": "secret"},
				"enabled":        true,
			},
		},
	}
}

func (f *QueueTriggerFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, ErrConfigNil
	}

	trigger, err := NewQueueTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue trigger: %w", err)
	}

	return trigger, nil
}
