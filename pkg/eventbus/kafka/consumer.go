package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/otelhelper"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func consumeEvents(
	ctx context.Context,
	logger *slog.Logger,
	reader *kafkago.Reader,
	tracer trace.Tracer,
	handlers map[events.EventType]eventbus.EventHandler,
) {
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.InfoContext(ctx, "Stopping consumer due to context cancellation or deadline exceeded")

				break
			}

			logger.ErrorContext(ctx, "failed to fetch message", "error", err)

			continue
		}

		logger.InfoContext(ctx, "Received message", "key", string(message.Key), "topic", message.Topic)

		var eventType events.EventType

		carrier := propagation.MapCarrier{}
		for _, header := range message.Headers {
			carrier[header.Key] = string(header.Value)

			if header.Key == events.EventTypeMetadataKey {
				eventType = events.EventType(header.Value)
			} else {
				carrier[header.Key] = string(header.Value)
			}
		}

		propagator := otel.GetTextMapPropagator()
		msgCtx := propagator.Extract(ctx, carrier)

		traceCtx, span := otelhelper.StartSpan(msgCtx, tracer, "worker.consumer consume",
			attribute.String("kafka.key", string(message.Key)),
			attribute.String("kafka.topic", message.Topic),
		)
		defer span.End()

		logger.InfoContext(msgCtx, "Processing message", "event_type", eventType)

		handler, exists := handlers[eventType]
		if !exists {
			if eventType == events.WorkflowFinishedEvent {
				span.SetStatus(codes.Ok, "workflow finished event, no handler needed")

				continue
			}

			logger.ErrorContext(msgCtx, "No handler found for event type", "event_type", eventType)
			otelhelper.SetError(span, errors.New("no handler found for event type"))

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
			logger.ErrorContext(msgCtx, "Unknown event type", "event_type", eventType)
			otelhelper.SetError(span, errors.New("unknown event type"))

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(msgCtx, "Failed to commit message", "error", err)
			}

			continue
		}

		err = json.Unmarshal(message.Value, event)
		if err != nil {
			logger.ErrorContext(msgCtx, "Failed to unmarshal event", "error", err, "event_type", eventType)
			otelhelper.SetError(span, err)

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(msgCtx, "Failed to commit message", "error", err)
			}

			continue
		}

		handlerErr := handler(traceCtx, event)
		if handlerErr != nil {
			logger.ErrorContext(msgCtx, "Failed to handle event", "error", handlerErr, "event_type", eventType)
			otelhelper.SetError(span, handlerErr)

			err := reader.CommitMessages(ctx, message)
			if err != nil {
				logger.ErrorContext(msgCtx, "Failed to commit message", "error", err)
			}

			continue
		}

		span.AddEvent("event_handled", trace.WithAttributes())
		logger.InfoContext(ctx, "Successfully handled event", "event_type", eventType)

		err = reader.CommitMessages(ctx, message)
		if err != nil {
			logger.ErrorContext(msgCtx, "Failed to commit message", "error", err)
		}
	}
}
