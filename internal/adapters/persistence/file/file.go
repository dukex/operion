package file

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"

	"github.com/dukex/operion/internal/domain"
)

type FilePersistence struct {
	root string
}

func NewFilePersistence(root string) *FilePersistence {
	return &FilePersistence{
		root: root,
	}
}

func (fp *FilePersistence) AllWorkflows() ([]*domain.Workflow, error) {
	root := os.DirFS(fp.root + "/workflows")
	jsonFiles, err := fs.Glob(root, "*.json")

	if err != nil {
		return nil, err
	}

	if len(jsonFiles) == 0 {
		return make([]*domain.Workflow, 0), nil
	}

	workflows := make([]*domain.Workflow, 0, len(jsonFiles))

	for _, file := range jsonFiles {
		workflow, err := fp.WorkflowByID(file[:len(file)-5])
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

func (fp *FilePersistence) WorkflowByID(workflowID string) (*domain.Workflow, error) {
	filePath := path.Join(fp.root+"/workflows", workflowID+".json")
	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var workflow domain.Workflow
	err = json.Unmarshal(body, &workflow)
	if err != nil {
		return nil, err
	}

	return &workflow, nil
}
