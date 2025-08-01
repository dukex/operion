package workflow

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_ConditionalFalseRoutesToErrorPipeline(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-repo-conditional")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with conditional that evaluates to false
	workflow := &models.Workflow{
		ID:   "conditional-false-workflow",
		Name: "Conditional False Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "admin-check",
				Name:     "Admin Only Step",
				ActionID: "log",
				UID:      "admin_step",
				Configuration: map[string]any{
					"message": "This should not execute",
				},
				Conditional: models.ConditionalExpression{
					Language:   "simple",
					Expression: "{{eq .trigger_data.webhook.user.role \"admin\"}}",
				},
				OnSuccess: stringPtr("success-step"),
				OnFailure: stringPtr("error-step"),
				Enabled:   true,
			},
			{
				ID:       "success-step",
				Name:     "Success Step",
				ActionID: "log",
				UID:      "success_step",
				Configuration: map[string]any{
					"message": "Success path executed",
				},
				Enabled: true,
			},
			{
				ID:       "error-step",
				Name:     "Error Step",
				ActionID: "log",
				UID:      "error_step",
				Configuration: map[string]any{
					"message": "Error path executed",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Create execution context with non-admin user
	executionCtx := &models.ExecutionContext{
		ID:         "test-exec-false",
		WorkflowID: "conditional-false-workflow",
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"user": map[string]any{
					"role": "user", // Not admin
					"id":   456,
				},
			},
		},
		StepResults: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
	}

	// Execute step - conditional should evaluate to false and route to error pipeline
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, executionCtx, "admin-check")

	require.NoError(t, err, "ExecuteStep should not return error for conditional false")
	require.Len(t, eventList, 2, "Should return 2 events: next step + failed step")

	// Check that we have a WorkflowStepFailed event
	var stepFailedEvent *events.WorkflowStepFailed
	var nextStepEvent *events.WorkflowStepAvailable
	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFailed:
			stepFailedEvent = e
		case *events.WorkflowStepAvailable:
			nextStepEvent = e
		}
	}

	require.NotNil(t, stepFailedEvent, "Should have WorkflowStepFailed event")
	require.NotNil(t, nextStepEvent, "Should have WorkflowStepAvailable event")

	// Verify the failed step event
	assert.Equal(t, "admin-check", stepFailedEvent.StepID)
	assert.Equal(t, "conditional expression evaluated to false", stepFailedEvent.Error)
	assert.Equal(t, "test-exec-false", stepFailedEvent.ExecutionID)

	// Verify next step routes to error pipeline (on_failure)
	assert.Equal(t, "error-step", nextStepEvent.StepID, "Should route to error step (on_failure)")

	// Clean up
	err = repo.Delete("conditional-false-workflow")
	assert.NoError(t, err)
	cleanupTestDirectory("./test-repo-conditional")
}

func TestExecutor_ConditionalTrueExecutesStep(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-repo-conditional-true")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with conditional that evaluates to true
	workflow := &models.Workflow{
		ID:   "conditional-true-workflow",
		Name: "Conditional True Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "admin-check",
				Name:     "Admin Only Step",
				ActionID: "log",
				UID:      "admin_step",
				Configuration: map[string]any{
					"message": "Admin access granted",
				},
				Conditional: models.ConditionalExpression{
					Language:   "simple",
					Expression: "{{eq .trigger_data.webhook.user.role \"admin\"}}",
				},
				OnSuccess: stringPtr("success-step"),
				OnFailure: stringPtr("error-step"),
				Enabled:   true,
			},
			{
				ID:       "success-step",
				Name:     "Success Step",
				ActionID: "log",
				UID:      "success_step",
				Configuration: map[string]any{
					"message": "Success path executed",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Create execution context with admin user
	executionCtx := &models.ExecutionContext{
		ID:         "test-exec-true",
		WorkflowID: "conditional-true-workflow",
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"user": map[string]any{
					"role": "admin", // Admin user
					"id":   123,
				},
			},
		},
		StepResults: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
	}

	// Execute step - conditional should evaluate to true and execute action
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, executionCtx, "admin-check")

	require.NoError(t, err)
	require.Len(t, eventList, 2, "Should return 2 events: next step + finished step")

	// Check that we have a WorkflowStepFinished event (not failed)
	var stepFinishedEvent *events.WorkflowStepFinished
	var nextStepEvent *events.WorkflowStepAvailable
	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFinished:
			stepFinishedEvent = e
		case *events.WorkflowStepAvailable:
			nextStepEvent = e
		case *events.WorkflowStepFailed:
			t.Errorf("Should not have WorkflowStepFailed event when conditional is true")
		}
	}

	require.NotNil(t, stepFinishedEvent, "Should have WorkflowStepFinished event")
	require.NotNil(t, nextStepEvent, "Should have WorkflowStepAvailable event")

	// Verify the finished step event
	assert.Equal(t, "admin-check", stepFinishedEvent.StepID)
	assert.Equal(t, "test-exec-true", stepFinishedEvent.ExecutionID)

	// Verify next step routes to success pipeline (on_success)
	assert.Equal(t, "success-step", nextStepEvent.StepID, "Should route to success step (on_success)")

	// Verify step result was stored
	assert.Contains(t, executionCtx.StepResults, "admin_step", "Step result should be stored")

	// Clean up
	err = repo.Delete("conditional-true-workflow")
	assert.NoError(t, err)
	cleanupTestDirectory("./test-repo-conditional-true")
}

