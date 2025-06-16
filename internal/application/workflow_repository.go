package application

import (
	"errors"

	"github.com/dukex/operion/internal/domain"
)

type WorkflowRepository struct {
	persistence domain.Persistence
}

func NewWorkflowRepository(persistence domain.Persistence) *WorkflowRepository {
	return &WorkflowRepository{
		persistence: persistence,
	}
}

func (r *WorkflowRepository) FetchAll() ([]*domain.Workflow, error) {
	workflows, err := r.persistence.AllWorkflows()

	if err != nil {
		return make([]*domain.Workflow, 0), err
	}

	return workflows, nil
}

func (r *WorkflowRepository) FetchByID(id string) (*domain.Workflow, error) {
	workflow, err := r.persistence.WorkflowByID(id)

	if err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, errors.New("workflow not found")
	}

	return workflow, nil
}
