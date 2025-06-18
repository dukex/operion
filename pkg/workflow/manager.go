package workflow

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/registry"
)

type Manager struct {
	workerID        string
	repository      *Repository
	registry        *registry.Registry
	executor        *Executor
	runningTriggers map[string]models.Trigger
	triggerMutex    sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *log.Entry
}

func NewManager(
	workerID string,
	repository *Repository,
	registry *registry.Registry,
	WorkflowExecutor *Executor,
) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		workerID:        workerID,
		repository:      repository,
		registry:        registry,
		executor:        WorkflowExecutor,
		runningTriggers: make(map[string]models.Trigger),
		ctx:             ctx,
		cancel:          cancel,
		logger: log.WithFields(log.Fields{
			"module":    "worker_manager",
			"worker_id": workerID,
		}),
	}
}

func (wm *Manager) StartWorkers() error {
	workflows, err := wm.repository.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	if len(workflows) == 0 {
		wm.logger.Info("No workflows")
		return nil
	}

	wm.logger.Infof("Found %d workflows", len(workflows))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, workflow := range workflows {
		wg.Add(1)
		go func(wf *models.Workflow) {
			defer wg.Done()
			if err := wm.startWorkflowTriggers(wf); err != nil {
				wm.logger.Errorf("Failed to start workflow %s: %v", wf.ID, err)
			}
		}(workflow)
	}

	go func() {
		<-c
		wm.logger.Info("Shutting down...\n")
		wm.Stop()
	}()

	wg.Wait()
	wm.logger.Info("Stopped\n")
	return nil
}

func (wm *Manager) startWorkflowTriggers(workflow *models.Workflow) error {
	logger := wm.logger.WithFields(log.Fields{
		"workflow_id":   workflow.ID,
		"workflow_name": workflow.Name,
	})
	logger.Info("Starting triggers for workflow")

	for _, triggerItem := range workflow.Triggers {
		logger = logger.WithFields(log.Fields{
			"trigger_id":     triggerItem.ID,
			"trigger_type":   triggerItem.Type,
			"trigger_config": triggerItem.Configuration,
		})

		config := make(map[string]interface{})
		for k, v := range triggerItem.Configuration {
			config[k] = v
		}
		config["workflow_id"] = workflow.ID
		config["id"] = triggerItem.ID
		trigger, err := wm.registry.CreateTrigger(triggerItem.Type, config)
		if err != nil {
			logger.Errorf("Failed to create trigger: %v", err)
			continue
		}

		wm.triggerMutex.Lock()
		wm.runningTriggers[triggerItem.ID] = trigger
		wm.triggerMutex.Unlock()

		callback := wm.createWorkflowCallback(workflow.ID)

		if err := trigger.Start(wm.ctx, callback); err != nil {
			logger.Errorf("Failed to start trigger: %v", err)

			wm.triggerMutex.Lock()
			delete(wm.runningTriggers, triggerItem.ID)
			wm.triggerMutex.Unlock()
			continue
		}

		logger.Info("Started trigger")
	}

	<-wm.ctx.Done()
	return nil
}

func (wm *Manager) createWorkflowCallback(workflowID string) models.TriggerCallback {
	return func(ctx context.Context, data map[string]interface{}) error {
		wm.logger.Infof("Executing workflow triggered by %s", data["trigger_id"])
		err := wm.executor.Execute(ctx, workflowID, data)
		if err != nil {
			wm.logger.Errorf("Error executing workflow %s: %v", workflowID, err)
		}
		return nil
	}
}

func (wm *Manager) Stop() {
	wm.cancel()

	wm.triggerMutex.Lock()
	defer wm.triggerMutex.Unlock()

	for triggerID, trigger := range wm.runningTriggers {
		wm.logger.Infof("Stopping trigger %s\n", triggerID)
		if err := trigger.Stop(context.Background()); err != nil {
			wm.logger.Errorf("Error stopping trigger %s: %v", triggerID, err)
		}
	}

	// Clear running triggers
	wm.runningTriggers = make(map[string]models.Trigger)
}
