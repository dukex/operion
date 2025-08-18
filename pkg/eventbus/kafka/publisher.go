package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func publishEvent(
	ctx context.Context,
	logger *slog.Logger,

	writer *kafkago.Writer,
	key string,
	event eventbus.Event,
) error {
	logger.InfoContext(ctx, "Publishing event", "key", key, "event_type", event.GetType())

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)

	headers := make([]kafkago.Header, 0, len(carrier)+2)

	for k, v := range carrier {
		headers = append(headers, kafkago.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	headers = append(headers, kafkago.Header{
		Key:   events.EventMetadataKey,
		Value: []byte(key),
	}, kafkago.Header{
		Key:   events.EventTypeMetadataKey,
		Value: []byte(event.GetType()),
	})

	publishCtx := context.WithoutCancel(ctx)
	err = writer.WriteMessages(publishCtx, kafkago.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
	})

	return err
}
