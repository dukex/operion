package kafka

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/events"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTc "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaEventBus_Integration(t *testing.T) {
	ctx := t.Context()

	kafkaContainer, err := kafkaTc.Run(ctx,
		"confluentinc/cp-kafka:7.4.0",
		testcontainers.WithEnv(map[string]string{
			"TOPIC_AUTO_CREATE": "true",
		}),
		kafkaTc.WithClusterID("test-cluster"),
	)

	require.NoError(t, err)
	defer func() {
		assert.NoError(t, kafkaContainer.Terminate(ctx))
	}()

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	t.Setenv("KAFKA_BROKERS", brokers[0])
	t.Setenv("KAFKA_GROUP_ID", "test-group-"+t.Name())

	// Create the topic explicitly
	conn, err := kafkago.Dial("tcp", brokers[0])
	require.NoError(t, err)
	defer conn.Close()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:         events.Topic,
		NumPartitions: 1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("NewEventBus", func(t *testing.T) {
		eventBus, err := NewEventBus(t.Context(), logger)
		require.NoError(t, err)
		require.NotNil(t, eventBus)
		defer eventBus.Close(t.Context())
	})

	// Test GenerateID
	t.Run("GenerateID", func(t *testing.T) {
		eventBus, err := NewEventBus(ctx, logger)
		require.NoError(t, err)
		defer eventBus.Close(ctx)

		id := eventBus.GenerateID(ctx)
		assert.NotEmpty(t, id)

		id2 := eventBus.GenerateID(ctx)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id, id2)
	})

	t.Run("PublishAndHandle", func(t *testing.T) {
		publisher, err := NewEventBus(t.Context(), logger)
		require.NoError(t, err)
		defer publisher.Close(t.Context())

		subscriber, err := NewEventBus(t.Context(), logger)
	 	require.NoError(t, err)
	 	defer subscriber.Close(t.Context())

		testEvent := &events.WorkflowTriggered{
			BaseEvent: events.BaseEvent{
				ID:         publisher.GenerateID(t.Context()),
				Type:       events.WorkflowTriggeredEvent,
				Timestamp:  time.Now().UTC(),
				WorkflowID: "test-workflow-123",
				WorkerID:   "test-worker-456",
				Metadata:   map[string]any{"test": "metadata"},
			},
			TriggerID:   "test-trigger",
			TriggerData: map[string]any{"test": "trigger-data"},
		}

	 	var receivedEvent *events.WorkflowTriggered
	 	eventReceived := make(chan struct{})

	 	err = subscriber.Handle(t.Context(), events.WorkflowTriggeredEvent, func(ctx context.Context, event any) error {
	 		workflowEvent, ok := event.(*events.WorkflowTriggered)
	 		require.True(t, ok, "Expected WorkflowTriggered event")
	 		receivedEvent = workflowEvent
	 		close(eventReceived)

	 		return nil
	 	})
	 	require.NoError(t, err)

	 	err = subscriber.Subscribe(t.Context())
	 	require.NoError(t, err)

	 	time.Sleep(2 * time.Second)

		err = publisher.Publish(t.Context(), testEvent.WorkflowID, testEvent)
		require.NoError(t, err)

	 	select {
	 	case <-eventReceived:
			require.NotNil(t, receivedEvent)
	 	case <-time.After(10 * time.Second):
	 		t.Fatal("Timeout waiting for event to be received")
	 	}

		assert.Equal(t, testEvent.ID, receivedEvent.ID)
		assert.Equal(t, testEvent.Type, receivedEvent.Type)
		assert.Equal(t, testEvent.WorkflowID, receivedEvent.WorkflowID)
		assert.Equal(t, testEvent.WorkerID, receivedEvent.WorkerID)
		assert.Equal(t, testEvent.TriggerID, receivedEvent.TriggerID)
		assert.Equal(t, testEvent.TriggerData, receivedEvent.TriggerData)
		assert.Equal(t, testEvent.Metadata, receivedEvent.Metadata)
	})

	// Test multiple event types
	t.Run("MultipleEventTypes", func(t *testing.T) {
		publisher, err := NewEventBus(t.Context(), logger)
		require.NoError(t, err)
		defer publisher.Close(t.Context())

		subscriber, err := NewEventBus(t.Context(), logger)
		require.NoError(t, err)
		defer subscriber.Close(t.Context())

		// Track received events
		triggeredReceived := make(chan struct{})
		finishedReceived := make(chan struct{})
		failedReceived := make(chan struct{})

		err = subscriber.Handle(t.Context(), events.WorkflowTriggeredEvent, func(ctx context.Context, event any) error {
			close(triggeredReceived)

			return nil
		})
		require.NoError(t, err)

		err = subscriber.Handle(t.Context(), events.WorkflowFinishedEvent, func(ctx context.Context, event any) error {
			close(finishedReceived)

			return nil
		})
		require.NoError(t, err)

		err = subscriber.Handle(t.Context(), events.WorkflowFailedEvent, func(ctx context.Context, event any) error {
			close(failedReceived)

			return nil
		})
		require.NoError(t, err)

		// Start subscriber
		err = subscriber.Subscribe(t.Context())
		require.NoError(t, err)

		// Give subscriber time to start
		time.Sleep(2 * time.Second)

		// Create and publish different event types
		workflowID := "test-workflow-multi"

		triggeredEvent := &events.WorkflowTriggered{
			BaseEvent: events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
			TriggerID: "test-trigger",
		}

		finishedEvent := &events.WorkflowFinished{
			BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, workflowID),
			ExecutionID: "test-execution",
			Duration:    5 * time.Second,
		}

		failedEvent := &events.WorkflowFailed{
			BaseEvent:   events.NewBaseEvent(events.WorkflowFailedEvent, workflowID),
			ExecutionID: "test-execution",
			Error:       "test error",
			Duration:    3 * time.Second,
		}

		// Publish events
		err = publisher.Publish(ctx, workflowID, triggeredEvent)
		require.NoError(t, err)

		err = publisher.Publish(ctx, workflowID, finishedEvent)
		require.NoError(t, err)

		err = publisher.Publish(ctx, workflowID, failedEvent)
		require.NoError(t, err)

		// Wait for all events to be received
		timeout := time.After(15 * time.Second)

		select {
		case <-triggeredReceived:
			assert.NotNil(t, triggeredEvent)
		case <-timeout:
			t.Fatal("Timeout waiting for WorkflowTriggered event")
		}

		select {
		case <-finishedReceived:
			assert.NotNil(t, finishedEvent)
		case <-timeout:
			t.Fatal("Timeout waiting for WorkflowFinished event")
		}

		select {
		case <-failedReceived:
			assert.NotNil(t, failedEvent)
		case <-timeout:
			t.Fatal("Timeout waiting for WorkflowFailed event")
		}
	})

	// // Test error handling
	// t.Run("HandlerError", func(t *testing.T) {
	// 	publisher, err := NewEventBus(ctx, logger)
	// 	require.NoError(t, err)
	// 	defer publisher.Close(ctx)

	// 	subscriber, err := NewEventBus(ctx, logger)
	// 	require.NoError(t, err)
	// 	defer subscriber.Close(ctx)

	// 	// Set up handler that returns an error
	// 	handlerCalled := make(chan struct{})
	// 	err = subscriber.Handle(ctx, events.WorkflowTriggeredEvent, func(ctx context.Context, event any) error {
	// 		close(handlerCalled)
	// 		return assert.AnError // Return an error
	// 	})
	// 	require.NoError(t, err)

	// 	// Start subscriber
	// 	err = subscriber.Subscribe(ctx)
	// 	require.NoError(t, err)

	// 	// Give subscriber time to start
	// 	time.Sleep(1 * time.Second)

	// 	// Publish event
	// 	testEvent := &events.WorkflowTriggered{
	// 		BaseEvent: events.NewBaseEvent(events.WorkflowTriggeredEvent, "test-workflow"),
	// 		TriggerID: "test-trigger",
	// 	}

	// 	err = publisher.Publish(ctx, testEvent.WorkflowID, testEvent)
	// 	require.NoError(t, err)

	// 	// Wait for handler to be called (even though it errors)
	// 	select {
	// 	case <-handlerCalled:
	// 		// Handler was called successfully
	// 	case <-time.After(10 * time.Second):
	// 		t.Fatal("Timeout waiting for handler to be called")
	// 	}
	// })
}