func TestExecutor_ConditionalEmptyDefaultsToTrue(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-repo-conditional-empty")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with no conditional (empty)
	workflow := &models.Workflow{
		ID:   "no-conditional-workflow",
		Name: "No Conditional Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "always-execute",
				Name:     "Always Execute Step",
				ActionID: "log",
				UID:      "always_step",
				Configuration: map[string]any{
					"message": "This should always execute",
				},
				// No conditional defined - should default to executing
				OnSuccess: stringPtr("success-step"),
				Enabled:   true,
			},
			{
				ID:       "success-step",
				Name:     "Success Step",
				ActionID: "log",
				UID:      "success_step",
				Configuration: map[string]any{
					"message": "Success path executed",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Create execution context
	executionCtx := &models.ExecutionContext{
		ID:          "test-exec-empty",
		WorkflowID:  "no-conditional-workflow",
		TriggerData: map[string]any{},
		StepResults: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
	}

	// Execute step - should always execute since no conditional is defined
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, executionCtx, "always-execute")

	require.NoError(t, err)
	require.Len(t, eventList, 2, "Should return 2 events: next step + finished step")

	// Check that we have a WorkflowStepFinished event (not failed)
	var stepFinishedEvent *events.WorkflowStepFinished
	var nextStepEvent *events.WorkflowStepAvailable
	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFinished:
			stepFinishedEvent = e
		case *events.WorkflowStepAvailable:
			nextStepEvent = e
		case *events.WorkflowStepFailed:
			t.Errorf("Should not have WorkflowStepFailed event when no conditional is defined")
		}
	}

	require.NotNil(t, stepFinishedEvent, "Should have WorkflowStepFinished event")
	require.NotNil(t, nextStepEvent, "Should have WorkflowStepAvailable event")

	// Verify the finished step event
	assert.Equal(t, "always-execute", stepFinishedEvent.StepID)
	assert.Equal(t, "test-exec-empty", stepFinishedEvent.ExecutionID)

	// Verify next step routes to success pipeline
	assert.Equal(t, "success-step", nextStepEvent.StepID, "Should route to success step")

	// Verify step result was stored
	assert.Contains(t, executionCtx.StepResults, "always_step", "Step result should be stored")

	// Clean up
	err = repo.Delete("no-conditional-workflow")
	assert.NoError(t, err)
	cleanupTestDirectory("./test-repo-conditional-empty")
}

func TestExecutor_ConditionalEvaluationError(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-repo-conditional-error")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with invalid conditional
	workflow := &models.Workflow{
		ID:   "conditional-error-workflow",
		Name: "Conditional Error Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "invalid-conditional",
				Name:     "Invalid Conditional Step",
				ActionID: "log",
				UID:      "invalid_step",
				Configuration: map[string]any{
					"message": "This should not execute",
				},
				Conditional: models.ConditionalExpression{
					Language:   "simple",
					Expression: "{{.trigger_data.webhook.nonexistent.field}}", // Invalid template
				},
				OnSuccess: stringPtr("success-step"),
				OnFailure: stringPtr("error-step"),
				Enabled:   true,
			},
			{
				ID:       "error-step",
				Name:     "Error Step",
				ActionID: "log",
				UID:      "error_step",
				Configuration: map[string]any{
					"message": "Error handling executed",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Create execution context
	executionCtx := &models.ExecutionContext{
		ID:         "test-exec-error",
		WorkflowID: "conditional-error-workflow",
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"user": map[string]any{
					"role": "admin",
				},
			},
		},
		StepResults: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
	}

	// Execute step - conditional evaluation should fail and return error
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, executionCtx, "invalid-conditional")

	require.Error(t, err, "ExecuteStep should return error for conditional evaluation failure")
	require.Len(t, eventList, 2, "Should return 2 events: next step + failed step")

	// Check that we have a WorkflowStepFailed event
	var stepFailedEvent *events.WorkflowStepFailed
	var nextStepEvent *events.WorkflowStepAvailable
	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFailed:
			stepFailedEvent = e
		case *events.WorkflowStepAvailable:
			nextStepEvent = e
		}
	}

	require.NotNil(t, stepFailedEvent, "Should have WorkflowStepFailed event")
	require.NotNil(t, nextStepEvent, "Should have WorkflowStepAvailable event")

	// Verify the failed step event
	assert.Equal(t, "invalid-conditional", stepFailedEvent.StepID)
	assert.Contains(t, stepFailedEvent.Error, "conditional evaluation failed")
	assert.Equal(t, "test-exec-error", stepFailedEvent.ExecutionID)

	// Verify next step routes to error pipeline (on_failure)
	assert.Equal(t, "error-step", nextStepEvent.StepID, "Should route to error step (on_failure)")

	// Verify error message contains details
	assert.Contains(t, err.Error(), "error evaluating conditional for step")

	// Clean up
	err = repo.Delete("conditional-error-workflow")
	assert.NoError(t, err)
	cleanupTestDirectory("./test-repo-conditional-error")
}

