package workflow

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/registry"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Executor struct {
	repository *Repository
	registry   *registry.Registry
	tracer     trace.Tracer
}

func NewExecutor(repository *Repository, registry *registry.Registry) *Executor {
	return &Executor{
		repository: repository,
		registry:   registry,
		tracer:     trc.GetTracer("executor"),
	}
}

func (s *Executor) Execute(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
	ctx, span := trc.StartSpan(ctx, s.tracer, "workflow.execute",
		attribute.String(trc.WorkflowIDKey, workflowID),
	)
	defer span.End()

	logger := log.WithFields(log.Fields{
		"module":       "workflow_executor",
		"workflow_id":  workflowID,
		"trigger_data": triggerData,
	})

	logger.Info("Starting execution of workflow")
	span.AddEvent("workflow_execution_started")

	workflow, err := s.getWorkflowByID(workflowID)
	if err != nil {
		logger.Errorf("Failed to fetch workflow: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch workflow")
		return fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}

	span.SetAttributes(
		attribute.String(trc.WorkflowNameKey, workflow.Name),
		attribute.Int("workflow.step_count", len(workflow.Steps)),
	)
	span.AddEvent("workflow_loaded")

	executionCtx := models.ExecutionContext{
		ID:          generateExecutionID(),
		WorkflowID:  workflowID,
		TriggerData: triggerData,
		Variables:   workflow.Variables,
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	span.SetAttributes(attribute.String(trc.ExecutionIDKey, executionCtx.ID))

	logger = logger.WithFields(log.Fields{
		"execution_id": executionCtx.ID,
		"variables":    executionCtx.Variables,
	})
	executionCtx.Logger = logger

	logger.Info("Created execution context")
	span.AddEvent("execution_context_created")

	if len(workflow.Steps) == 0 {
		logger.Info("Workflow has no steps to execute")
		span.AddEvent("no_steps_to_execute")
		span.SetStatus(codes.Ok, "workflow executed successfully (no steps)")
		return nil
	}

	logger.Infof("Workflow has %d steps to execute", len(workflow.Steps))
	span.AddEvent("starting_step_execution", trace.WithAttributes(
		attribute.Int("total_steps", len(workflow.Steps)),
	))

	currentStepID := workflow.Steps[0].ID
	stepCount := 0
	
	logger.Infof("Starting execution with first step ID: %s", currentStepID)

	for currentStepID != "" {
		stepCount++
		stepCtx, stepSpan := trc.StartSpan(ctx, s.tracer, "workflow.execute_step",
			attribute.String(trc.WorkflowIDKey, workflowID),
			attribute.String(trc.ExecutionIDKey, executionCtx.ID),
			attribute.Int("step.sequence", stepCount),
		)

		step, found := s.findStepByID(workflow.Steps, currentStepID)
		if !found {
			stepSpan.RecordError(fmt.Errorf("step %s not found", currentStepID))
			stepSpan.SetStatus(codes.Error, "step not found")
			stepSpan.End()
			return fmt.Errorf("step %s not found in workflow %s", currentStepID, workflowID)
		}

		stepSpan.SetAttributes(
			append(trc.StepAttributes(step.ID, step.Name),
				append(trc.ActionAttributes(step.Action.ID, step.Action.Type),
					attribute.Bool("step.enabled", step.Enabled))...)...,
		)

		logger = logger.WithFields(log.Fields{
			"step_id":          step.ID,
			"step_action":      step.Action.Name,
			"step_action_type": step.Action.Type,
			"step_enabled":     step.Enabled,
			"step_conditional": step.Conditional,
			"step_on_success":  step.OnSuccess,
			"step_on_failure":  step.OnFailure,
		})

		logger.Infof("Executing step %d: %s (type: %s, enabled: %t)", stepCount, step.Name, step.Action.Type, step.Enabled)
		stepSpan.AddEvent("step_started")

		if !step.Enabled {
			logger.Info("Step is disabled, skipping")
			stepSpan.AddEvent("step_disabled_skipped")
			stepSpan.SetStatus(codes.Ok, "step disabled, skipped")
			stepSpan.End()
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		logger.Info("Executing step action")
		stepSpan.AddEvent("evaluating_conditional")

		shouldExecute, err := s.evaluateConditional(step.Conditional, executionCtx)
		if err != nil {
			log.Printf("Error evaluating conditional for step %s: %v", step.ID, err)
			stepSpan.RecordError(err)
			stepSpan.SetStatus(codes.Error, "conditional evaluation failed")
			stepSpan.End()
			return fmt.Errorf("error evaluating conditional for step %s: %w", step.ID, err)
		}

		if !shouldExecute {
			log.Printf("Conditional evaluated to false for step %s, skipping", step.ID)
			stepSpan.AddEvent("conditional_false_skipped")
			stepSpan.SetStatus(codes.Ok, "conditional false, skipped")
			stepSpan.End()
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		stepSpan.AddEvent("executing_action")
		result, err := s.executeAction(stepCtx, step.Action, &executionCtx)
		if err != nil {
			logger.Errorf("Failed to execute action for step: %v", err)
			stepSpan.RecordError(err)
			stepSpan.SetStatus(codes.Error, "action execution failed")
			stepSpan.End()
			return fmt.Errorf("failed to execute action for step %s: %w", step.ID, err)
		}

		if executionCtx.StepResults == nil {
			executionCtx.StepResults = make(map[string]interface{})
		}
		executionCtx.StepResults[step.Name] = result
		logger.Infof("Step executed successfully, result: %v", result)

		stepSpan.AddEvent("action_completed")
		stepSpan.SetStatus(codes.Ok, "step executed successfully")
		stepSpan.End()

		nextStepID := s.getNextStepID(step, true)
		logger.Infof("Moving to next step. Current: %s, Next: %s", currentStepID, nextStepID)
		currentStepID = nextStepID
	}

	logger.Infof("Completed execution of workflow (execution ID: %s)", executionCtx.ID)
	span.AddEvent("workflow_execution_completed", trace.WithAttributes(
		attribute.Int("steps_executed", stepCount),
	))
	span.SetStatus(codes.Ok, "workflow executed successfully")
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
	ctx, span := trc.StartSpan(ctx, s.tracer, "workflow.execute_action",
		append(trc.ActionAttributes(actionItem.ID, actionItem.Type),
			attribute.String(trc.WorkflowIDKey, executionCtx.WorkflowID),
			attribute.String(trc.ExecutionIDKey, executionCtx.ID))...,
	)
	defer span.End()

	if s.registry == nil {
		executionCtx.Logger.Infof("Registry not initialized, skipping action %s", actionItem.ID)
		span.AddEvent("registry_not_initialized")
		span.SetStatus(codes.Ok, "registry not initialized, action skipped")
		return nil, nil
	}

	span.AddEvent("preparing_action_config")
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

	span.AddEvent("creating_action_instance")
	logger.Infof("Creating action of type '%s' with config: %+v", actionItem.Type, config)
	action, err := s.registry.CreateActionWithContext(ctx, actionItem.Type, config)
	if err != nil {
		logger.Errorf("Failed to create action of type '%s': %v", actionItem.Type, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create action")
		return nil, err
	}
	logger.Infof("Successfully created action of type '%s'", actionItem.Type)

	span.AddEvent("executing_action")
	result, err := action.Execute(ctx, *executionCtx.WithLogger(logger))
	if err != nil {
		logger.Errorf("Action failed: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "action execution failed")
		return nil, err
	}

	logger.Info("Action completed successfully")
	span.AddEvent("action_completed")
	span.SetStatus(codes.Ok, "action executed successfully")
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
