// Package trigger provides repository management for workflow triggers.
package trigger

import (
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
