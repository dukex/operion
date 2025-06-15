package domain

type Persistence interface {
	// Save(workflow Workflow) error
	// GetWorkflow(id string) (Workflow, error)
	// DeleteWorkflow(id string) error
	AllWorkflows() ([]Workflow, error)

	// SaveAction(action Action) error
	// GetAction(id string) (Action, error)
	// DeleteAction(id string) error
}
