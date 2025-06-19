package event_bus

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dukex/operion/pkg/events"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
}

type EventSubscriber interface {
	Subscribe(ctx context.Context, topic string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event interface{}) error

type EventBusI interface {
	EventPublisher
	EventSubscriber
	Close() error
}

type EventBus struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	tracer     trace.Tracer
}

func NewEventBus(pub message.Publisher, sub message.Subscriber, id string) *EventBus {
	return &EventBus{
		publisher:  pub,
		subscriber: sub,
		tracer:     trc.GetTracer("event-bus"),
	}
}

func (eb *EventBus) Publish(ctx context.Context, event interface{}) error {
	ctx, span := trc.StartSpan(ctx, eb.tracer, "event_bus.publish")
	defer span.End()

	var topic string
	var eventID string

	switch e := event.(type) {
	case events.WorkflowTriggered:
		topic = string(events.WorkflowTriggeredEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String(trc.TriggerTypeKey, e.TriggerType),
			attribute.String(trc.TriggerIDKey, e.TriggerID),
		)
	case events.WorkflowFinished:
		topic = string(events.WorkflowFinishedEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String("execution_id", e.ExecutionID),
		)
	case events.WorkflowFailed:
		topic = string(events.WorkflowFailedEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String("execution_id", e.ExecutionID),
			attribute.String("error", e.Error),
		)
	case events.WorkflowStepStarted:
		topic = string(events.WorkflowStepStartedEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String(trc.StepIDKey, e.StepID),
		)
	case events.WorkflowStepFinished:
		topic = string(events.WorkflowStepFinishedEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String(trc.StepIDKey, e.StepID),
		)
	case events.WorkflowStepFailed:
		topic = string(events.WorkflowStepFailedEvent)
		eventID = e.ID
		span.SetAttributes(
			attribute.String(trc.WorkflowIDKey, e.WorkflowID),
			attribute.String(trc.StepIDKey, e.StepID),
			attribute.String("error", e.Error),
		)
	default:
		topic = "unknown"
		eventID = "unknown"
	}

	span.SetAttributes(
		attribute.String("event.topic", topic),
		attribute.String("event.id", eventID),
	)
	span.AddEvent("serializing_event")

	payload, err := json.Marshal(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal event")
		return err
	}

	messageID := generateMessageID()
	span.SetAttributes(attribute.String("message.id", messageID))
	span.AddEvent("publishing_message")

	msg := message.NewMessage(messageID, payload)
	if err := eb.publisher.Publish(topic, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to publish message")
		return err
	}

	span.AddEvent("message_published")
	span.SetStatus(codes.Ok, "event published successfully")
	return nil
}

func (eb *EventBus) Subscribe(ctx context.Context, topic string, handler EventHandler) error {
	ctx, span := trc.StartSpan(ctx, eb.tracer, "event_bus.subscribe",
		attribute.String("subscription.topic", topic),
	)
	defer span.End()

	span.AddEvent("subscribing_to_topic")
	messages, err := eb.subscriber.Subscribe(ctx, topic)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to subscribe to topic")
		return err
	}

	span.AddEvent("subscription_established")
	span.SetStatus(codes.Ok, "subscription established")

	go func() {
		tracer := trc.GetTracer("event-bus-handler")
		for msg := range messages {
			// Create a fresh context for each message to avoid parent span issues
			msgCtx, msgSpan := trc.StartSpan(context.Background(), tracer, "event_bus.handle_message",
				attribute.String("subscription.topic", topic),
				attribute.String("message.id", msg.UUID),
			)

			var event interface{}
			eventType := events.EventType(topic)

			msgSpan.AddEvent("determining_event_type")
			switch eventType {
			case events.WorkflowTriggeredEvent:
				event = &events.WorkflowTriggered{}
			case events.WorkflowFinishedEvent:
				event = &events.WorkflowFinished{}
			case events.WorkflowFailedEvent:
				event = &events.WorkflowFailed{}
			case events.WorkflowStepStartedEvent:
				event = &events.WorkflowStepStarted{}
			case events.WorkflowStepFinishedEvent:
				event = &events.WorkflowStepFinished{}
			case events.WorkflowStepFailedEvent:
				event = &events.WorkflowStepFailed{}
			default:
				msgSpan.AddEvent("unknown_event_type_nack")
				msgSpan.SetStatus(codes.Error, "unknown event type")
				msgSpan.End()
				msg.Nack()
				continue
			}

			msgSpan.AddEvent("deserializing_event")
			if err := json.Unmarshal(msg.Payload, event); err != nil {
				msgSpan.RecordError(err)
				msgSpan.AddEvent("deserialization_failed_nack")
				msgSpan.SetStatus(codes.Error, "failed to deserialize event")
				msgSpan.End()
				msg.Nack()
				continue
			}

			msgSpan.AddEvent("calling_event_handler")
			if err := handler(msgCtx, event); err != nil {
				msgSpan.RecordError(err)
				msgSpan.AddEvent("handler_failed_nack")
				msgSpan.SetStatus(codes.Error, "event handler failed")
				msgSpan.End()
				msg.Nack()
				continue
			}

			msgSpan.AddEvent("message_acked")
			msgSpan.SetStatus(codes.Ok, "message processed successfully")
			msgSpan.End()
			msg.Ack()
		}
	}()

	return nil
}

func (eb *EventBus) Close() error {
	if err := eb.publisher.Close(); err != nil {
		return err
	}
	return eb.subscriber.Close()
}

func generateMessageID() string {
	return "msg-" + generateUUID()
}

func generateUUID() string {
	return uuid.New().String()
}
