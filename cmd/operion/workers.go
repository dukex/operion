package main

import (
	"fmt"

	"github.com/dukex/operion/internal/adapters/persistence/file"
	"github.com/dukex/operion/internal/application"
	"github.com/dukex/operion/internal/domain"
	file_write_action "github.com/dukex/operion/pkg/actions/file_write"
	http_action "github.com/dukex/operion/pkg/actions/http_request"
	log_action "github.com/dukex/operion/pkg/actions/log"
	transform_action "github.com/dukex/operion/pkg/actions/transform"
	"github.com/dukex/operion/pkg/registry"
	triggers "github.com/dukex/operion/pkg/triggers/schedule"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func runWorkers(cmd *cli.Command) error {
	workerID := cmd.String("worker-id")

	if workerID == "" {
		workerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	}

	logger := log.WithFields(
		log.Fields{
			"module":    "cli",
			"worker_id": workerID,
		},
	)

	logger.Info("Starting workers with workflows ")

	persistence := file.NewFilePersistence("/Users/emerson.almeidacaju.com.br/code/dukex/operion/data")

	workflowRepo := application.NewWorkflowRepository(persistence)
	triggerRegistry := createTriggerRegistry()
	actionRegistry := createActionRegistry()
	WorkflowExecutor := application.NewWorkflowExecutor(workflowRepo, actionRegistry)

	workerManager := application.NewWorkerManager(
		workerID,
		workflowRepo,
		triggerRegistry,
		WorkflowExecutor,
	)

	if err := workerManager.StartWorkers(); err != nil {
		logger.Fatalf("Failed to start workers: %v", err)
	}

	return nil
}

func createTriggerRegistry() *registry.TriggerRegistry {
	trigger := registry.NewTriggerRegistry()

	trigger.Register("schedule", func(config map[string]interface{}) (domain.Trigger, error) {
		return triggers.NewScheduleTrigger(config)
	})

	return trigger
}

func createActionRegistry() *registry.ActionRegistry {
	action := registry.NewActionRegistry()

	action.Register("http_request", func(config map[string]interface{}) (domain.Action, error) {
		return http_action.NewHTTPRequestAction(config)
	})

	action.Register("log", func(config map[string]interface{}) (domain.Action, error) {
		return log_action.NewLogAction(config)
	})

	action.Register("transform", func(config map[string]interface{}) (domain.Action, error) {
		return transform_action.NewTransformAction(config)
	})

	action.Register("file_write", func(config map[string]interface{}) (domain.Action, error) {
		return file_write_action.NewFileWriteAction(config)
	})

	return action
}