func TestExecutor_ConditionalComplexScenarios(t *testing.T) {
	tests := []struct {
		name         string
		expression   string
		executionCtx *models.ExecutionContext
		expectTrue   bool
		description  string
	}{
		{
			name:       "API status check",
			expression: "{{and (gt .step_results.api_call.status 199) (lt .step_results.api_call.status 300)}}",
			executionCtx: &models.ExecutionContext{
				ID:         "test-api-check",
				WorkflowID: "api-workflow",
				StepResults: map[string]any{
					"api_call": map[string]any{
						"status": 200,
						"body":   map[string]any{"success": true},
					},
				},
			},
			expectTrue:  true,
			description: "API status 200 should be considered success",
		},
		{
			name:       "Multi-condition validation",
			expression: "{{and (eq .variables.environment \"production\") (eq .trigger_data.webhook.verified true) (gt .step_results.user_count 0)}}",
			executionCtx: &models.ExecutionContext{
				ID:         "test-multi-condition",
				WorkflowID: "production-workflow",
				Variables: map[string]any{
					"environment": "production",
				},
				TriggerData: map[string]any{
					"webhook": map[string]any{
						"verified": true,
					},
				},
				StepResults: map[string]any{
					"user_count": 42,
				},
			},
			expectTrue:  true,
			description: "All conditions met for production deployment",
		},
		{
			name:       "Feature flag check",
			expression: "{{or (eq .variables.feature_enabled true) (eq .trigger_data.webhook.user.role \"admin\")}}",
			executionCtx: &models.ExecutionContext{
				ID:         "test-feature-flag",
				WorkflowID: "feature-workflow",
				Variables: map[string]any{
					"feature_enabled": false,
				},
				TriggerData: map[string]any{
					"webhook": map[string]any{
						"user": map[string]any{
							"role": "admin",
						},
					},
				},
			},
			expectTrue:  true,
			description: "Admin should bypass feature flag",
		},
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persistence := file.NewFilePersistence("./test-repo-complex-" + tt.name)
			registry := createTestRegistry()
			executor := NewExecutor(persistence, registry)

			workflowID := "complex-conditional-" + tt.name
			workflow := &models.Workflow{
				ID:   workflowID,
				Name: "Complex Conditional Test",
				Steps: []*models.WorkflowStep{
					{
						ID:       "complex-step",
						Name:     "Complex Conditional Step",
						ActionID: "log",
						UID:      "complex_step",
						Configuration: map[string]any{
							"message": "Complex conditional executed",
						},
						Conditional: models.ConditionalExpression{
							Language:   "simple",
							Expression: tt.expression,
						},
						OnSuccess: stringPtr("success-step"),
						OnFailure: stringPtr("error-step"),
						Enabled:   true,
					},
				},
			}

			// Save workflow
			repo := NewRepository(persistence)
			_, err := repo.Create(workflow)
			require.NoError(t, err)

			// Execute step
			eventList, err := executor.ExecuteStep(ctx, logger, workflow, tt.executionCtx, "complex-step")

			if tt.expectTrue {
				require.NoError(t, err, "Should not error when conditional is true: %s", tt.description)

				// Should have WorkflowStepFinished event
				var hasFinished bool
				for _, event := range eventList {
					if _, ok := event.(*events.WorkflowStepFinished); ok {
						hasFinished = true
						break
					}
				}
				assert.True(t, hasFinished, "Should have WorkflowStepFinished event: %s", tt.description)
			} else {
				// Should route to error pipeline
				var hasAvailable bool
				for _, event := range eventList {
					if stepAvailable, ok := event.(*events.WorkflowStepAvailable); ok {
						assert.Equal(t, "error-step", stepAvailable.StepID, "Should route to error step: %s", tt.description)
						hasAvailable = true
						break
					}
				}
				assert.True(t, hasAvailable, "Should have next step available: %s", tt.description)
			}

			// Clean up
			err = repo.Delete(workflowID)
			assert.NoError(t, err)
			cleanupTestDirectory("./test-repo-complex-" + tt.name)
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func cleanupTestDirectory(path string) {
	_ = os.RemoveAll(path)
}
