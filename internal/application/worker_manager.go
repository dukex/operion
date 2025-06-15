package application

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/dukex/operion/internal/domain"
)

// WorkerManager manages workflow execution workers
type WorkerManager struct {
	workerID         string
	workflowRepo     *WorkflowRepository
	triggerRegistry  *TriggerRegistry
	workflowService  *WorkflowService
	runningTriggers  map[string]domain.Trigger
	triggerMutex     sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(
	workerID string,
	workflowRepo *WorkflowRepository,
	triggerRegistry *TriggerRegistry,
	workflowService *WorkflowService,
) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerManager{
		workerID:        workerID,
		workflowRepo:    workflowRepo,
		triggerRegistry: triggerRegistry,
		workflowService: workflowService,
		runningTriggers: make(map[string]domain.Trigger),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// StartWorkers starts workers for workflows matching the filter
func (wm *WorkerManager) StartWorkers(filterTags string) error {
	workflows, err := wm.workflowRepo.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	filteredWorkflows := wm.filterWorkflows(workflows, filterTags)
	
	if len(filteredWorkflows) == 0 {
		fmt.Println("No workflows match the filter criteria")
		return nil
	}

	fmt.Printf("[Worker %s] Found %d workflows to execute\n", wm.workerID, len(filteredWorkflows))

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start each workflow
	for _, workflow := range filteredWorkflows {
		wg.Add(1)
		go func(wf domain.Workflow) {
			defer wg.Done()
			if err := wm.startWorkflowTriggers(wf); err != nil {
				log.Printf("[Worker %s] Failed to start workflow %s: %v", wm.workerID, wf.ID, err)
			}
		}(workflow)
	}

	// Wait for shutdown signal
	go func() {
		<-c
		fmt.Printf("\n[Worker %s] Shutting down...\n", wm.workerID)
		wm.Stop()
	}()

	wg.Wait()
	fmt.Printf("[Worker %s] Stopped\n", wm.workerID)
	return nil
}

// startWorkflowTriggers starts all triggers for a workflow
func (wm *WorkerManager) startWorkflowTriggers(workflow domain.Workflow) error {
	fmt.Printf("[Worker %s] Starting triggers for workflow: %s (%s)\n", wm.workerID, workflow.Name, workflow.ID)

	for _, triggerItem := range workflow.Triggers {
		// Add the trigger ID to the configuration
		config := make(map[string]interface{})
		for k, v := range triggerItem.Configuration {
			config[k] = v
		}
		config["id"] = triggerItem.ID
		trigger, err := wm.triggerRegistry.Create(triggerItem.Type, config)
		if err != nil {
			log.Printf("[Worker %s] Failed to create trigger %s for workflow %s: %v", 
				wm.workerID, triggerItem.ID, workflow.ID, err)
			continue
		}

		// Store the trigger
		wm.triggerMutex.Lock()
		wm.runningTriggers[triggerItem.ID] = trigger
		wm.triggerMutex.Unlock()

		// Create callback for this workflow
		callback := wm.createWorkflowCallback(workflow.ID)

		// Start the trigger
		if err := trigger.Start(wm.ctx, callback); err != nil {
			log.Printf("[Worker %s] Failed to start trigger %s for workflow %s: %v", 
				wm.workerID, triggerItem.ID, workflow.ID, err)
			
			// Remove from running triggers if failed to start
			wm.triggerMutex.Lock()
			delete(wm.runningTriggers, triggerItem.ID)
			wm.triggerMutex.Unlock()
			continue
		}

		fmt.Printf("[Worker %s] Started trigger %s (%s) for workflow %s\n", 
			wm.workerID, triggerItem.ID, triggerItem.Type, workflow.ID)
	}

	// Keep the goroutine alive until context is cancelled
	<-wm.ctx.Done()
	return nil
}

// createWorkflowCallback creates a callback function for workflow execution
func (wm *WorkerManager) createWorkflowCallback(workflowID string) domain.TriggerCallback {
	return func(ctx context.Context, data map[string]interface{}) error {
		fmt.Printf("[Worker %s] Executing workflow %s triggered by %s\n", 
			wm.workerID, workflowID, data["trigger_id"])
		
		// Execute the workflow using the workflow service
		wm.workflowService.ExecuteWorkflow(ctx, workflowID, data)
		return nil
	}
}

// Stop stops all running triggers and cancels the context
func (wm *WorkerManager) Stop() {
	wm.cancel()

	wm.triggerMutex.Lock()
	defer wm.triggerMutex.Unlock()

	for triggerID, trigger := range wm.runningTriggers {
		fmt.Printf("[Worker %s] Stopping trigger %s\n", wm.workerID, triggerID)
		if err := trigger.Stop(context.Background()); err != nil {
			log.Printf("[Worker %s] Error stopping trigger %s: %v", wm.workerID, triggerID, err)
		}
	}

	// Clear running triggers
	wm.runningTriggers = make(map[string]domain.Trigger)
}

// GetWorkerID returns the worker ID
func (wm *WorkerManager) GetWorkerID() string {
	return wm.workerID
}

// filterWorkflows filters workflows based on tags
func (wm *WorkerManager) filterWorkflows(workflows []domain.Workflow, filterTags string) []domain.Workflow {
	if filterTags == "" {
		return workflows
	}

	tags := strings.Split(filterTags, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}

	var filtered []domain.Workflow
	for _, workflow := range workflows {
		if wm.workflowMatchesTags(workflow, tags) {
			filtered = append(filtered, workflow)
		}
	}

	return filtered
}

// workflowMatchesTags checks if a workflow matches the required tags
func (wm *WorkerManager) workflowMatchesTags(workflow domain.Workflow, tags []string) bool {
	workflowTags, exists := workflow.Metadata["tags"]
	if !exists {
		return false
	}

	var workflowTagList []string
	switch t := workflowTags.(type) {
	case []interface{}:
		for _, tag := range t {
			if str, ok := tag.(string); ok {
				workflowTagList = append(workflowTagList, str)
			}
		}
	case []string:
		workflowTagList = t
	case string:
		workflowTagList = strings.Split(t, ",")
		for i := range workflowTagList {
			workflowTagList[i] = strings.TrimSpace(workflowTagList[i])
		}
	default:
		return false
	}

	for _, requiredTag := range tags {
		found := false
		for _, workflowTag := range workflowTagList {
			if strings.EqualFold(requiredTag, workflowTag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}