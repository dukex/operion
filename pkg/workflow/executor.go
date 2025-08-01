// Package workflow provides workflow execution engine and repository management.
package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/registry"
	"github.com/google/uuid"
)

type Executor struct {
	persistence persistence.Persistence
	registry    *registry.Registry
}

func NewExecutor(
	persistence persistence.Persistence,
	registry *registry.Registry,
) *Executor {
	return &Executor{
		registry:    registry,
		persistence: persistence,
	}
}

func (s *Executor) Start(ctx context.Context, logger *slog.Logger, workflowID string, triggerData map[string]any) ([]eventbus.Event, error) {
	logger.Info("Starting execution of workflow")

	workflowRepository := NewRepository(s.persistence)
	workflowItem, err := workflowRepository.FetchByID(workflowID)

	if err != nil {
		logger.Error("Failed to get workflow", "error", err)
		return nil, err
	}

	logger.Info("Created execution context")

	if len(workflowItem.Steps) == 0 {
		logger.Info("Workflow has no steps to execute")
		return nil, fmt.Errorf("workflow %s has no steps", workflowID)
	}

	// TODO: save it to the database
	executionCtx := &models.ExecutionContext{
		ID:          generateExecutionID(),
		WorkflowID:  workflowID,
		TriggerData: triggerData,
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	return []eventbus.Event{
		&events.WorkflowStepAvailable{
			BaseEvent:        events.NewBaseEvent(events.WorkflowStepAvailableEvent, workflowID),
			ExecutionID:      executionCtx.ID,
			StepID:           workflowItem.Steps[0].ID,
			ExecutionContext: executionCtx,
		},
	}, nil
}

func (s *Executor) ExecuteStep(ctx context.Context, logger *slog.Logger, workflow *models.Workflow, executionCtx *models.ExecutionContext, currentStepID string) (
	[]eventbus.Event, error) {
	step, found := s.findStepByID(workflow.Steps, currentStepID)
	if !found {
		return nil, fmt.Errorf("step %s not found in workflow %s", currentStepID, workflow.ID)
	}

	logger = logger.With(
		"step_id", step.ID,
		"step_name", step.Name,
	)

	if !step.Enabled {
		logger.Info("Step is disabled, skipping")

		return s.nextStep(step, logger, workflow.ID, executionCtx, true), nil // Treat as success
	}

	logger.Info("Executing step action")

	// TODO: Handle conditionals
	// 	shouldExecute, err := s.evaluateConditional(step.Conditional, executionCtx)
	// 	if err != nil {
	// 		log.Printf("Error evaluating conditional for step %s: %v", step.ID, err)
	// 		return fmt.Errorf("error evaluating conditional for step %s: %w", step.ID, err)
	// 	}

	// 	if !shouldExecute {
	// 		log.Printf("Conditional evaluated to false for step %s, skipping", step.ID)
	// 		currentStepID = s.getNextStepID(step, true) // Treat as success
	// 		continue
	// 	}

	result, err := s.executeAction(ctx, logger, step, executionCtx)
	if err != nil {
		logger.Error("Failed to execute action for step", "error", err)

		return append(s.nextStep(step, logger, workflow.ID, executionCtx, false),
			&events.WorkflowStepFailed{
				BaseEvent:   events.NewBaseEvent(events.WorkflowStepFailedEvent, workflow.ID),
				ExecutionID: executionCtx.ID,
				StepID:      currentStepID,
				ActionID:    step.ActionID,
				Error:       err.Error(),
				Duration:    0, // TODO: calculate the duration
			},
		), fmt.Errorf("failed to execute action for step %s: %w", step.ID, err)
	}

	if executionCtx.StepResults == nil {
		executionCtx.StepResults = make(map[string]any)
	}
	executionCtx.StepResults[step.UID] = result
	logger.Info("Step executed successfully", "result", result)

	return append(s.nextStep(step, logger, workflow.ID, executionCtx, true),
		&events.WorkflowStepFinished{
			BaseEvent:   events.NewBaseEvent(events.WorkflowStepFinishedEvent, workflow.ID),
			ExecutionID: executionCtx.ID,
			StepID:      step.ID,
			ActionID:    step.ActionID,
			Result:      result,
			Duration:    0, // TODO: calculate the duration
		},
	), nil
}

func (s *Executor) nextStep(
	step *models.WorkflowStep,
	logger *slog.Logger,
	workflowId string,
	executionCtx *models.ExecutionContext,
	success bool,
) []eventbus.Event {
	nextStepID, found := s.getNextStepID(step, success)

	eventsToDispatcher := make([]eventbus.Event, 0)

	if !found {
		logger.Info("No next step defined, ending workflow execution")

		eventsToDispatcher = append(eventsToDispatcher, &events.WorkflowFinished{
			BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, workflowId),
			ExecutionID: executionCtx.ID,
			Result:      executionCtx.StepResults,
		})
	} else {
		logger.Info("Moving to next step", "next_step_id", nextStepID)

		eventsToDispatcher = append(eventsToDispatcher, &events.WorkflowStepAvailable{
			BaseEvent:        events.NewBaseEvent(events.WorkflowStepAvailableEvent, workflowId),
			ExecutionID:      executionCtx.ID,
			StepID:           nextStepID,
			ExecutionContext: executionCtx,
		})
	}

	return eventsToDispatcher
}