func TestKafkaEventBus_NewEventBusErrors(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with no KAFKA_BROKERS environment variable
	t.Run("NoBrokers", func(t *testing.T) {
		originalBrokers := os.Getenv("KAFKA_BROKERS")
		os.Unsetenv("KAFKA_BROKERS")
		defer os.Setenv("KAFKA_BROKERS", originalBrokers)

		eventBus, err := NewEventBus(ctx, logger)
		assert.Error(t, err)
		assert.Nil(t, eventBus)
		assert.Contains(t, err.Error(), "no Kafka brokers configured")
	})

	// Test with empty KAFKA_BROKERS
	t.Run("EmptyBrokers", func(t *testing.T) {
		originalBrokers := os.Getenv("KAFKA_BROKERS")
		os.Setenv("KAFKA_BROKERS", "")
		defer os.Setenv("KAFKA_BROKERS", originalBrokers)

		eventBus, err := NewEventBus(ctx, logger)
		assert.Error(t, err)
		assert.Nil(t, eventBus)
		assert.Contains(t, err.Error(), "no Kafka brokers configured")
	})
}

func TestKafkaEventBus_Close(t *testing.T) {
	ctx := context.Background()

	// Start Kafka container
	kafkaContainer, err := kafkaTc.Run(ctx,
		"confluentinc/cp-kafka:7.4.0",
		kafkaTc.WithClusterID("test-cluster"),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, kafkaContainer.Terminate(ctx))
	}()

	// Get broker connection string
	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	t.Setenv("KAFKA_BROKERS", brokers[0])

	// Create the topic explicitly
	conn, err := kafkago.Dial("tcp", brokers[0])
	require.NoError(t, err)
	defer conn.Close()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:         events.Topic,
		NumPartitions: 1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create event bus
	eventBus, err := NewEventBus(ctx, logger)
	require.NoError(t, err)
	require.NotNil(t, eventBus)

	// Close should not return error
	err = eventBus.Close(ctx)
	assert.NoError(t, err)

	// Multiple closes should not panic
	err = eventBus.Close(ctx)
	assert.NoError(t, err)
}