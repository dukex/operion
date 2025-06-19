package main

import (
	"fmt"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/dukex/operion/pkg/channels/kafka"
	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

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

	// Setup persistence
	persistence := setupPersistence(cmd.String("data-path"))

	// Setup repository
	workflowRepository := workflow.NewRepository(persistence)

	// Setup registry
	registry.RegisterAllComponents()
	registry := registry.DefaultRegistry

	// Setup event bus
	eventBus, err := setupEventBus(cmd, logger, workerID)
	if err != nil {
		return fmt.Errorf("failed to setup event bus: %w", err)
	}
	defer eventBus.Close()

	workflowExecutor := workflow.NewExecutor(
		workflowRepository,
		registry,
	)

	worker := NewWorker(
		workerID,
		workflowRepository,
		workflowExecutor,
		eventBus,
	)

	if err := worker.Start(); err != nil {
		logger.Fatalf("Failed to start event-driven worker: %v", err)
	}

	return nil
}

func setupPersistence(dataPath string) persistence.Persistence {
	if dataPath == "" {
		dataPath = os.Getenv("DATA_PATH")
		if dataPath == "" {
			dataPath = "./data"
		}
	}
	return file.NewFilePersistence(dataPath)
}

func setupEventBus(cmd *cli.Command, logger *log.Entry, id string) (event_bus.EventBusI, error) {
	var eventBus event_bus.EventBusI
	watermillLogger := watermill.NewStdLogger(true, true)

	pub, sub, err := kafka.CreateChannel(watermillLogger, "operion-worker")

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka pub/sub: %w", err)
	}
	eventBus = event_bus.NewEventBus(pub, sub, id)

	return eventBus, nil
}
