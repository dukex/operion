package file

import (
	"encoding/json"
	"os"

	"github.com/dukex/operion/internal/domain"
)

type FilePersistence struct {
	path string
}

func NewFilePersistence(path string) *FilePersistence {
	return &FilePersistence{
		path: path,
	}
}

func (fp *FilePersistence) AllWorkflows() ([]domain.Workflow, error) {
	body, err := os.ReadFile(fp.path)
	if err != nil {
		return nil, err
	}

	var workflows []domain.Workflow

	err = json.Unmarshal(body, &workflows)
	if err != nil {
		return nil, err
	}

	return workflows, nil
}