// func (s *Executor) getWorkflowByID(workflowID string) (*models.Workflow, error) {
// 	if s.repository == nil {
// 		return nil, fmt.Errorf("workflow repository not initialized")
// 	}

// 	workflow, err := s.repository.FetchByID(workflowID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if workflow.ID == "" {
// 		return nil, fmt.Errorf("workflow with ID %s not found", workflowID)
// 	}

// 	return workflow, nil
// }

func (s *Executor) findStepByID(steps []*models.WorkflowStep, stepID string) (*models.WorkflowStep, bool) {
	for _, step := range steps {
		if step.ID == stepID {
			return step, true
		}
	}
	return nil, false
}

// func (s *Executor) evaluateConditional(conditional models.ConditionalExpression, ctx models.ExecutionContext) (bool, error) {
// 	if conditional.Expression == "" && conditional.Language == "" {
// 		return true, nil
// 	}

// 	switch conditional.Language {
// 	case "simple", "":
// 		if conditional.Expression == "true" || conditional.Expression == "" {
// 			return true, nil
// 		}
// 		return false, nil
// 	default:
// 		// ctx.Logger.Errorf("Unsupported conditional language: %s, defaulting to true", conditional.Language)
// 		return true, nil
// 	}
// }

func (s *Executor) executeAction(ctx context.Context, logger *slog.Logger, step *models.WorkflowStep, executionCtx *models.ExecutionContext) (any, error) {
	if s.registry == nil {
		// executionCtx.Logger.Infof("Registry not initialized, skipping action %s", actionItem.ID)
		return nil, nil
	}

	config := make(map[string]any)
	maps.Copy(config, step.Configuration)
	config["id"] = step.ActionID

	logger = logger.With(
		"action_id", step.ActionID,
	)

	action, err := s.registry.CreateAction(step.ActionID, config)
	if err != nil {
		// logger.Errorf("Failed to create action: %v", err)
		return nil, err
	}

	result, err := action.Execute(ctx, *executionCtx, logger.With())
	if err != nil {
		// logger.Errorf("Actionfailed: %v", err)
		return nil, err
	}

	logger.Info("Action completed successfully")
	return result, err
}

func (s *Executor) getNextStepID(step *models.WorkflowStep, success bool) (string, bool) {
	if success && step.OnSuccess != nil {
		return *step.OnSuccess, true
	} else if !success && step.OnFailure != nil {
		return *step.OnFailure, true
	}

	// No next step specified, end workflow
	return "", false
}

func generateExecutionID() string {
	return fmt.Sprintf("exec-%s", uuid.New().String()[:8])
}
