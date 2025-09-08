package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTc "github.com/testcontainers/testcontainers-go/modules/kafka"
)

var (
	kafkaContainer *kafkaTc.KafkaContainer
	brokers        string
	logger         *slog.Logger
)

func TestMain(m *testing.M) {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx := context.Background()

	var err error

	kafkaContainer, err = kafkaTc.Run(ctx, "confluentinc/confluent-local:7.7.0", testcontainers.WithEnv(map[string]string{
		"KAFKA_CREATE_TOPICS": "true",
	}))
	if err != nil {
		panic("Failed to start Kafka container: " + err.Error())
	}

	kafkaBrokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		panic("Failed to get Kafka brokers: " + err.Error())
	}

	brokers = kafkaBrokers[0]

	createTopics(brokers)

	code := m.Run()

	if err := kafkaContainer.Terminate(ctx); err != nil {
		panic("Failed to terminate Kafka container: " + err.Error())
	}

	os.Exit(code)
}

func createTestEvent(eventType events.EventType) eventbus.Event {
	switch eventType {
	case events.WorkflowTriggeredEvent:
		return &events.WorkflowTriggered{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			TriggerID:   "test-trigger",
			TriggerData: map[string]any{"key": "value"},
		}
	case events.WorkflowFinishedEvent:
		return &events.WorkflowFinished{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			Result:      map[string]any{"result": "success"},
			Duration:    time.Second,
		}
	case events.WorkflowFailedEvent:
		return &events.WorkflowFailed{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			Error:       "test error",
			Duration:    time.Second,
		}
	case events.NodeActivationEvent:
		return &events.NodeActivation{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			NodeID:      "test-node",
			WorkflowID:  "test-workflow",
			InputPort:   "input",
			InputData:   map[string]any{"data": "test"},
			SourceNode:  "source-node",
			SourcePort:  "output",
		}
	case events.NodeCompletionEvent:
		return &events.NodeCompletion{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			NodeID:      "test-node",
			WorkflowID:  "test-workflow",
			Status:      models.NodeStatusSuccess,
			OutputData:  map[string]any{"output": "data"},
			DurationMs:  1000,
			CompletedAt: time.Now(),
		}
	case events.NodeExecutionFinishedEvent:
		return &events.NodeExecutionFinished{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			NodeID:      "test-node",
			OutputData:  map[string]any{"output": "data"},
			Duration:    time.Second,
		}
	case events.NodeExecutionFailedEvent:
		return &events.NodeExecutionFailed{
			BaseEvent:   events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID: "test-execution",
			NodeID:      "test-node",
			Error:       "test error",
			Duration:    time.Second,
		}
	case events.WorkflowExecutionStartedEvent:
		return &events.WorkflowExecutionStarted{
			BaseEvent:    events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:  "test-execution",
			WorkflowName: "test-workflow-name",
			TriggerType:  "webhook",
			TriggerData:  map[string]any{"data": "test"},
			Variables:    map[string]any{"var": "value"},
			Initiator:    "test-user",
		}
	case events.WorkflowExecutionCompletedEvent:
		return &events.WorkflowExecutionCompleted{
			BaseEvent:     events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:   "test-execution",
			Status:        "completed",
			DurationMs:    1000,
			NodesExecuted: 5,
			FinalResults:  map[string]any{"result": "success"},
		}
	case events.WorkflowExecutionFailedEvent:
		return &events.WorkflowExecutionFailed{
			BaseEvent:      events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:    "test-execution",
			Status:         "failed",
			DurationMs:     1000,
			Error:          events.WorkflowError{NodeID: "test-node", Message: "test error", Code: "ERROR_001"},
			NodesExecuted:  3,
			PartialResults: map[string]any{"partial": "result"},
		}
	case events.WorkflowExecutionCancelledEvent:
		return &events.WorkflowExecutionCancelled{
			BaseEvent:     events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:   "test-execution",
			Status:        "cancelled",
			DurationMs:    500,
			Reason:        "user requested",
			CancelledBy:   "test-user",
			NodesExecuted: 2,
		}
	case events.WorkflowExecutionTimeoutEvent:
		return &events.WorkflowExecutionTimeout{
			BaseEvent:      events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:    "test-execution",
			Status:         "timeout",
			DurationMs:     30000,
			TimeoutLimitMs: 30000,
			NodesExecuted:  4,
			StuckNode:      "test-node",
			PartialResults: map[string]any{"partial": "result"},
		}
	case events.WorkflowExecutionPausedEvent:
		return &events.WorkflowExecutionPaused{
			BaseEvent:    events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:  "test-execution",
			Status:       "paused",
			PauseReason:  "approval required",
			PausedAtNode: "approval-node",
			ApprovalData: map[string]any{"approval": "data"},
		}
	case events.WorkflowExecutionResumedEvent:
		return &events.WorkflowExecutionResumed{
			BaseEvent:       events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:     "test-execution",
			Status:          "running",
			ResumedBy:       "test-user",
			PauseDurationMs: 5000,
			ApprovalResult:  "approved",
		}
	case events.WorkflowVariablesUpdatedEvent:
		return &events.WorkflowVariablesUpdated{
			BaseEvent:        events.NewBaseEvent(eventType, "test-workflow"),
			ExecutionID:      "test-execution",
			UpdatedVariables: map[string]any{"var": "new-value"},
			UpdatedBy:        "test-user",
		}
	default:
		return &events.WorkflowTriggered{
			BaseEvent: events.NewBaseEvent(eventType, "test-workflow"),
		}
	}
}

