package eventbus

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dukex/operion/pkg/events"
)

type WatermillEventBus struct {
	publisher     message.Publisher
	subscriber    message.Subscriber
	subscriptions map[events.EventType]EventHandler
	subscribedTopics []string
}

func NewWatermillEventBus(pub message.Publisher, sub message.Subscriber) EventBus {
	return &WatermillEventBus{
		publisher:     pub,
		subscriber:    sub,
		subscriptions: make(map[events.EventType]EventHandler),
	}
}

func (eb *WatermillEventBus) GenerateID() string {
	return watermill.NewULID()
}

func (eb *WatermillEventBus) Publish(ctx context.Context, topic string, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage("msg-"+eb.GenerateID(), payload)
	msg.Metadata.Set(events.EventMetadataKey, topic)
	msg.Metadata.Set(events.EventTypeMetadataKey, string(event.GetType()))
	return eb.publisher.Publish(topic, msg)
}

func (eb *WatermillEventBus) Subscribe(ctx context.Context, topics ...string) error {
	// Default to operion.events if no topics specified
	if len(topics) == 0 {
		topics = []string{events.Topic}
	}
	
	eb.subscribedTopics = topics
	
	// Subscribe to each topic
	for _, topic := range topics {
		messages, err := eb.subscriber.Subscribe(ctx, topic)
		if err != nil {
			return err
		}

		go func(topicName string, msgChan <-chan *message.Message) {
			for msg := range msgChan {
				var event interface{}

				eventType := events.EventType(msg.Metadata.Get(events.EventTypeMetadataKey))
				handler, exists := eb.subscriptions[eventType]
				if !exists {
					msg.Ack()
					continue
				}

				switch eventType {
				case events.WorkflowTriggeredEvent:
					event = &events.WorkflowTriggered{}
				case events.WorkflowFinishedEvent:
					event = &events.WorkflowFinished{}
				case events.WorkflowFailedEvent:
					event = &events.WorkflowFailed{}
				case events.WorkflowStepAvailableEvent:
					event = &events.WorkflowStepAvailable{}
				case events.WorkflowStepFinishedEvent:
					event = &events.WorkflowStepFinished{}
				case events.WorkflowStepFailedEvent:
					event = &events.WorkflowStepFailed{}
				case events.TriggerDetectedEvent:
					event = &events.TriggerEvent{}
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
		}(topic, messages)
	}

	return nil
}

func (eb *WatermillEventBus) Handle(eventType events.EventType, handler EventHandler) error {
	eb.subscriptions[eventType] = handler
	return nil
}

func (eb *WatermillEventBus) Close() error {
	if err := eb.publisher.Close(); err != nil {
		return err
	}
	return eb.subscriber.Close()
}
