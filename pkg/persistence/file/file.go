// Package file provides file-based persistence implementation for workflows and triggers.
package file

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
)

type FilePersistence struct {
	root string
}

func NewFilePersistence(root string) persistence.Persistence {
	return &FilePersistence{
		root: strings.Replace(root, "file://", "", 1),
	}
}

func (fp *FilePersistence) Close() error {
	return nil
}

func (fp *FilePersistence) HealthCheck() error {
	if _, err := os.Stat(fp.root); os.IsNotExist(err) {
		return os.ErrNotExist
	}

	return nil
}

func (fp *FilePersistence) Workflows() ([]*models.Workflow, error) {
	root := os.DirFS(fp.root + "/workflows")

	jsonFiles, err := fs.Glob(root, "*.json")
	if err != nil {
		return nil, err
	}

	if len(jsonFiles) == 0 {
		return make([]*models.Workflow, 0), nil
	}

	workflows := make([]*models.Workflow, 0, len(jsonFiles))

	for _, file := range jsonFiles {
		workflow, err := fp.WorkflowByID(file[:len(file)-5])
		if err != nil {
			return nil, err
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

func (fp *FilePersistence) WorkflowByID(workflowID string) (*models.Workflow, error) {
	filePath := path.Join(fp.root+"/workflows", workflowID+".json")

	body, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var workflow models.Workflow

	err = json.Unmarshal(body, &workflow)
	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

func (fp *FilePersistence) SaveWorkflow(workflow *models.Workflow) error {
	err := os.MkdirAll(fp.root+"/workflows", 0755)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}

	workflow.UpdatedAt = now

	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return err
	}

	filePath := path.Join(fp.root+"/workflows", workflow.ID+".json")

	return os.WriteFile(filePath, data, 0644)
}

func (fp *FilePersistence) DeleteWorkflow(id string) error {
	filePath := path.Join(fp.root+"/workflows", id+".json")

	err := os.Remove(filePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}
