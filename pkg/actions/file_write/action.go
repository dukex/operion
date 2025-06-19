package file_write_action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jsonata "github.com/blues/jsonata-go"
	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
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

// GetSchema returns the JSON Schema for File Write Action configuration
func GetFileWriteActionSchema() *models.RegisteredComponent {
	return &models.RegisteredComponent{
		Type:        "file_write",
		Name:        "Write File",
		Description: "Write data to a file",
		Schema: &models.JSONSchema{
			Type:        "object",
			Title:       "File Write Action Configuration",
			Description: "Configuration for writing data to files",
			Properties: map[string]*models.Property{
				"file_name": {
					Type:        "string",
					Description: "Name of the file to write",
				},
				"directory": {
					Type:        "string",
					Description: "Directory path where to write the file",
					Default:     "/tmp",
				},
				"overwrite": {
					Type:        "boolean",
					Description: "Whether to overwrite existing files",
					Default:     false,
				},
				"input": {
					Type:        "string",
					Description: "JSONata expression to extract input data (optional)",
				},
			},
			Required: []string{"file_name"},
		},
	}
}

func (a *FileWriteAction) Execute(ctx context.Context, executionCtx models.ExecutionContext) (interface{}, error) {
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

func (a *FileWriteAction) extract(executionCtx models.ExecutionContext) (interface{}, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	e, err := jsonata.Compile(a.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile input expression '%s': %w", a.Input, err)
	}

	results, err := e.Eval(executionCtx.StepResults)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input expression '%s': %w", a.Input, err)
	}

	return results, nil
}
