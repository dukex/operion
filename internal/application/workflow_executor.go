package application

import (
	"context"
	"fmt"

	"github.com/dukex/operion/internal/domain"
	"github.com/dukex/operion/pkg/registry"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type WorkflowExecutor struct {
	repository     *WorkflowRepository
	actionRegistry *registry.ActionRegistry
}

func NewWorkflowExecutor(repository *WorkflowRepository, actionRegistry *registry.ActionRegistry) *WorkflowExecutor {
	return &WorkflowExecutor{
		repository:     repository,
		actionRegistry: actionRegistry,
	}
}

func (s *WorkflowExecutor) Execute(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
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

	executionCtx := domain.ExecutionContext{
		WorkflowID:  workflowID,
		ExecutionID: generateExecutionID(),
		TriggerData: triggerData,
		Variables:   workflow.Variables,
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	logger = logger.WithFields(log.Fields{
		"execution_id": executionCtx.ExecutionID,
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
			currentStepID = s.getNextStepID(step, false)
			continue
		}

		if !shouldExecute {
			log.Printf("Conditional evaluated to false for step %s, skipping", step.ID)
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		result, err := s.executeAction(ctx, step.Action, &executionCtx)
		if err != nil {
			logger.Errorf("Failed to execute action for step: %v", err)
			currentStepID = s.getNextStepID(step, false)
			continue
		}

		if executionCtx.StepResults == nil {
			executionCtx.StepResults = make(map[string]interface{})
		}
		executionCtx.StepResults[step.Name] = result
		logger.Infof("Step executed successfully, result: %v", result)

		currentStepID = s.getNextStepID(step, true)
	}

	log.Printf("Completed execution of workflow %s (execution ID: %s)", workflowID, executionCtx.ExecutionID)
	return nil
}

// getWorkflowByID fetches a workflow by ID from the repository
func (s *WorkflowExecutor) getWorkflowByID(workflowID string) (*domain.Workflow, error) {
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

func (s *WorkflowExecutor) findStepByID(steps []domain.WorkflowStep, stepID string) (domain.WorkflowStep, bool) {
	for _, step := range steps {
		if step.ID == stepID {
			return step, true
		}
	}
	return domain.WorkflowStep{}, false
}

// evaluateConditional evaluates a conditional expression
func (s *WorkflowExecutor) evaluateConditional(conditional domain.ConditionalExpression, ctx domain.ExecutionContext) (bool, error) {
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

func (s *WorkflowExecutor) executeAction(ctx context.Context, actionItem domain.ActionItem, executionCtx *domain.ExecutionContext) (interface{}, error) {
	if s.actionRegistry == nil {
		executionCtx.Logger.Infof("Action registry not initialized, skipping action %s", actionItem.ID)
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

	action, err := s.actionRegistry.Create(actionItem.Type, config)
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
func (s *WorkflowExecutor) getNextStepID(step domain.WorkflowStep, success bool) string {
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
