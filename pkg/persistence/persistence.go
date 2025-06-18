package persistence

import "github.com/dukex/operion/pkg/models"

type Persistence interface {
	Workflows() ([]*models.Workflow, error)
	SaveWorkflow(workflow *models.Workflow) error
	WorkflowByID(id string) (*models.Workflow, error)
	DeleteWorkflow(id string) error
}
