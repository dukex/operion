package workflow

import (
	"errors"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/google/uuid"
)

type Repository struct {
	persistence persistence.Persistence
}

func NewRepository(persistence persistence.Persistence) *Repository {
	return &Repository{
		persistence: persistence,
	}
}

func (r *Repository) FetchAll() ([]*models.Workflow, error) {
	workflows, err := r.persistence.Workflows()

	if err != nil {
		return make([]*models.Workflow, 0), err
	}

	return workflows, nil
}

func (r *Repository) FetchByID(id string) (*models.Workflow, error) {
	workflow, err := r.persistence.WorkflowByID(id)

	if err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, errors.New("workflow not found")
	}

	return workflow, nil
}

func (r *Repository) Create(workflow *models.Workflow) (*models.Workflow, error) {
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	if workflow.Status == "" {
		workflow.Status = models.WorkflowStatusInactive
	}

	err := r.persistence.SaveWorkflow(workflow)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

func (r *Repository) Update(id string, workflow *models.Workflow) (*models.Workflow, error) {
	existing, err := r.persistence.WorkflowByID(id)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, errors.New("workflow not found")
	}

	workflow.ID = id
	workflow.CreatedAt = existing.CreatedAt
	workflow.UpdatedAt = time.Now()

	err = r.persistence.SaveWorkflow(workflow)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

func (r *Repository) Delete(id string) error {
	existing, err := r.persistence.WorkflowByID(id)
	if err != nil {
		return err
	}

	if existing == nil {
		return errors.New("workflow not found")
	}

	return r.persistence.DeleteWorkflow(id)
}
