package event_bus

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dukex/operion/pkg/events"
	"github.com/google/uuid"
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
}

func NewEventBus(pub message.Publisher, sub message.Subscriber, id string) *EventBus {
	return &EventBus{
		publisher:  pub,
		subscriber: sub,
	}
}

func (eb *EventBus) Publish(ctx context.Context, event interface{}) error {
	var topic string

	switch event.(type) {
	case events.WorkflowTriggered:
		topic = string(events.WorkflowTriggeredEvent)
	case events.WorkflowFinished:
		topic = string(events.WorkflowFinishedEvent)
	case events.WorkflowFailed:
		topic = string(events.WorkflowFailedEvent)
	case events.WorkflowStepStarted:
		topic = string(events.WorkflowStepStartedEvent)
	case events.WorkflowStepFinished:
		topic = string(events.WorkflowStepFinishedEvent)
	case events.WorkflowStepFailed:
		topic = string(events.WorkflowStepFailedEvent)
	default:
		topic = "unknown"
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(generateMessageID(), payload)
	return eb.publisher.Publish(topic, msg)
}

func (eb *EventBus) Subscribe(ctx context.Context, topic string, handler EventHandler) error {
	messages, err := eb.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go func() {
		for msg := range messages {
			var event interface{}
			eventType := events.EventType(topic)

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
				msg.Nack()
				continue
			}

			if err := json.Unmarshal(msg.Payload, event); err != nil {
				msg.Nack()
				continue
			}

			if err := handler(ctx, event); err != nil {
				msg.Nack()
				continue
			}

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
