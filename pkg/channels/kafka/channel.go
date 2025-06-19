package kafka

import (
	"errors"
	"os"
	"strings"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
)

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
			Unmarshaler:           kafka.DefaultMarshaler{},
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
			Marshaler:             kafka.DefaultMarshaler{},
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
