package file_write_action

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dukex/operion/internal/domain"
)

// FileWriteAction writes data to a file
type FileWriteAction struct {
	ID        string
	FileName  string
	Directory string
	Overwrite bool
}

// NewFileWriteAction creates a new file write action
func NewFileWriteAction(config map[string]interface{}) (*FileWriteAction, error) {
	id, _ := config["id"].(string)
	fileName, _ := config["file_name"].(string)
	directory, _ := config["directory"].(string)
	overwrite, _ := config["overwrite"].(bool)

	// Default values
	if id == "" {
		id = "file_write_action"
	}
	if directory == "" {
		directory = "/tmp"
	}

	return &FileWriteAction{
		ID:        id,
		FileName:  fileName,
		Directory: directory,
		Overwrite: overwrite,
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

func (a *FileWriteAction) Execute(ctx context.Context, input domain.ExecutionContext) (domain.ExecutionContext, error) {
	log.Printf("Executing FileWriteAction '%s' to file '%s'", a.ID, a.FileName)

	// Get the data to write from the previous step
	var dataToWrite interface{}
	
	// Look for the most recent step result that has actual data
	// We need to find the step that was executed just before this one
	if len(input.StepResults) > 0 {
		// Get all step result keys and try to find the most relevant one
		// In this case, we want the transform result (get_price) not the HTTP result
		stepKeys := make([]string, 0, len(input.StepResults))
		for stepID := range input.StepResults {
			stepKeys = append(stepKeys, stepID)
		}
		
		// Use the most recent step result (prefer processed data over raw HTTP data)
		// Look for transform/processed results first, prioritize by order of execution
		var candidateSteps []string
		var transformSteps []string
		var otherSteps []string
		
		for _, stepID := range stepKeys {
			stepResult := input.StepResults[stepID]
			if stepResult != nil {
				// Check if this looks like processed data rather than raw HTTP response
				if resultMap, ok := stepResult.(map[string]interface{}); ok {
					if _, hasStatusCode := resultMap["status_code"]; !hasStatusCode {
						// Prioritize transform results
						if _, hasTransform := resultMap["transform"]; hasTransform {
							transformSteps = append(transformSteps, stepID)
						} else {
							otherSteps = append(otherSteps, stepID)
						}
					}
				}
			}
		}
		
		// Prefer transform results first, then other processed results
		candidateSteps = append(candidateSteps, transformSteps...)
		candidateSteps = append(candidateSteps, otherSteps...)
		
		if len(candidateSteps) > 0 {
			stepID := candidateSteps[len(candidateSteps)-1] // Use the last (most recent) step
			dataToWrite = input.StepResults[stepID]
			log.Printf("Using processed data from step '%s' for file write", stepID)
		}
		
		// If no transform data found, use any available data
		if dataToWrite == nil {
			for stepID, stepResult := range input.StepResults {
				if stepResult != nil {
					dataToWrite = stepResult
					log.Printf("Using fallback data from step '%s' for file write", stepID)
					break
				}
			}
		}
	}

	if dataToWrite == nil {
		return input, fmt.Errorf("no data to write to file")
	}

	// Convert data to JSON string
	jsonData, err := json.MarshalIndent(dataToWrite, "", "  ")
	if err != nil {
		return input, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Create full file path
	fullPath := filepath.Join(a.Directory, a.FileName)

	// Check if file exists and overwrite flag
	if !a.Overwrite {
		if _, err := os.Stat(fullPath); err == nil {
			return input, fmt.Errorf("file '%s' already exists and overwrite is false", fullPath)
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(a.Directory, 0755); err != nil {
		return input, fmt.Errorf("failed to create directory '%s': %w", a.Directory, err)
	}

	// Write data to file
	if err := os.WriteFile(fullPath, jsonData, 0644); err != nil {
		return input, fmt.Errorf("failed to write file '%s': %w", fullPath, err)
	}

	// Add results to the ExecutionContext
	if input.StepResults == nil {
		input.StepResults = make(map[string]interface{})
	}
	input.StepResults[a.ID] = map[string]interface{}{
		"file_path":   fullPath,
		"bytes_written": len(jsonData),
		"success":     true,
	}

	log.Printf("FileWriteAction '%s' successfully wrote %d bytes to '%s'", a.ID, len(jsonData), fullPath)
	return input, nil
}