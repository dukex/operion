package kafka

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/protocol"
)

func TestNewKafkaTrigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name      string
		config    map[string]any
		wantError bool
	}{
		{
			name: "valid basic config",
			config: map[string]any{
				"topic": "test-topic",
			},
			wantError: false,
		},
		{
			name: "valid config with consumer group",
			config: map[string]any{
				"topic":          "test-topic",
				"consumer_group": "test-group",
			},
			wantError: false,
		},
		{
			name: "valid config with brokers",
			config: map[string]any{
				"topic":   "test-topic",
				"brokers": "localhost:9092,localhost:9093",
			},
			wantError: false,
		},
		{
			name:      "missing topic",
			config:    map[string]any{},
			wantError: true,
		},
		{
			name: "empty topic",
			config: map[string]any{
				"topic": "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewKafkaTrigger(tt.config, logger)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewKafkaTrigger() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("NewKafkaTrigger() unexpected error: %v", err)

				return
			}

			if trigger == nil {
				t.Errorf("NewKafkaTrigger() returned nil trigger")

				return
			}

			// Verify basic properties
			if trigger.Topic != tt.config["topic"].(string) {
				t.Errorf("NewKafkaTrigger() topic = %v, want %v", trigger.Topic, tt.config["topic"])
			}
		})
	}
}

func TestKafkaTriggerValidate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name      string
		trigger   *KafkaTrigger
		wantError bool
	}{
		{
			name: "valid trigger",
			trigger: &KafkaTrigger{
				Topic:   "test-topic",
				Brokers: []string{"localhost:9092"},
				logger:  logger,
			},
			wantError: false,
		},
		{
			name: "empty topic",
			trigger: &KafkaTrigger{
				Topic:   "",
				Brokers: []string{"localhost:9092"},
				logger:  logger,
			},
			wantError: true,
		},
		{
			name: "empty brokers",
			trigger: &KafkaTrigger{
				Topic:   "test-topic",
				Brokers: []string{},
				logger:  logger,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()

			if tt.wantError && err == nil {
				t.Errorf("Validate() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestKafkaTriggerEnvironmentBrokers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with environment variable
	os.Setenv("KAFKA_BROKERS", "env-broker1:9092,env-broker2:9092")
	defer os.Unsetenv("KAFKA_BROKERS")

	config := map[string]any{
		"topic": "test-topic",
	}

	trigger, err := NewKafkaTrigger(config, logger)
	if err != nil {
		t.Fatalf("NewKafkaTrigger() unexpected error: %v", err)
	}

	expectedBrokers := []string{"env-broker1:9092", "env-broker2:9092"}
	if len(trigger.Brokers) != len(expectedBrokers) {
		t.Errorf("Brokers length = %v, want %v", len(trigger.Brokers), len(expectedBrokers))

		return
	}

	for i, broker := range trigger.Brokers {
		if broker != expectedBrokers[i] {
			t.Errorf("Broker[%d] = %v, want %v", i, broker, expectedBrokers[i])
		}
	}
}

func TestKafkaTriggerConsumerGroupGeneration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"topic": "test-topic",
	}

	trigger, err := NewKafkaTrigger(config, logger)
	if err != nil {
		t.Fatalf("NewKafkaTrigger() unexpected error: %v", err)
	}

	// Should generate default consumer group when not provided
	expectedPrefix := "operion-triggers-"
	if len(trigger.ConsumerGroup) < len(expectedPrefix) {
		t.Errorf("ConsumerGroup length too short: %v", trigger.ConsumerGroup)

		return
	}

	if trigger.ConsumerGroup[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("ConsumerGroup = %v, should start with %v", trigger.ConsumerGroup, expectedPrefix)
	}
}

func TestKafkaTriggerStopWithoutStart(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"topic": "test-topic",
	}

	trigger, err := NewKafkaTrigger(config, logger)
	if err != nil {
		t.Fatalf("NewKafkaTrigger() unexpected error: %v", err)
	}

	// Should be safe to stop without starting
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = trigger.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}
}

// Mock callback for testing.
func mockCallback(ctx context.Context, data map[string]any) error {
	return nil
}

func TestKafkaTriggerCallbackInterface(t *testing.T) {
	// Test that our mock callback implements the protocol.TriggerCallback interface
	var callback protocol.TriggerCallback = mockCallback

	// This should compile without errors
	_ = callback
}
