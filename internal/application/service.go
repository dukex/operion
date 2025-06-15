package application

import (
	"context"
	"fmt"
	"log"

	"github.com/dukex/operion/internal/domain"
	"github.com/google/uuid"
)

// WorkflowService orchestrates workflow creation, retrieval, and execution
type WorkflowService struct {
	repository      *WorkflowRepository
	actionRegistry  *ActionRegistry
}
// NewWorkflowService creates a new workflow service with dependencies
func NewWorkflowService(repository *WorkflowRepository, actionRegistry *ActionRegistry) *WorkflowService {
	return &WorkflowService{
		repository:     repository,
		actionRegistry: actionRegistry,
	}
}

// ExecuteWorkflow executes a workflow with the given trigger data
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID string, triggerData map[string]interface{}) error {
	log.Printf("Starting execution of workflow %s", workflowID)

	// 1. Fetch workflow from repository
	workflow, err := s.getWorkflowByID(workflowID)
	if err != nil {
		return fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}

	// 2. Create an ExecutionContext
	executionCtx := domain.ExecutionContext{
		WorkflowID:  workflowID,
		ExecutionID: generateExecutionID(),
		TriggerData: triggerData,
		Variables:   workflow.Variables,
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	log.Printf("Created execution context %s for workflow %s", executionCtx.ExecutionID, workflowID)

	// 3. Start from the first step
	if len(workflow.Steps) == 0 {
		log.Printf("Workflow %s has no steps to execute", workflowID)
		return nil
	}

	currentStepID := workflow.Steps[0].ID
	
	// 4. Loop through steps based on OnSuccess/OnFailure paths
	for currentStepID != "" {
		step, found := s.findStepByID(workflow.Steps, currentStepID)
		if !found {
			return fmt.Errorf("step %s not found in workflow %s", currentStepID, workflowID)
		}

		if !step.Enabled {
			log.Printf("Step %s is disabled, skipping", step.ID)
			currentStepID = s.getNextStepID(step, true) // Treat as success
			continue
		}

		log.Printf("Executing step %s (%s)", step.ID, step.Action.Name)

		// 5. Evaluate conditions before executing
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

		// 6. Execute the action
		success := s.executeAction(ctx, step.Action, &executionCtx)
		
		// 7. Determine next step based on success/failure
		currentStepID = s.getNextStepID(step, success)
	}

	log.Printf("Completed execution of workflow %s (execution ID: %s)", workflowID, executionCtx.ExecutionID)
	return nil
}

// getWorkflowByID fetches a workflow by ID from the repository
func (s *WorkflowService) getWorkflowByID(workflowID string) (domain.Workflow, error) {
	if s.repository == nil {
		return domain.Workflow{}, fmt.Errorf("workflow repository not initialized")
	}

	workflows, err := s.repository.FetchAll()
	if err != nil {
		return domain.Workflow{}, err
	}

	for _, workflow := range workflows {
		if workflow.ID == workflowID {
			return workflow, nil
		}
	}

	return domain.Workflow{}, fmt.Errorf("workflow with ID %s not found", workflowID)
}

// findStepByID finds a workflow step by its ID
func (s *WorkflowService) findStepByID(steps []domain.WorkflowStep, stepID string) (domain.WorkflowStep, bool) {
	for _, step := range steps {
		if step.ID == stepID {
			return step, true
		}
	}
	return domain.WorkflowStep{}, false
}

// evaluateConditional evaluates a conditional expression
func (s *WorkflowService) evaluateConditional(conditional domain.ConditionalExpression, ctx domain.ExecutionContext) (bool, error) {
	// If no conditional is specified, always execute
	if conditional.Expression == "" && conditional.Language == "" {
		return true, nil
	}

	// For now, implement simple conditional evaluation
	// In a real implementation, you'd use a proper expression evaluator
	switch conditional.Language {
	case "simple", "":
		// Simple boolean evaluation - for demo purposes
		if conditional.Expression == "true" || conditional.Expression == "" {
			return true, nil
		}
		return false, nil
	default:
		log.Printf("Unsupported conditional language: %s, defaulting to true", conditional.Language)
		return true, nil
	}
}

// executeAction executes a workflow step action
func (s *WorkflowService) executeAction(ctx context.Context, actionItem domain.ActionItem, executionCtx *domain.ExecutionContext) bool {
	if s.actionRegistry == nil {
		log.Printf("Action registry not initialized, skipping action %s", actionItem.ID)
		return false
	}

	// Create action from registry - add ID to configuration
	config := make(map[string]interface{})
	for k, v := range actionItem.Configuration {
		config[k] = v
	}
	config["id"] = actionItem.ID

	action, err := s.actionRegistry.Create(actionItem.Type, config)
	if err != nil {
		log.Printf("Failed to create action %s of type %s: %v", actionItem.ID, actionItem.Type, err)
		return false
	}

	// Execute the action
	updatedCtx, err := action.Execute(ctx, *executionCtx)
	if err != nil {
		log.Printf("Action %s failed: %v", actionItem.ID, err)
		return false
	}

	// Update execution context with results
	*executionCtx = updatedCtx

	log.Printf("Action %s completed successfully", actionItem.ID)
	return true
}

// getNextStepID determines the next step based on success/failure
func (s *WorkflowService) getNextStepID(step domain.WorkflowStep, success bool) string {
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
