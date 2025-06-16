package domain

type Persistence interface {
	// Save(workflow Workflow) error
	// DeleteWorkflow(id string) error
	AllWorkflows() ([]*Workflow, error)
	WorkflowByID(id string) (*Workflow, error)

	// SaveAction(action Action) error
	// GetAction(id string) (Action, error)
	// DeleteAction(id string) error
}
