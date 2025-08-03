package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestApp(tempDir string) *fiber.App {
	persistence := file.NewFilePersistence(tempDir)

	app := NewAPI(
		slog.Default(),
		persistence,
		registry.NewRegistry(slog.Default()),
	)

	return app.App()
}

func TestAPI_RootEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Operion API", string(body))
}

func TestAPI_HealthCheck(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "OK", string(body))
}

func TestAPI_GetWorkflows_Empty(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var workflows []models.Workflow

	err = json.NewDecoder(resp.Body).Decode(&workflows)
	require.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestAPI_GetWorkflows_WithData(t *testing.T) {
	tempDir := t.TempDir()
	persistence := file.NewFilePersistence(tempDir)

	// Create test workflows
	workflow1 := &models.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow 1",
		Status: "active",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step1",
				Name:     "Log Step",
				ActionID: "log",
				UID:      "log_step",
				Configuration: map[string]any{
					"message": "Test message",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"test_var": "test_value",
		},
	}

	workflow2 := &models.Workflow{
		ID:     "test-workflow-2",
		Name:   "Test Workflow 2",
		Status: "inactive",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step1",
				Name:     "Transform Step",
				ActionID: "transform",
				UID:      "transform_step",
				Configuration: map[string]any{
					"expression": "{ \"result\": \"transformed\" }",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{},
	}

	// Save workflows
	repo := workflow.NewRepository(persistence)
	_, err := repo.Create(workflow1)
	require.NoError(t, err)
	_, err = repo.Create(workflow2)
	require.NoError(t, err)

	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var workflows []models.Workflow

	err = json.NewDecoder(resp.Body).Decode(&workflows)
	require.NoError(t, err)
	assert.Len(t, workflows, 2)

	// Verify workflow data
	workflowIDs := []string{workflows[0].ID, workflows[1].ID}
	assert.Contains(t, workflowIDs, "test-workflow-1")
	assert.Contains(t, workflowIDs, "test-workflow-2")
}

func TestAPI_GetWorkflow_Success(t *testing.T) {
	tempDir := t.TempDir()
	persistence := file.NewFilePersistence(tempDir)

	// Create test workflow
	workflow1 := &models.Workflow{
		ID:     "test-workflow-specific",
		Name:   "Specific Test Workflow",
		Status: "active",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step1",
				Name:     "HTTP Request Step",
				ActionID: "http_request",
				UID:      "http_step",
				Configuration: map[string]any{
					"protocol": "https",
					"host":     "api.example.com",
					"path":     "/data",
					"method":   "GET",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"api_key": "test-key",
		},
	}

	// Save workflow
	repo := workflow.NewRepository(persistence)
	_, err := repo.Create(workflow1)
	require.NoError(t, err)

	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-workflow-specific", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var returnedWorkflow models.Workflow

	err = json.NewDecoder(resp.Body).Decode(&returnedWorkflow)
	require.NoError(t, err)

	assert.Equal(t, "test-workflow-specific", returnedWorkflow.ID)
	assert.Equal(t, "Specific Test Workflow", returnedWorkflow.Name)
	assert.Equal(t, models.WorkflowStatus("active"), returnedWorkflow.Status)
	assert.Len(t, returnedWorkflow.Steps, 1)
	assert.Equal(t, "http_request", returnedWorkflow.Steps[0].ActionID)
	assert.Equal(t, "test-key", returnedWorkflow.Variables["api_key"])
}

func TestAPI_GetWorkflow_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows/non-existent-workflow", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAPI_GetWorkflow_InvalidID(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows/", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	// Should return the list of workflows instead
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAPI_CORS_Headers(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodOptions, "/workflows", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestAPI_ContentType_JSON(t *testing.T) {
	tempDir := t.TempDir()
	app := setupTestApp(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestAPI_Integration_WorkflowLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	persistence := file.NewFilePersistence(tempDir)

	// Create a comprehensive test workflow
	complexWorkflow := &models.Workflow{
		ID:          "integration-test-workflow",
		Name:        "Integration Test Workflow",
		Description: "A comprehensive workflow for integration testing",
		Status:      "active",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step1",
				Name:     "Log Initial Message",
				ActionID: "log",
				UID:      "initial_log",
				Configuration: map[string]any{
					"message": "Starting integration test workflow",
				},
				OnSuccess: stringPtr("step2"),
				Enabled:   true,
			},
			{
				ID:       "step2",
				Name:     "HTTP API Call",
				ActionID: "http_request",
				UID:      "api_call",
				Configuration: map[string]any{
					"protocol": "https",
					"host":     "httpbin.org",
					"path":     "/json",
					"method":   "GET",
					"timeout":  30,
				},
				OnSuccess: stringPtr("step3"),
				OnFailure: stringPtr("step4"),
				Enabled:   true,
			},
			{
				ID:       "step3",
				Name:     "Transform Response",
				ActionID: "transform",
				UID:      "transform_response",
				Configuration: map[string]any{
					"expression": "{ \"processed\": true, \"original\": \"{{.step_results.api_call.body}}\" }",
				},
				OnSuccess: stringPtr("step5"),
				Enabled:   true,
			},
			{
				ID:       "step4",
				Name:     "Log Error",
				ActionID: "log",
				UID:      "error_log",
				Configuration: map[string]any{
					"message": "API call failed",
				},
				Enabled: true,
			},
			{
				ID:       "step5",
				Name:     "Final Log",
				ActionID: "log",
				UID:      "final_log",
				Configuration: map[string]any{
					"message": "Integration test completed successfully",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"environment": "test",
			"version":     "1.0.0",
			"config": map[string]any{
				"retry_attempts": 3,
				"timeout":        30,
			},
		},
		WorkflowTriggers: []*models.WorkflowTrigger{
			{
				ID:        "integration-test-trigger",
				Name:      "Integration Test Trigger",
				TriggerID: "schedule",
				Configuration: map[string]any{
					"schedule": "0 0 * * *",
				},
			},
		},
	}

	// Save workflow
	repo := workflow.NewRepository(persistence)
	_, err := repo.Create(complexWorkflow)
	require.NoError(t, err)

	app := setupTestApp(tempDir)

	// Test 1: Fetch all workflows and verify our workflow is there
	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var workflows []models.Workflow

	err = json.NewDecoder(resp.Body).Decode(&workflows)
	require.NoError(t, err)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "integration-test-workflow", workflows[0].ID)

	// Test 2: Fetch specific workflow and verify all details
	req = httptest.NewRequest(http.MethodGet, "/workflows/integration-test-workflow", nil)
	req.Header.Set("Accept", "application/json")
	resp, err = app.Test(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var fetchedWorkflow models.Workflow

	err = json.NewDecoder(resp.Body).Decode(&fetchedWorkflow)
	require.NoError(t, err)

	// Verify workflow structure
	assert.Equal(t, "integration-test-workflow", fetchedWorkflow.ID)
	assert.Equal(t, "Integration Test Workflow", fetchedWorkflow.Name)
	assert.Equal(t, "A comprehensive workflow for integration testing", fetchedWorkflow.Description)
	assert.Equal(t, models.WorkflowStatus("active"), fetchedWorkflow.Status)
	assert.Len(t, fetchedWorkflow.Steps, 5)
	assert.Len(t, fetchedWorkflow.WorkflowTriggers, 1)

	// Verify step configurations
	logStep := fetchedWorkflow.Steps[0]
	assert.Equal(t, "log", logStep.ActionID)
	assert.Equal(t, "Starting integration test workflow", logStep.Configuration["message"])

	httpStep := fetchedWorkflow.Steps[1]
	assert.Equal(t, "http_request", httpStep.ActionID)
	assert.Equal(t, "https", httpStep.Configuration["protocol"])
	assert.Equal(t, "httpbin.org", httpStep.Configuration["host"])

	// Verify variables
	assert.Equal(t, "test", fetchedWorkflow.Variables["environment"])
	assert.Equal(t, "1.0.0", fetchedWorkflow.Variables["version"])

	config, ok := fetchedWorkflow.Variables["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(3), config["retry_attempts"])

	// Verify triggers
	trigger := fetchedWorkflow.WorkflowTriggers[0]
	assert.Equal(t, "schedule", trigger.TriggerID)
	assert.Equal(t, "0 0 * * *", trigger.Configuration["schedule"])
}

// Helper function to create string pointers.
func stringPtr(s string) *string {
	return &s
}
