package workflows

type Creator struct {
}

func NewCreator(repository *Repository) *Creator {
	return &Creator{}
}
