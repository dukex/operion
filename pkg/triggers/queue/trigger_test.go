package queue

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueTrigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_redis_config",
			config: map[string]any{
				"provider":       "redis",
				"queue":          "test_queue",
				"consumer_group": "test_group",
				"connection": map[string]any{
					"addr":     "localhost:6379",
					"password": "",
					"db":       "0",
				},
			},
			expectError: false,
		},
		{
			name: "missing_queue",
			config: map[string]any{
				"provider": "redis",
			},
			expectError: true,
			errorMsg:    "queue trigger queue name is required",
		},
		{
			name: "without_id",
			config: map[string]any{
				"provider": "redis",
				"queue":    "test_queue",
			},
			expectError: false,
		},
		{
			name: "unsupported_provider",
			config: map[string]any{
				"provider": "rabbitmq",
				"queue":    "test_queue",
			},
			expectError: true,
			errorMsg:    "unsupported queue provider: rabbitmq",
		},
		{
			name: "default_provider",
			config: map[string]any{
				"queue": "test_queue",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewQueueTrigger(tt.config, logger)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, trigger)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, trigger)
				assert.Equal(t, tt.config["queue"], trigger.Queue)

				if tt.config["provider"] == nil {
					assert.Equal(t, "redis", trigger.Provider)
				} else {
					assert.Equal(t, tt.config["provider"], trigger.Provider)
				}
			}
		})
	}
}

func TestQueueTriggerFactory(t *testing.T) {
	factory := NewQueueTriggerFactory()

	assert.Equal(t, "queue", factory.ID())

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := map[string]any{
		"id":    "test-queue-trigger",
		"queue": "test_queue",
	}

	trigger, err := factory.Create(config, logger)
	require.NoError(t, err)
	assert.NotNil(t, trigger)
}

func TestQueueTriggerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	validConfig := map[string]any{
		"id":    "test-queue-trigger",
		"queue": "test_queue",
	}

	trigger, err := NewQueueTrigger(validConfig, logger)
	require.NoError(t, err)

	err = trigger.Validate()
	assert.NoError(t, err)
}
