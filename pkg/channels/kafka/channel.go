// Package kafka provides Apache Kafka integration for event messaging.
package kafka

import (
	"errors"
	"os"
	"strings"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dukex/operion/pkg/events"
)

func MetadataKey(topic string, msg *message.Message) (string, error) {
	if msg == nil {
		return "", errors.New("message is nil")
	}

	if msg.Metadata == nil || msg.Metadata[events.EventMetadataKey] == "" {
		return "", errors.New("message metadata is nil")
	}

	key, ok := msg.Metadata[events.EventMetadataKey]
	if !ok || key == "" {
		return "", errors.New("message metadata does not contain 'key'")
	}

	return key, nil
}

func CreateChannel(logger watermill.LoggerAdapter, serviceName string) (*kafka.Publisher, *kafka.Subscriber, error) {
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		return nil, nil, errors.New("KAFKA_BROKERS environment variable is not set or empty")
	}

	saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()
	saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               brokers,
			Unmarshaler:           kafka.NewWithPartitioningMarshaler(MetadataKey),
			OverwriteSaramaConfig: saramaSubscriberConfig,
			ConsumerGroup:         "cg-" + serviceName,
			OTELEnabled:           true,
		},
		logger,
	)
	if err != nil {
		return nil, nil, err
	}

	saramaPublisherConfig := sarama.NewConfig()
	saramaPublisherConfig.Producer.Return.Successes = true

	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             kafka.NewWithPartitioningMarshaler(MetadataKey),
			OverwriteSaramaConfig: saramaPublisherConfig,
			OTELEnabled:           true,
		},
		logger,
	)
	if err != nil {
		return nil, nil, err
	}

	return publisher, subscriber, nil
}
