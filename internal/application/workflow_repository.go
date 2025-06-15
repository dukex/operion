package application

import "github.com/dukex/operion/internal/domain"

type WorkflowRepository struct {
	persistence domain.Persistence
}

func NewWorkflowRepository(persistence domain.Persistence) *WorkflowRepository {
	return &WorkflowRepository{
		persistence: persistence,
	}
}

func (r *WorkflowRepository) FetchAll() ([]domain.Workflow, error) {
	workflows, err := r.persistence.AllWorkflows()

	if err != nil {
		return make([]domain.Workflow, 0), err
	}

	return workflows, nil
}
