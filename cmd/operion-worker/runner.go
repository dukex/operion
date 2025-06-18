package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func setupPersistence() persistence.Persistence {
	return file.NewFilePersistence(os.Getenv("DATA_PATH"))
}

func setupEventBus(cmd *cli.Command, logger *log.Entry) (event_bus.EventBusI, error) {
	var eventBus event_bus.EventBusI
	watermillLogger := watermill.NewStdLogger(false, false)

	if cmd.Bool("kafka") {
		logger.Info("Using Kafka as message broker")
		pub, sub, err := createKafkaPubSub(watermillLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kafka pub/sub: %w", err)
		}
		eventBus = event_bus.NewEventBus(pub, sub)
	} else {
		logger.Info("Using GoChannel as message broker")
		pubSub := gochannel.NewGoChannel(
			gochannel.Config{},
			watermillLogger,
		)
		eventBus = event_bus.NewEventBus(pubSub, pubSub)
	}

	return eventBus, nil
}

func RunWorkers(cmd *cli.Command) error {
	workerID := cmd.String("worker-id")

	if workerID == "" {
		workerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	}

	logger := log.WithFields(
		log.Fields{
			"module":    "worker",
			"worker_id": workerID,
			"action":    "run",
		},
	)

	logger.Info("Starting worker")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workflowRepository := workflow.NewRepository(
		setupPersistence(),
	)

	workflowExecutor := workflow.NewExecutor(
		workflowRepository,
		createRegistry(),
	)

	eventBus, err := setupEventBus(
		cmd,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to setup event bus: %w", err)
	}
	defer eventBus.Close()

	worker := NewWorker(
		workerID,
		workflowRepository,
		workflowExecutor,
		eventBus,
		logger,
	)

	if err := worker.Start(ctx); err != nil {
		logger.Fatalf("Failed to start event-driven worker: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down worker...")
	cancel()

	return nil
}

func createRegistry() *registry.Registry {
	reg := registry.GetDefaultRegistry()
	return reg
}

func createKafkaPubSub(logger watermill.LoggerAdapter) (*kafka.Publisher, *kafka.Subscriber, error) {
	brokers := []string{"kafka:9092"}
	if host := os.Getenv("KAFKA_BROKERS"); host != "" {
		brokers = []string{host}
	}

	saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()
	saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               brokers,
			Unmarshaler:           kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaSubscriberConfig,
			ConsumerGroup:         "operion-workers",
		},
		logger,
	)

	if err != nil {
		panic(err)
	}

	saramaPublisherConfig := sarama.NewConfig()
	saramaPublisherConfig.Producer.Return.Successes = true
	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaPublisherConfig,
		},
		logger,
	)
	if err != nil {
		return nil, nil, err
	}

	return publisher, subscriber, nil
}
