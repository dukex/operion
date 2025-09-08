package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	kafkago "github.com/segmentio/kafka-go"
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

	headers := []kafkago.Header{
		{
			Key:   events.EventMetadataKey,
			Value: []byte(key),
		}, {
			Key:   events.EventTypeMetadataKey,
			Value: []byte(event.GetType()),
		},
	}

	publishCtx := context.WithoutCancel(ctx)
	err = writer.WriteMessages(publishCtx, kafkago.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
	})

	return err
}
