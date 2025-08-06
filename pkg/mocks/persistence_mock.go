package mocks

import (
	"context"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/mock"
)

// MockPersistence is a mock implementation of persistence.Persistence interface.
type MockPersistence struct {
	mock.Mock
}

func (m *MockPersistence) Workflows(ctx context.Context) ([]*models.Workflow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*models.Workflow), args.Error(1)
}

func (m *MockPersistence) SaveWorkflow(ctx context.Context, workflow *models.Workflow) error {
	args := m.Called(ctx, workflow)

	return args.Error(0)
}

func (m *MockPersistence) WorkflowByID(ctx context.Context, id string) (*models.Workflow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*models.Workflow), args.Error(1)
}

func (m *MockPersistence) DeleteWorkflow(ctx context.Context, id string) error {
	args := m.Called(ctx, id)

	return args.Error(0)
}

func (m *MockPersistence) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockPersistence) WorkflowTriggersBySourceID(ctx context.Context, sourceID string, status models.WorkflowStatus) ([]*models.TriggerMatch, error) {
	args := m.Called(ctx, sourceID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*models.TriggerMatch), args.Error(1)
}

func (m *MockPersistence) WorkflowTriggersBySourceAndEvent(ctx context.Context, sourceID, eventType string, status models.WorkflowStatus) ([]*models.TriggerMatch, error) {
	args := m.Called(ctx, sourceID, eventType, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*models.TriggerMatch), args.Error(1)
}

func (m *MockPersistence) Close(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}
