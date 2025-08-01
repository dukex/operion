// Package postgresql provides PostgreSQL persistence implementation for workflows and triggers.
package postgresql

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/sqlbase"
)

type PostgreSQLPersistence struct {
	db               *sql.DB
	logger           *slog.Logger
	workflowRepo     *WorkflowRepository
}

// NewPostgreSQLPersistence creates a new PostgreSQL persistence layer
func NewPostgreSQLPersistence(logger *slog.Logger, databaseURL string) (persistence.Persistence, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize components
	migrationManager := sqlbase.NewMigrationManager(logger, db, Migrations())
	workflowRepo := NewWorkflowRepository(db)

	p := &PostgreSQLPersistence{
		db:               db,
		logger:           logger,
		workflowRepo:     workflowRepo,
	}

	// Run migrations on initialization
	if err := migrationManager.RunMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return p, nil
}

// Close closes the database connection
func (p *PostgreSQLPersistence) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// HealthCheck verifies the database connection is healthy
func (p *PostgreSQLPersistence) HealthCheck() error {
	return p.db.Ping()
}

// Workflows returns all workflows from the database
func (p *PostgreSQLPersistence) Workflows() ([]*models.Workflow, error) {
	return p.workflowRepo.GetAll()
}

// WorkflowByID returns a workflow by its ID
func (p *PostgreSQLPersistence) WorkflowByID(id string) (*models.Workflow, error) {
	return p.workflowRepo.GetByID(id)
}

// SaveWorkflow saves a workflow to the database
func (p *PostgreSQLPersistence) SaveWorkflow(workflow *models.Workflow) error {
	return p.workflowRepo.Save(workflow)
}

// DeleteWorkflow soft deletes a workflow by setting deleted_at timestamp
func (p *PostgreSQLPersistence) DeleteWorkflow(id string) error {
	return p.workflowRepo.Delete(id)
}

