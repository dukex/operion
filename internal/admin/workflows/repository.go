package workflows

import "github.com/dukex/operion/internal/domain"

type Repository struct {
	persistence domain.Persistence
}

func NewRepository(persistence domain.Persistence) *Repository {
	return &Repository{
		persistence: persistence,
	}
}

func (r *Repository) FetchAll() ([]domain.Workflow, error) {
	workflows, err := r.persistence.AllWorkflows()

	if err != nil {
		return make([]domain.Workflow, 0), err
	}

	return workflows, nil
}
