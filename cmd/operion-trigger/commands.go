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

func RunTriggerService(cmd *cli.Command) error {
	serviceID := cmd.String("trigger-id")
	if serviceID == "" {
		serviceID = fmt.Sprintf("trigger-%s", uuid.New().String()[:8])
	}

	logger := log.WithFields(log.Fields{
		"module":     "trigger",
		"service_id": serviceID,
		"action":     "run",
	})

	logger.Info("Starting trigger service")

	// Setup persistence
	persistence := setupPersistence(cmd.String("data-path"))

	// Setup repository
	workflowRepository := workflow.NewRepository(persistence)

	// Setup registry
	registry.RegisterAllComponents()
	registry := registry.DefaultRegistry

	// Setup event bus
	eventBus, err := setupEventBus(cmd, logger, serviceID)
	if err != nil {
		return fmt.Errorf("failed to setup event bus: %w", err)
	}
	defer eventBus.Close()

	// Create and start trigger manager
	triggerManager := NewTriggerManager(
		serviceID,
		workflowRepository,
		registry,
		eventBus,
	)

	if err := triggerManager.Start(); err != nil {
		logger.Fatalf("Failed to start trigger service: %v", err)
	}

	return nil
}

func ListTriggers(cmd *cli.Command) error {
	logger := log.WithFields(log.Fields{
		"module": "trigger-service",
		"action": "list",
	})

	// Setup persistence and repository
	persistence := setupPersistence(cmd.String("data-path"))
	workflowRepository := workflow.NewRepository(persistence)

	// Fetch all workflows
	workflows, err := workflowRepository.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	logger.Infof("Found %d workflows", len(workflows))

	fmt.Println("Available Triggers:")
	fmt.Println("==================")

	totalTriggers := 0
	for _, workflow := range workflows {
		if len(workflow.Triggers) == 0 {
			continue
		}

		fmt.Printf("\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)
		fmt.Printf("Status: %s\n", workflow.Status)
		fmt.Printf("Triggers:\n")

		for _, trigger := range workflow.Triggers {
			fmt.Printf("  - ID: %s\n", trigger.ID)
			fmt.Printf("    Type: %s\n", trigger.Type)
			fmt.Printf("    Config: %v\n", trigger.Configuration)
			totalTriggers++
		}
	}

	fmt.Printf("\nTotal triggers: %d\n", totalTriggers)
	return nil
}

func ValidateTriggers(cmd *cli.Command) error {
	logger := log.WithFields(log.Fields{
		"module": "trigger-service",
		"action": "validate",
	})

	// Setup persistence and repository
	persistence := setupPersistence(cmd.String("data-path"))
	workflowRepository := workflow.NewRepository(persistence)

	registry.RegisterAllComponents()
	registry := registry.DefaultRegistry

	// Fetch all workflows
	workflows, err := workflowRepository.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	logger.Infof("Validating triggers in %d workflows", len(workflows))

	fmt.Println("Trigger Validation Results:")
	fmt.Println("===========================")

	totalTriggers := 0
	validTriggers := 0
	invalidTriggers := 0

	for _, workflow := range workflows {
		if len(workflow.Triggers) == 0 {
			continue
		}

		fmt.Printf("\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)

		for _, trigger := range workflow.Triggers {
			totalTriggers++
			fmt.Printf("  Trigger: %s (%s)\n", trigger.ID, trigger.Type)

			// Try to create trigger to validate
			config := make(map[string]interface{})
			for k, v := range trigger.Configuration {
				config[k] = v
			}
			config["workflow_id"] = workflow.ID
			config["trigger_id"] = trigger.ID
			config["id"] = trigger.ID

			if _, err := registry.CreateTrigger(trigger.Type, config); err != nil {
				fmt.Printf("    ❌ INVALID: %v\n", err)
				invalidTriggers++
			} else {
				fmt.Printf("    ✅ VALID\n")
				validTriggers++
			}
		}
	}

	fmt.Printf("\nValidation Summary:\n")
	fmt.Printf("  Total triggers: %d\n", totalTriggers)
	fmt.Printf("  Valid triggers: %d\n", validTriggers)
	fmt.Printf("  Invalid triggers: %d\n", invalidTriggers)

	if invalidTriggers > 0 {
		return fmt.Errorf("found %d invalid triggers", invalidTriggers)
	}

	fmt.Println("All triggers are valid! ✅")
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

	pub, sub, err := kafka.CreateChannel(watermillLogger, "operion-trigger")

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka pub/sub: %w", err)
	}
	eventBus = event_bus.NewEventBus(pub, sub, id)

	return eventBus, nil
}
