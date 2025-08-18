package postgresql_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/postgresql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var postgresContainer *postgres.PostgresContainer

// isDockerAvailable checks if Docker daemon is running.
func isDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "docker", "info")

	return cmd.Run() == nil
}

func dropDb(ctx context.Context, t *testing.T, databaseURL string) {
	t.Helper()

	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)

	for _, table := range []string{"workflow_steps", "workflow_triggers", "workflows", "schema_migrations"} {
		_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table)
		require.NoError(t, err)
	}

	err = db.Close()
	require.NoError(t, err)
}

func setupTestDB(t *testing.T) (*postgresql.Persistence, context.Context, string) {
	t.Helper()

	// Skip tests if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping PostgreSQL tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	if postgresContainer == nil || !postgresContainer.IsRunning() {
		var err error

		postgresContainer, err = postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("operion_test"),
			postgres.WithUsername("operion"),
			postgres.WithPassword("operion"),
			postgres.BasicWaitStrategies(),
		)
		require.NoError(t, err)
	}

	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dropDb(ctx, t, databaseURL)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	persistence, err := postgresql.NewPersistence(ctx, logger, databaseURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		dropDb(ctx, t, databaseURL)

		err = persistence.Close(ctx)
		require.NoError(t, err)

		cancel()
	})

	return persistence, ctx, databaseURL
}

// Helper function to create string pointers.
func stringPtr(s string) *string {
	return &s
}

func TestNewPersistence_Migrations(t *testing.T) {
	_, ctx, databaseURL := setupTestDB(t)

	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)

	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	// Verify tables were created
	var exists bool

	err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT FROM
information_schema.tables WHERE table_name = 'workflows')`).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "workflows table should exist")

	err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT FROM
information_schema.tables WHERE table_name = 'schema_migrations')`).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "schema_migrations table should exist")

	var version int

	err = db.QueryRowContext(ctx, "SELECT version FROM schema_migrations WHERE version = 1").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestNewPersistence_HealthCheck(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	err := p.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestNewPersistence_SaveAndRetrieveWorkflow(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	workflow := &models.Workflow{
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

	err := p.SaveWorkflow(ctx, workflow)
	require.NoError(t, err)
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())

	// Test retrieving workflow by ID
	retrieved, err := p.WorkflowByID(ctx, workflow.ID)
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
	notFound, err := p.WorkflowByID(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestNewPersistence_UpdateWorkflow(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	workflow := &models.Workflow{
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

	err := p.SaveWorkflow(ctx, workflow)
	require.NoError(t, err)

	initialUpdatedAt := workflow.UpdatedAt

	// Wait a moment to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Update workflow
	workflow.Name = "Updated Test Workflow"
	workflow.Description = "An updated test workflow"
	workflow.Status = models.WorkflowStatusPaused

	err = p.SaveWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify update
	retrieved, err := p.WorkflowByID(ctx, workflow.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, "Updated Test Workflow", retrieved.Name)
	assert.Equal(t, "An updated test workflow", retrieved.Description)
	assert.Equal(t, models.WorkflowStatusPaused, retrieved.Status)
	assert.True(t, retrieved.UpdatedAt.After(initialUpdatedAt))
}

func TestNewPersistence_ListWorkflows(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	workflows := []*models.Workflow{
		{
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

	for _, workflow := range workflows {
		err := p.SaveWorkflow(ctx, workflow)
		require.NoError(t, err)
	}

	retrieved, err := p.Workflows(ctx)
	require.NoError(t, err)

	assert.Len(t, retrieved, len(workflows))
}

func TestNewPersistence_DeleteWorkflow(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	workflow := &models.Workflow{
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
	err := p.SaveWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := p.WorkflowByID(ctx, workflow.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete workflow
	err = p.DeleteWorkflow(ctx, workflow.ID)
	require.NoError(t, err)

	// Verify it's gone (soft delete)
	deleted, err := p.WorkflowByID(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)

	// Delete non-existent workflow (should not error)
	err = p.DeleteWorkflow(ctx, uuid.NewString())
	assert.NoError(t, err)
}

func TestNewPersistence_ComplexWorkflow(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	workflow := &models.Workflow{
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

	err := p.SaveWorkflow(ctx, workflow)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := p.WorkflowByID(ctx, workflow.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, workflow.Name, retrieved.Name)
	assert.Len(t, retrieved.WorkflowTriggers, len(workflow.WorkflowTriggers))
	assert.Len(t, retrieved.Steps, len(workflow.Steps))

	// Verify first trigger
	assert.Len(t, retrieved.WorkflowTriggers, 2)

	for _, trigger := range retrieved.WorkflowTriggers {
		switch trigger.TriggerID {
		case "schedule":
			assert.Equal(t, "0 0 * * *", trigger.Configuration["cron"])
			assert.Equal(t, true, trigger.Configuration["enabled"])
		case "webhook":
			assert.Equal(t, "/webhook/test", trigger.Configuration["path"])
			assert.Equal(t, "POST", trigger.Configuration["method"])
		}
	}

	assert.Len(t, retrieved.Steps, len(workflow.Steps))

	for _, step := range retrieved.Steps {
		switch step.UID {
		case "fetch_data":
			assert.Equal(t, "http_request", step.ActionID)
			assert.Equal(t, "https://api.example.com/data", step.Configuration["url"])
			assert.Equal(t, "GET", step.Configuration["method"])
			assert.NotNil(t, step.OnSuccess)
			assert.Equal(t, "transform_data", *step.OnSuccess)
		case "transform_data":
			assert.Equal(t, "transform", step.ActionID)
			assert.Equal(t, "$.data", step.Configuration["expression"])
			assert.Equal(t, "{{steps.fetch_data.body}}", step.Configuration["input"])
			assert.NotNil(t, step.OnSuccess)
			assert.Equal(t, "log_result", *step.OnSuccess)
		case "log_result":
			assert.Equal(t, "log", step.ActionID)
			assert.Equal(t, "Processing completed: {{steps.transform_data.result}}", step.Configuration["message"])
		case "error_handler":
			assert.Equal(t, "log", step.ActionID)
			assert.Equal(t, "Error occurred: {{steps.fetch_data.error}}", step.Configuration["message"])
		}
	}

	// Verify variables and metadata
	assert.Equal(t, "https://api.example.com", retrieved.Variables["api_base_url"])
	assert.Equal(t, float64(30), retrieved.Variables["timeout"]) // JSON unmarshals numbers as float64
	assert.Equal(t, "1.0.0", retrieved.Metadata["version"])
}
