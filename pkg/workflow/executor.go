package workflow

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/registry"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Executor struct {
	repository *Repository
	registry   *registry.Registry
}

func NewExecutor(repository *Repository, registry *registry.Registry) *Executor {
	return &Executor{
		repository: repository,
		registry:   registry,
	}
}

func (s *Executor) Execute(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
	logger := log.WithFields(log.Fields{
		"module":       "workflow_executor",
		"workflow_id":  workflowID,
		"trigger_data": triggerData,
	})

	logger.Info("Starting execution of workflow")

	workflow, err := s.getWorkflowByID(workflowID)
	if err != nil {
		logger.Errorf("Failed to fetch workflow: %v", err)
		return fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}

	executionCtx := models.ExecutionContext{
		ID:          generateExecutionID(),
		WorkflowID:  workflowID,
		TriggerData: triggerData,
		Variables:   workflow.Variables,
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	logger = logger.WithFields(log.Fields{
		"execution_id": executionCtx.ID,
		"variables":    executionCtx.Variables,
	})
	executionCtx.Logger = logger

	logger.Info("Created execution context")

	if len(workflow.Steps) == 0 {
		logger.Info("Workflow has no steps to execute")
		return nil
	}

	currentStepID := workflow.Steps[0].ID

	for currentStepID != "" {
		step, found := s.findStepByID(workflow.Steps, currentStepID)
		if !found {
			return fmt.Errorf("step %s not found in workflow %s", currentStepID, workflowID)
		}

		logger = logger.WithFields(log.Fields{
			"step_id":          step.ID,
			"step_action":      step.Action.Name,
			"step_enabled":     step.Enabled,
			"step_conditional": step.Conditional,
			"step_on_success":  step.OnSuccess,
			"step_on_failure":  step.OnFailure,
		})

		if !step.Enabled {
			logger.Info("Step is disabled, skipping")
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		logger.Info("Executing step action")

		shouldExecute, err := s.evaluateConditional(step.Conditional, executionCtx)
		if err != nil {
			log.Printf("Error evaluating conditional for step %s: %v", step.ID, err)
			return fmt.Errorf("error evaluating conditional for step %s: %w", step.ID, err)
		}

		if !shouldExecute {
			log.Printf("Conditional evaluated to false for step %s, skipping", step.ID)
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		result, err := s.executeAction(ctx, step.Action, &executionCtx)
		if err != nil {
			logger.Errorf("Failed to execute action for step: %v", err)
			return fmt.Errorf("failed to execute action for step %s: %w", step.ID, err)
		}

		if executionCtx.StepResults == nil {
			executionCtx.StepResults = make(map[string]interface{})
		}
		executionCtx.StepResults[step.UID] = result
		logger.Infof("Step executed successfully, result: %v", result)

		currentStepID = s.getNextStepID(step, true)
	}

	logger.Infof("Completed execution of workflow (execution ID: %s)", executionCtx.ID)
	return nil
}

func (s *Executor) getWorkflowByID(workflowID string) (*models.Workflow, error) {
	if s.repository == nil {
		return nil, fmt.Errorf("workflow repository not initialized")
	}

	workflow, err := s.repository.FetchByID(workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.ID == "" {
		return nil, fmt.Errorf("workflow with ID %s not found", workflowID)
	}

	return workflow, nil
}

func (s *Executor) findStepByID(steps []models.WorkflowStep, stepID string) (models.WorkflowStep, bool) {
	for _, step := range steps {
		if step.ID == stepID {
			return step, true
		}
	}
	return models.WorkflowStep{}, false
}

func (s *Executor) evaluateConditional(conditional models.ConditionalExpression, ctx models.ExecutionContext) (bool, error) {
	if conditional.Expression == "" && conditional.Language == "" {
		return true, nil
	}

	switch conditional.Language {
	case "simple", "":
		if conditional.Expression == "true" || conditional.Expression == "" {
			return true, nil
		}
		return false, nil
	default:
		ctx.Logger.Errorf("Unsupported conditional language: %s, defaulting to true", conditional.Language)
		return true, nil
	}
}

func (s *Executor) executeAction(ctx context.Context, actionItem models.ActionItem, executionCtx *models.ExecutionContext) (interface{}, error) {
	if s.registry == nil {
		executionCtx.Logger.Infof("Registry not initialized, skipping action %s", actionItem.ID)
		return nil, nil
	}

	config := make(map[string]interface{})
	for k, v := range actionItem.Configuration {
		config[k] = v
	}
	config["id"] = actionItem.ID

	logger := executionCtx.Logger.WithFields(log.Fields{
		"action_id":     actionItem.ID,
		"action_type":   actionItem.Type,
		"action_config": config,
	})

	action, err := s.registry.CreateAction(actionItem.Type, config)
	if err != nil {
		logger.Errorf("Failed to create action: %v", err)
		return nil, err
	}

	result, err := action.Execute(ctx, *executionCtx.WithLogger(logger))
	if err != nil {
		logger.Errorf("Actionfailed: %v", err)
		return nil, err
	}

	logger.Info("Action completed successfully")
	return result, err
}

// getNextStepID determines the next step based on success/failure
func (s *Executor) getNextStepID(step models.WorkflowStep, success bool) string {
	if success && step.OnSuccess != nil {
		return *step.OnSuccess
	} else if !success && step.OnFailure != nil {
		return *step.OnFailure
	}

	// No next step specified, end workflow
	return ""
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec-%s", uuid.New().String()[:8])
}
