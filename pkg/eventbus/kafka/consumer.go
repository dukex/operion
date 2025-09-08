package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	kafkago "github.com/segmentio/kafka-go"
)

func consumeEvents(
	ctx context.Context,
	logger *slog.Logger,
	reader *kafkago.Reader,
	handlers map[events.EventType]eventbus.EventHandler,
) {
	const maxRetries = 3
	retryCount := 0

	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.InfoContext(ctx, "Reached end of stream", "error", err)

				break
			}
			
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.InfoContext(ctx, "Stopping consumer due to context cancellation or deadline exceeded")

				break
			}
			
			if retryCount < maxRetries {
				retryCount++
				logger.InfoContext(ctx, "Error fetching message, retrying...", "attempt", retryCount, "error", err)

				continue
			}

			logger.ErrorContext(ctx, "failed to fetch message", "error", err)

			break
		}

		logger.InfoContext(ctx, "Received message", "key", string(message.Key), "topic", message.Topic)

		// Extract event type from headers
		var eventType events.EventType
		for _, header := range message.Headers {
			if header.Key == events.EventTypeMetadataKey {
				eventType = events.EventType(header.Value)
				break
			}
		}

		logger.InfoContext(ctx, "Processing message", "event_type", eventType)

		handler, exists := handlers[eventType]
		if !exists {
			if eventType == events.WorkflowFinishedEvent {
				continue
			}

			logger.ErrorContext(ctx, "No handler found for event type", "event_type", eventType)

			continue
		}

		var event any

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
		default:
			logger.ErrorContext(ctx, "Unknown event type", "event_type", eventType)

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to commit message", "error", err)
			}

			continue
		}

		err = json.Unmarshal(message.Value, event)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to unmarshal event", "error", err, "event_type", eventType)

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to commit message", "error", err)
			}

			continue
		}

		handlerErr := handler(ctx, event)
		if handlerErr != nil {
			logger.ErrorContext(ctx, "Failed to handle event", "error", handlerErr, "event_type", eventType)

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to commit message", "error", err)
			}

			continue
		}

		logger.InfoContext(ctx, "Successfully handled event", "event_type", eventType)

		err = reader.CommitMessages(ctx, message)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to commit message", "error", err)
		}
	}
}
