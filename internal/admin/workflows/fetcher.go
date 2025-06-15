package workflows

import "github.com/dukex/operion/internal/domain"

type Fetcher struct {
	repository *Repository
}

func NewFetcher(repository *Repository) *Fetcher {
	return &Fetcher{
		repository: repository,
	}
}

func (f *Fetcher) FetchAll() ([]domain.Workflow, error) {
	return f.repository.FetchAll()
}
