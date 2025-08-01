package postgresql_test

import (
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/postgresql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create string pointers.
func stringPtr(s string) *string {
	return &s
}

func getTestDatabaseURL() string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// Skip tests if no test database URL is provided
		return ""
	}

	return url
}

func setupTestDB(t *testing.T) *postgresql.Persistence {
	t.Helper()

	databaseURL := getTestDatabaseURL()
	if databaseURL == "" {
		t.Skip("Skipping PostgreSQL tests - TEST_DATABASE_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	persistence, err := postgresql.NewPersistence(t.Context(), logger, databaseURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		for _, table := range []string{"workflows", "schema_migrations"} {
			db, err := sql.Open("postgres", databaseURL)
			require.NoError(t, err)

			_, err = db.ExecContext(t.Context(), "TRUNCATE TABLE "+table)
			require.NoError(t, err)

			err = db.Close()
			require.NoError(t, err)
		}

		err = persistence.Close(t.Context())
		require.NoError(t, err)
	})

	return persistence
}

func TestNewPersistence_Migrations(t *testing.T) {
	if getTestDatabaseURL() == "" {
		t.Skip("Skipping PostgreSQL tests - TEST_DATABASE_URL not set")
	}

	databaseURL := getTestDatabaseURL()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Connect directly to database to test migration
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)

	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	// Drop tables to test fresh migration
	_, _ = db.ExecContext(t.Context(), "DROP TABLE IF EXISTS workflows")
	_, _ = db.ExecContext(t.Context(), "DROP TABLE IF EXISTS schema_migrations")

	// Create new persistence instance (should run migrations)
	persistence, err := postgresql.NewPersistence(t.Context(), logger, databaseURL)
	require.NoError(t, err)

	defer func() {
		err := persistence.Close(t.Context())
		require.NoError(t, err)
	}()

	// Verify tables were created
	var exists bool

	err = db.QueryRowContext(t.Context(), `SELECT EXISTS (SELECT FROM
information_schema.tables WHERE table_name = 'workflows')`).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "workflows table should exist")

	err = db.QueryRowContext(t.Context(), `SELECT EXISTS (SELECT FROM
information_schema.tables WHERE table_name = 'schema_migrations')`).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "schema_migrations table should exist")

	// Verify migration version was recorded
	var version int

	err = db.QueryRowContext(t.Context(), "SELECT version FROM schema_migrations WHERE version = 1").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestNewPersistence_HealthCheck(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	err := p.HealthCheck(t.Context())
	assert.NoError(t, err)
}

