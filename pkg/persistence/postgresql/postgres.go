// Package postgresql provides PostgreSQL persistence implementation for workflows and triggers.
package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/sqlbase"

	_ "github.com/lib/pq"
)

// Persistence implements the persistence layer for PostgreSQL.
type Persistence struct {
	db                    *sql.DB
	logger                *slog.Logger
	workflowRepo          *WorkflowRepository
	nodeRepo              *NodeRepository
	connectionRepo        *ConnectionRepository
	executionContextRepo  *ExecutionContextRepository
	inputCoordinationRepo *InputCoordinationRepository
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
	nodeRepo := NewNodeRepository(database, logger)
	connectionRepo := NewConnectionRepository(database, logger)
	executionContextRepo := NewExecutionContextRepository(database, logger)
	inputCoordinationRepo := NewInputCoordinationRepository(database, logger)

	postgres := &Persistence{
		db:                    database,
		logger:                logger,
		workflowRepo:          workflowRepo,
		nodeRepo:              nodeRepo,
		connectionRepo:        connectionRepo,
		executionContextRepo:  executionContextRepo,
		inputCoordinationRepo: inputCoordinationRepo,
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

// WorkflowTriggersBySourceEventAndProvider finds triggers by source ID, event type, and provider ID.
// Deprecated: Use NodeRepository.FindTriggerNodesBySourceEventAndProvider instead
func (p *Persistence) WorkflowTriggersBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	return p.nodeRepo.FindTriggerNodesBySourceEventAndProvider(ctx, sourceID, eventType, providerID, status)
}

// WorkflowRepository returns the workflow repository implementation.
func (p *Persistence) WorkflowRepository() persistence.WorkflowRepository {
	return p.workflowRepo
}

// Repository accessors - return the properly initialized repository implementations

func (p *Persistence) NodeRepository() persistence.NodeRepository {
	return p.nodeRepo
}

func (p *Persistence) ConnectionRepository() persistence.ConnectionRepository {
	return p.connectionRepo
}

func (p *Persistence) ExecutionContextRepository() persistence.ExecutionContextRepository {
	return p.executionContextRepo
}

func (p *Persistence) InputCoordinationRepository() persistence.InputCoordinationRepository {
	return p.inputCoordinationRepo
}
