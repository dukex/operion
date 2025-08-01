// Package postgresql provides PostgreSQL persistence implementation for workflows and triggers.
package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/sqlbase"
)

// Persistence implements the persistence layer for PostgreSQL.
type Persistence struct {
	db           *sql.DB
	logger       *slog.Logger
	workflowRepo *WorkflowRepository
}

// NewPersistence creates a new PostgreSQL persistence layer.
func NewPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) (*Persistence, error) {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	err = database.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize components
	migrationManager := sqlbase.NewMigrationManager(logger, database, migrations())
	workflowRepo := NewWorkflowRepository(database, logger)

	postgres := &Persistence{
		db:           database,
		logger:       logger,
		workflowRepo: workflowRepo,
	}

	// Run migrations on initialization
	err = migrationManager.RunMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return postgres, nil
}

// Close closes the database connection.
func (p *Persistence) Close(ctx context.Context) error {
	if p.db != nil {
		err := p.db.Close()
		if err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	return nil
}

// HealthCheck verifies the database connection is healthy.
func (p *Persistence) HealthCheck(ctx context.Context) error {
	err := p.db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Workflows returns all workflows from the database.
func (p *Persistence) Workflows(ctx context.Context) ([]*models.Workflow, error) {
	return p.workflowRepo.GetAll(ctx)
}

// WorkflowByID returns a workflow by its ID.
func (p *Persistence) WorkflowByID(ctx context.Context, id string) (*models.Workflow, error) {
	return p.workflowRepo.GetByID(ctx, id)
}

// SaveWorkflow saves a workflow to the database.
func (p *Persistence) SaveWorkflow(ctx context.Context, workflow *models.Workflow) error {
	return p.workflowRepo.Save(ctx, workflow)
}

// DeleteWorkflow soft deletes a workflow by setting deleted_at timestamp.
func (p *Persistence) DeleteWorkflow(ctx context.Context, id string) error {
	return p.workflowRepo.Delete(ctx, id)
}