func TestNewPersistence_SaveAndRetrieveWorkflow(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	workflow := &models.Workflow{
		ID:          "test-workflow-1",
		Name:        "Test Workflow",
		Description: "A test workflow",
		WorkflowTriggers: []*models.WorkflowTrigger{
			{
				ID:          uuid.New().String(),
				Name:        "Daily Schedule",
				Description: "Runs daily at midnight",
				TriggerID:   "schedule",
				Configuration: map[string]any{
					"cron": "0 0 * * *",
				},
			},
		},
		Steps: []*models.WorkflowStep{
			{
				ID:       uuid.New().String(),
				UID:      "step1",
				ActionID: "log",
				Name:     "Log Message",
				Configuration: map[string]any{
					"message": "Hello World",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"test_var": "test_value",
		},
		Status: models.WorkflowStatusActive,
		Metadata: map[string]any{
			"created_by": "test",
		},
		Owner: "test-user",
	}

	// Test saving workflow
	err := p.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())

	// Test retrieving workflow by ID
	retrieved, err := p.WorkflowByID(t.Context(), "test-workflow-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, workflow.Name, retrieved.Name)
	assert.Equal(t, workflow.Description, retrieved.Description)
	assert.Equal(t, workflow.Status, retrieved.Status)
	assert.Equal(t, workflow.Owner, retrieved.Owner)
	assert.Len(t, retrieved.WorkflowTriggers, len(workflow.WorkflowTriggers))
	assert.Len(t, retrieved.Steps, len(workflow.Steps))
	assert.Equal(t, workflow.Variables["test_var"], retrieved.Variables["test_var"])
	assert.Equal(t, workflow.Metadata["created_by"], retrieved.Metadata["created_by"])

	// Test retrieving non-existent workflow
	notFound, err := p.WorkflowByID(t.Context(), "non-existent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestNewPersistence_UpdateWorkflow(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	workflow := &models.Workflow{
		ID:          "test-workflow-2",
		Name:        "Test Workflow",
		Description: "A test workflow",
		WorkflowTriggers: []*models.WorkflowTrigger{
			{
				ID:          uuid.New().String(),
				Name:        "Daily Schedule",
				Description: "Runs daily at midnight",
				TriggerID:   "schedule",
				Configuration: map[string]any{
					"cron": "0 0 * * *",
				},
			},
		},
		Steps: []*models.WorkflowStep{
			{
				ID:       uuid.New().String(),
				UID:      "step1",
				ActionID: "log",
				Name:     "Log Message",
				Configuration: map[string]any{
					"message": "Hello World",
				},
				Enabled: true,
			},
		},
		Status: models.WorkflowStatusActive,
		Owner:  "test-user",
	}

	// Save initial workflow
	err := p.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	initialUpdatedAt := workflow.UpdatedAt

	// Wait a moment to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update workflow
	workflow.Name = "Updated Test Workflow"
	workflow.Description = "An updated test workflow"
	workflow.Status = models.WorkflowStatusPaused

	err = p.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Verify update
	retrieved, err := p.WorkflowByID(t.Context(), "test-workflow-2")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, "Updated Test Workflow", retrieved.Name)
	assert.Equal(t, "An updated test workflow", retrieved.Description)
	assert.Equal(t, models.WorkflowStatusPaused, retrieved.Status)
	assert.True(t, retrieved.UpdatedAt.After(initialUpdatedAt))
}

func TestNewPersistence_ListWorkflows(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	// Create multiple test workflows
	workflows := []*models.Workflow{
		{
			ID:          "test-workflow-3",
			Name:        "Test Workflow 3",
			Description: "Description 3",
			WorkflowTriggers: []*models.WorkflowTrigger{
				{
					TriggerID: "schedule",
					Configuration: map[string]any{
						"cron": "0 0 * * *",
					},
				},
			},
			Steps: []*models.WorkflowStep{
				{
					UID:      "step1",
					ActionID: "log",
					Name:     "Log Message 3",
					Configuration: map[string]any{
						"message": "Hello 3",
					},
					Enabled: true,
				},
			},
			Status: models.WorkflowStatusActive,
			Owner:  "test-user",
		},
		{
			ID:          "test-workflow-4",
			Name:        "Test Workflow 4",
			Description: "Description 4",
			WorkflowTriggers: []*models.WorkflowTrigger{
				{
					TriggerID: "schedule",
					Configuration: map[string]any{
						"cron": "0 0 * * *",
					},
				},
			},
			Steps: []*models.WorkflowStep{
				{
					UID:      "step1",
					ActionID: "log",
					Name:     "Log Message 4",
					Configuration: map[string]any{
						"message": "Hello 4",
					},
					Enabled: true,
				},
			},
			Status: models.WorkflowStatusInactive,
			Owner:  "test-user",
		},
	}

	// Save workflows
	for _, workflow := range workflows {
		err := p.SaveWorkflow(t.Context(), workflow)
		require.NoError(t, err)
	}

	// Retrieve all workflows
	retrieved, err := p.Workflows(t.Context())
	require.NoError(t, err)

	// Should have at least our test workflows
	var testWorkflows []*models.Workflow

	for _, w := range retrieved {
		if w.ID == "test-workflow-3" || w.ID == "test-workflow-4" {
			testWorkflows = append(testWorkflows, w)
		}
	}

	assert.Len(t, testWorkflows, 2)
}

func TestNewPersistence_DeleteWorkflow(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	workflow := &models.Workflow{
		ID:          "test-workflow-5",
		Name:        "Test Workflow to Delete",
		Description: "A test workflow for deletion",
		WorkflowTriggers: []*models.WorkflowTrigger{
			{
				ID:          uuid.New().String(),
				Name:        "Daily Schedule",
				Description: "Runs daily at midnight",
				TriggerID:   "schedule",
				Configuration: map[string]any{
					"cron": "0 0 * * *",
				},
			},
		},
		Steps: []*models.WorkflowStep{
			{
				UID:      "step1",
				ActionID: "log",
				Name:     "Log Delete Message",
				Configuration: map[string]any{
					"message": "Hello Delete",
				},
				Enabled: true,
			},
		},
		Status: models.WorkflowStatusActive,
		Owner:  "test-user",
	}

	// Save workflow
	err := p.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := p.WorkflowByID(t.Context(), "test-workflow-5")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete workflow
	err = p.DeleteWorkflow(t.Context(), "test-workflow-5")
	require.NoError(t, err)

	// Verify it's gone (soft delete)
	deleted, err := p.WorkflowByID(t.Context(), "test-workflow-5")
	require.NoError(t, err)
	assert.Nil(t, deleted)

	// Delete non-existent workflow (should not error)
	err = p.DeleteWorkflow(t.Context(), "non-existent")
	assert.NoError(t, err)
}

func TestNewPersistence_ComplexWorkflow(t *testing.T) {
	p := setupTestDB(t)

	defer func() {
		err := p.Close(t.Context())
		require.NoError(t, err)
	}()

	workflow := &models.Workflow{
		ID:          "test-complex-workflow",
		Name:        "Complex Test Workflow",
		Description: "A complex workflow with multiple triggers and steps",
		WorkflowTriggers: []*models.WorkflowTrigger{
			{
				TriggerID: "schedule",
				Configuration: map[string]any{
					"cron":        "0 0 * * *",
					"workflow_id": "test-complex-workflow",
					"enabled":     true,
				},
			},
			{
				TriggerID: "webhook",
				Configuration: map[string]any{
					"path":        "/webhook/test",
					"method":      "POST",
					"workflow_id": "test-complex-workflow",
				},
			},
		},
		Steps: []*models.WorkflowStep{
			{
				UID:      "fetch_data",
				ActionID: "http_request",
				Name:     "Fetch Data",
				Configuration: map[string]any{
					"url":    "https://api.example.com/data",
					"method": "GET",
					"headers": map[string]any{
						"Authorization": "Bearer token",
					},
				},
				OnSuccess: stringPtr("transform_data"),
				OnFailure: stringPtr("error_handler"),
				Enabled:   true,
			},
			{
				UID:      "transform_data",
				ActionID: "transform",
				Name:     "Transform Data",
				Configuration: map[string]any{
					"expression": "$.data",
					"input":      "{{steps.fetch_data.body}}",
				},
				OnSuccess: stringPtr("log_result"),
				Enabled:   true,
			},
			{
				UID:      "log_result",
				ActionID: "log",
				Name:     "Log Result",
				Configuration: map[string]any{
					"message": "Processing completed: {{steps.transform_data.result}}",
					"level":   "info",
				},
				Enabled: true,
			},
			{
				UID:      "error_handler",
				ActionID: "log",
				Name:     "Error Handler",
				Configuration: map[string]any{
					"message": "Error occurred: {{steps.fetch_data.error}}",
					"level":   "error",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"api_base_url": "https://api.example.com",
			"timeout":      30,
			"retry_count":  3,
		},
		Status: models.WorkflowStatusActive,
		Metadata: map[string]any{
			"version":     "1.0.0",
			"environment": "test",
			"tags":        []string{"test", "complex", "api"},
		},
		Owner: "test-user",
	}

	// Save complex workflow
	err := p.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := p.WorkflowByID(t.Context(), "test-complex-workflow")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, workflow.Name, retrieved.Name)
	assert.Len(t, retrieved.WorkflowTriggers, len(workflow.WorkflowTriggers))
	assert.Len(t, retrieved.Steps, len(workflow.Steps))

	// Verify first trigger
	trigger1 := retrieved.WorkflowTriggers[0]
	assert.Equal(t, "schedule", trigger1.TriggerID)
	assert.Equal(t, "0 0 * * *", trigger1.Configuration["cron"])
	assert.Equal(t, true, trigger1.Configuration["enabled"])

	// Verify first step
	step1 := retrieved.Steps[0]
	assert.Equal(t, "fetch_data", step1.UID)
	assert.Equal(t, "http_request", step1.ActionID)
	assert.Equal(t, "https://api.example.com/data", step1.Configuration["url"])
	assert.NotNil(t, step1.OnSuccess)
	assert.Equal(t, "transform_data", *step1.OnSuccess)

	// Verify variables and metadata
	assert.Equal(t, "https://api.example.com", retrieved.Variables["api_base_url"])
	assert.Equal(t, float64(30), retrieved.Variables["timeout"]) // JSON unmarshals numbers as float64
	assert.Equal(t, "1.0.0", retrieved.Metadata["version"])
}
