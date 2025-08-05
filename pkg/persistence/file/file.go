// Package file provides file-based persistence implementation for workflows and triggers.
package file

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

// Close performs any necessary cleanup. For file-based persistence, there is nothing to clean up.
func (fp *FilePersistence) Close(_ context.Context) error {
	return nil
}

// HealthCheck checks if the file persistence layer is healthy by verifying the root directory exists.
func (fp *FilePersistence) HealthCheck(_ context.Context) error {
	if _, err := os.Stat(fp.root); os.IsNotExist(err) {
		return os.ErrNotExist
	}

	return nil
}

// Workflows retrieves all workflows from the file system.
func (fp *FilePersistence) Workflows(ctx context.Context) ([]*models.Workflow, error) {
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
		workflow, err := fp.WorkflowByID(ctx, file[:len(file)-5])
		if err != nil {
			return nil, err
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// WorkflowByID retrieves a workflow by its ID from the file system.
func (fp *FilePersistence) WorkflowByID(_ context.Context, workflowID string) (*models.Workflow, error) {
	filePath := filepath.Clean(path.Join(fp.root, "workflows", workflowID+".json"))

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

// SaveWorkflow saves a workflow to the file system.
func (fp *FilePersistence) SaveWorkflow(_ context.Context, workflow *models.Workflow) error {
	err := os.MkdirAll(fp.root+"/workflows", 0750)
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

	return os.WriteFile(filePath, data, 0600)
}

// DeleteWorkflow removes a workflow by its ID.
func (fp *FilePersistence) DeleteWorkflow(_ context.Context, id string) error {
	filePath := path.Join(fp.root+"/workflows", id+".json")

	err := os.Remove(filePath)

	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}
