package trigger

import (
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
)

type Repository struct {
	persistence persistence.Persistence
}

func NewRepository(persistence persistence.Persistence) *Repository {
	return &Repository{
		persistence: persistence,
	}
}

func (r *Repository) FetchAll() ([]*models.Trigger, error) {
	triggers, err := r.persistence.Triggers()

	if err != nil {
		return make([]*models.Trigger, 0), err
	}

	return triggers, nil
}
