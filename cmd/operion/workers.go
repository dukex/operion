package main

import (
	"fmt"
	"log"

	"github.com/dukex/operion/internal/adapters/persistence/file"
	"github.com/dukex/operion/internal/application"
)


func runWorkers(workflowsPath, filterTags, workerID string) {
	fmt.Printf("Starting workers with workflows from: %s\n", workflowsPath)
	if filterTags != "" {
		fmt.Printf("Filter tags: %s\n", filterTags)
	}

	persistence := file.NewFilePersistence(workflowsPath)
	workflowRepo := application.NewWorkflowRepository(persistence)
	triggerRegistry := application.SetupTriggerRegistry()
	actionRegistry := application.SetupActionRegistry()
	workflowService := application.NewWorkflowService(workflowRepo, actionRegistry)

	workerManager := application.NewWorkerManager(
		workerID,
		workflowRepo,
		triggerRegistry,
		workflowService,
	)

	fmt.Printf("Worker ID: %s\n", workerManager.GetWorkerID())

	// Start workers
	if err := workerManager.StartWorkers(filterTags); err != nil {
		log.Fatalf("Failed to start workers: %v", err)
	}
}
