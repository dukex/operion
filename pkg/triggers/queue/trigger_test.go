package queue_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/triggers/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueTrigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name             string
		config           map[string]any
		expectedProvider queue.Provider
		expectError      bool
		errorMsg         string
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
			expectedProvider: queue.RedisProvider,
			expectError:      false,
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
			expectedProvider: queue.RedisProvider,
			expectError:      false,
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
			expectedProvider: queue.RedisProvider,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := queue.NewTrigger(t.Context(), tt.config, logger)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, trigger)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, trigger)
				assert.Equal(t, tt.config["queue"], trigger.Queue)

				assert.Equal(t, tt.expectedProvider, trigger.Provider)
			}
		})
	}
}

func TestQueueTriggerFactory(t *testing.T) {
	factory := queue.NewTriggerFactory()

	assert.Equal(t, "queue", factory.ID())

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := map[string]any{
		"id":    "test-queue-trigger",
		"queue": "test_queue",
	}

	trigger, err := factory.Create(t.Context(), config, logger)
	require.NoError(t, err)
	assert.NotNil(t, trigger)
}

func TestQueueTriggerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	validConfig := map[string]any{
		"id":    "test-queue-trigger",
		"queue": "test_queue",
	}

	trigger, err := queue.NewTrigger(t.Context(), validConfig, logger)
	require.NoError(t, err)

	err = trigger.Validate(t.Context())
	assert.NoError(t, err)
}
