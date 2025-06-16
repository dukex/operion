package file_write_action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blues/jsonata-go"
	log "github.com/sirupsen/logrus"

	"github.com/dukex/operion/internal/domain"
)

type FileWriteAction struct {
	ID        string
	FileName  string
	Directory string
	Overwrite bool
	Input     string
}

func NewFileWriteAction(config map[string]interface{}) (*FileWriteAction, error) {
	id, _ := config["id"].(string)
	fileName, _ := config["file_name"].(string)
	directory, _ := config["directory"].(string)
	overwrite, _ := config["overwrite"].(bool)
	input, _ := config["input"].(string)

	if directory == "" {
		directory = "/tmp"
	}

	return &FileWriteAction{
		ID:        id,
		FileName:  fileName,
		Directory: directory,
		Overwrite: overwrite,
		Input:     input,
	}, nil
}

func (a *FileWriteAction) GetID() string   { return a.ID }
func (a *FileWriteAction) GetType() string { return "file_write" }
func (a *FileWriteAction) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":        a.ID,
		"file_name": a.FileName,
		"directory": a.Directory,
		"overwrite": a.Overwrite,
	}
}
func (a *FileWriteAction) Validate() error { return nil }

func (a *FileWriteAction) Execute(ctx context.Context, executionCtx domain.ExecutionContext) (interface{}, error) {
	logger := executionCtx.Logger.WithFields(log.Fields{
		"module": "http_request_action",
	})
	logger.Info("Executing FileWriteAction")

	data, err := a.extract(executionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract input data: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	fullPath := filepath.Join(a.Directory, a.FileName)

	if !a.Overwrite {
		if _, err := os.Stat(fullPath); err == nil {
			return nil, fmt.Errorf("file '%s' already exists and overwrite is false", fullPath)
		}
	}

	if err := os.MkdirAll(a.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory '%s': %w", a.Directory, err)
	}

	if err := os.WriteFile(fullPath, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file '%s': %w", fullPath, err)
	}

	result := map[string]interface{}{
		"file_path":     fullPath,
		"bytes_written": len(jsonData),
		"success":       true,
	}

	logger.Infof("FileWriteAction successfully wrote %d bytes to '%s'", len(jsonData), fullPath)
	return result, nil
}

func (a *FileWriteAction) extract(executionCtx domain.ExecutionContext) (interface{}, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	e := jsonata.MustCompile(a.Input)
	results, err := e.Eval(executionCtx.StepResults)

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input expression '%s': %w", a.Input, err)
	}

	return results, nil
}
