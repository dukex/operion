// Package postgresql provides PostgreSQL persistence implementation for workflows and triggers.
package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/sqlbase"

	_ "github.com/lib/pq"
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

// WorkflowTriggersBySourceEventAndProvider finds triggers by source ID, event type, and provider ID.
func (p *Persistence) WorkflowTriggersBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	return p.workflowRepo.FindTriggersBySourceEventAndProvider(ctx, sourceID, eventType, providerID, status)
}

// WorkflowRepository returns the workflow repository implementation.
func (p *Persistence) WorkflowRepository() persistence.WorkflowRepository {
	return p.workflowRepo
}

// Node-based repository implementations (stub implementations for now)
// TODO: These will be properly implemented when we add node-based persistence support

func (p *Persistence) NodeRepository() persistence.NodeRepository {
	return &postgresNodeRepository{}
}

func (p *Persistence) ConnectionRepository() persistence.ConnectionRepository {
	return &postgresConnectionRepository{}
}

func (p *Persistence) ExecutionContextRepository() persistence.ExecutionContextRepository {
	return &postgresExecutionContextRepository{}
}

func (p *Persistence) InputCoordinationRepository() persistence.InputCoordinationRepository {
	return &postgresInputCoordinationRepository{db: p.db}
}

// Stub implementations for node-based repositories (not yet implemented)

type postgresNodeRepository struct{}

func (nr *postgresNodeRepository) GetNodesFromPublishedWorkflow(ctx context.Context, publishedWorkflowID string) ([]*models.WorkflowNode, error) {
	return nil, errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) GetNodeFromPublishedWorkflow(ctx context.Context, publishedWorkflowID, nodeID string) (*models.WorkflowNode, error) {
	return nil, errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) SaveNode(ctx context.Context, workflowID string, node *models.WorkflowNode) error {
	return errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) UpdateNode(ctx context.Context, workflowID string, node *models.WorkflowNode) error {
	return errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) DeleteNode(ctx context.Context, workflowID, nodeID string) error {
	return errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) GetNodesByWorkflow(ctx context.Context, workflowID string) ([]*models.WorkflowNode, error) {
	return nil, errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

func (nr *postgresNodeRepository) FindTriggerNodesBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	return nil, errors.New("node-based operations not yet implemented in PostgreSQL persistence")
}

type postgresConnectionRepository struct{}

func (cr *postgresConnectionRepository) GetConnectionsFromPublishedWorkflow(ctx context.Context, publishedWorkflowID, sourceNodeID string) ([]*models.Connection, error) {
	return nil, errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) GetConnectionsByTargetNode(ctx context.Context, publishedWorkflowID, targetNodeID string) ([]*models.Connection, error) {
	return nil, errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) GetAllConnectionsFromPublishedWorkflow(ctx context.Context, publishedWorkflowID string) ([]*models.Connection, error) {
	return nil, errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) SaveConnection(ctx context.Context, workflowID string, connection *models.Connection) error {
	return errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) UpdateConnection(ctx context.Context, workflowID string, connection *models.Connection) error {
	return errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) DeleteConnection(ctx context.Context, workflowID, connectionID string) error {
	return errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

func (cr *postgresConnectionRepository) GetConnectionsByWorkflow(ctx context.Context, workflowID string) ([]*models.Connection, error) {
	return nil, errors.New("connection operations not yet implemented in PostgreSQL persistence")
}

type postgresExecutionContextRepository struct{}

func (ecr *postgresExecutionContextRepository) SaveExecutionContext(ctx context.Context, execCtx *models.ExecutionContext) error {
	return errors.New("execution context operations not yet implemented in PostgreSQL persistence")
}

func (ecr *postgresExecutionContextRepository) GetExecutionContext(ctx context.Context, executionID string) (*models.ExecutionContext, error) {
	return nil, errors.New("execution context operations not yet implemented in PostgreSQL persistence")
}

func (ecr *postgresExecutionContextRepository) UpdateExecutionContext(ctx context.Context, execCtx *models.ExecutionContext) error {
	return errors.New("execution context operations not yet implemented in PostgreSQL persistence")
}

func (ecr *postgresExecutionContextRepository) GetExecutionsByWorkflow(ctx context.Context, publishedWorkflowID string) ([]*models.ExecutionContext, error) {
	return nil, errors.New("execution context operations not yet implemented in PostgreSQL persistence")
}

func (ecr *postgresExecutionContextRepository) GetExecutionsByStatus(ctx context.Context, status models.ExecutionStatus) ([]*models.ExecutionContext, error) {
	return nil, errors.New("execution context operations not yet implemented in PostgreSQL persistence")
}

// PostgreSQL Input Coordination Repository (stub implementation).
type postgresInputCoordinationRepository struct {
	db *sql.DB
}

func (icr *postgresInputCoordinationRepository) SaveInputState(ctx context.Context, state *models.NodeInputState) error {
	return errors.New("input coordination operations not yet implemented in PostgreSQL persistence")
}

func (icr *postgresInputCoordinationRepository) LoadInputState(ctx context.Context, nodeExecutionID string) (*models.NodeInputState, error) {
	return nil, errors.New("input coordination operations not yet implemented in PostgreSQL persistence")
}

func (icr *postgresInputCoordinationRepository) FindPendingNodeExecution(ctx context.Context, nodeID, executionID string) (*models.NodeInputState, error) {
	return nil, errors.New("input coordination operations not yet implemented in PostgreSQL persistence")
}

func (icr *postgresInputCoordinationRepository) DeleteInputState(ctx context.Context, nodeExecutionID string) error {
	return errors.New("input coordination operations not yet implemented in PostgreSQL persistence")
}

func (icr *postgresInputCoordinationRepository) CleanupExpiredStates(ctx context.Context, maxAge time.Duration) error {
	return errors.New("input coordination operations not yet implemented in PostgreSQL persistence")
}