func TestNewEventBus(t *testing.T) {
	tests := []struct {
		name        string
		brokers     string
		expectError bool
	}{
		{
			name:        "valid brokers",
			brokers:     brokers,
			expectError: false,
		},
		{
			name:        "empty brokers",
			brokers:     "",
			expectError: true,
		},
		{
			name:        "whitespace only brokers",
			brokers:     "localhost:9092",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KAFKA_BROKERS", tt.brokers)

			bus, err := NewEventBus(context.Background(), logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, bus)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bus)

				if bus != nil {
					err = bus.Close(context.Background())
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestKafkaEventBus_GenerateID(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	id1 := bus.GenerateID(context.Background())
	id2 := bus.GenerateID(context.Background())

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestKafkaEventBus_Handle(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	called := false
	handler := func(ctx context.Context, event any) error {
		called = true

		return nil
	}

	err = bus.Handle(context.Background(), events.WorkflowTriggeredEvent, handler)
	assert.NoError(t, err)

	kafkaBus := bus.(*kafkaEventBus)
	assert.Contains(t, kafkaBus.handlers, events.WorkflowTriggeredEvent)
	assert.False(t, called)
}

func TestKafkaEventBus_PublishAndSubscribe(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	receivedEvents := make(chan eventbus.Event, 1)
	handler := func(ctx context.Context, event any) error {
		if e, ok := event.(eventbus.Event); ok {
			receivedEvents <- e
		}

		return nil
	}

	err = bus.Handle(context.Background(), events.WorkflowTriggeredEvent, handler)
	require.NoError(t, err)

	err = bus.Subscribe(context.Background())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	testEvent := createTestEvent(events.WorkflowTriggeredEvent)
	err = bus.Publish(context.Background(), "test-key", testEvent)
	require.NoError(t, err)

	select {
	case received := <-receivedEvents:
		assert.Equal(t, testEvent.GetType(), received.GetType())
	case <-time.After(10 * time.Second):
		t.Fatal("Did not receive event within timeout")
	}
}

func TestKafkaEventBus_MultipleEventTypes(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	receivedEvents := make(chan eventbus.Event, 2)
	handler := func(ctx context.Context, event any) error {
		if e, ok := event.(eventbus.Event); ok {
			receivedEvents <- e
		}

		return nil
	}

	err = bus.Handle(context.Background(), events.WorkflowTriggeredEvent, handler)
	require.NoError(t, err)

	err = bus.Handle(context.Background(), events.NodeActivationEvent, handler)
	require.NoError(t, err)

	err = bus.Subscribe(context.Background())
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	workflowEvent := createTestEvent(events.WorkflowTriggeredEvent)
	nodeEvent := createTestEvent(events.NodeActivationEvent)

	err = bus.Publish(context.Background(), "key1", workflowEvent)
	require.NoError(t, err)

	err = bus.Publish(context.Background(), "key2", nodeEvent)
	require.NoError(t, err)

	receivedTypes := make(map[events.EventType]bool)

	for range 2 {
		select {
		case received := <-receivedEvents:
			receivedTypes[received.GetType()] = true
		case <-time.After(10 * time.Second):
			t.Fatal("Did not receive all events within timeout")
		}
	}

	assert.True(t, receivedTypes[events.WorkflowTriggeredEvent])
	assert.True(t, receivedTypes[events.NodeActivationEvent])
}

func TestKafkaEventBus_Close(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	err = bus.Close(context.Background())
	assert.NoError(t, err)
}

func TestPublishEvent(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	kafkaBus := bus.(*kafkaEventBus)
	testEvent := createTestEvent(events.WorkflowTriggeredEvent)

	err = publishEvent(context.Background(), logger, kafkaBus.writer, "test-key", testEvent)
	assert.NoError(t, err)
}

func TestExtractEvent(t *testing.T) {
	tests := []struct {
		name         string
		eventType    events.EventType
		expectError  bool
		expectedType any
	}{
		{"WorkflowTriggered", events.WorkflowTriggeredEvent, false, &events.WorkflowTriggered{}},
		{"WorkflowFinished", events.WorkflowFinishedEvent, false, &events.WorkflowFinished{}},
		{"WorkflowFailed", events.WorkflowFailedEvent, false, &events.WorkflowFailed{}},
		{"NodeActivation", events.NodeActivationEvent, false, &events.NodeActivation{}},
		{"NodeCompletion", events.NodeCompletionEvent, false, &events.NodeCompletion{}},
		{"Unknown", "unknown.event", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := extractEvent(tt.eventType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, event)
			} else {
				assert.NoError(t, err)
				assert.IsType(t, tt.expectedType, event)
			}
		})
	}
}

func TestConsumeEvents_ErrorHandling(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	kafkaBus := bus.(*kafkaEventBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumeEvents(ctx, logger, kafkaBus.reader, kafkaBus.handlers)

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestKafkaEventBus_WithGroupID(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", brokers)
	t.Setenv("KAFKA_GROUP_ID", "test-group")

	bus, err := NewEventBus(context.Background(), logger)
	require.NoError(t, err)

	defer func() {
		err := bus.Close(context.Background())
		assert.NoError(t, err)
	}()

	kafkaBus := bus.(*kafkaEventBus)
	assert.NotNil(t, kafkaBus.reader)
}

func TestKafkaEventBus_EventSerialization(t *testing.T) {
	testEvent := &events.WorkflowTriggered{
		BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, "test-workflow"),
		TriggerID:   "test-trigger",
		TriggerData: map[string]any{"key": "value", "number": 42},
	}

	data, err := json.Marshal(testEvent)
	require.NoError(t, err)

	var unmarshaled events.WorkflowTriggered

	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, testEvent.GetType(), unmarshaled.GetType())
	assert.Equal(t, testEvent.TriggerID, unmarshaled.TriggerID)
	assert.Equal(t, testEvent.WorkflowID, unmarshaled.WorkflowID)
}

func createTopics(brokers string) {
	conn, err := kafkago.Dial("tcp", brokers)
	if err != nil {
		panic(err.Error())
	}

	defer func() {
		if err := conn.Close(); err != nil {
			panic(err.Error())
		}
	}()

	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}

	var controllerConn *kafkago.Conn

	controllerConn, err = kafkago.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}

	defer func() {
		err := controllerConn.Close()
		if err != nil {
			panic(err.Error())
		}
	}()

	topicConfigs := []kafkago.TopicConfig{
		{
			Topic:             events.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}
}
