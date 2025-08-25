package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dukex/operion/pkg/models"
)

// ExecutionContextRepository handles execution context-related file operations.
type ExecutionContextRepository struct {
	root string // File system root for storing execution contexts
}

// NewExecutionContextRepository creates a new execution context repository.
func NewExecutionContextRepository(root string) *ExecutionContextRepository {
	return &ExecutionContextRepository{root: root}
}

// validateExecutionID validates that the execution ID is safe for file operations.
func (ecr *ExecutionContextRepository) validateExecutionID(executionID string) error {
	if executionID == "" {
		return errors.New("execution ID cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(executionID, "..") || strings.Contains(executionID, "/") || strings.Contains(executionID, "\\") {
		return errors.New("execution ID contains invalid characters")
	}

	return nil
}

// SaveExecutionContext saves an execution context to the file system.
func (ecr *ExecutionContextRepository) SaveExecutionContext(ctx context.Context, execCtx *models.ExecutionContext) error {
	// Prepare execution context with defaults
	contextToSave := *execCtx
	if contextToSave.NodeResults == nil {
		contextToSave.NodeResults = make(map[string]models.NodeResult)
	}

	if contextToSave.Variables == nil {
		contextToSave.Variables = make(map[string]any)
	}

	if contextToSave.TriggerData == nil {
		contextToSave.TriggerData = make(map[string]any)
	}

	if contextToSave.Metadata == nil {
		contextToSave.Metadata = make(map[string]any)
	}

	// Create execution contexts directory if it doesn't exist
	execContextsDir := filepath.Join(ecr.root, "execution_contexts")

	err := os.MkdirAll(execContextsDir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create execution contexts directory: %w", err)
	}

	// Write execution context to file
	filePath := filepath.Join(execContextsDir, execCtx.ID+".json")

	data, err := json.Marshal(contextToSave)
	if err != nil {
		return fmt.Errorf("failed to marshal execution context %s: %w", execCtx.ID, err)
	}

	err = os.WriteFile(filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write execution context %s: %w", execCtx.ID, err)
	}

	return nil
}

// GetExecutionContext retrieves an execution context by its ID from the file system.
func (ecr *ExecutionContextRepository) GetExecutionContext(ctx context.Context, executionID string) (*models.ExecutionContext, error) {
	// Validate execution ID to prevent path traversal
	if err := ecr.validateExecutionID(executionID); err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	filePath := filepath.Join(ecr.root, "execution_contexts", executionID+".json")

	data, err := os.ReadFile(filePath) // #nosec G304 -- filePath is validated and constructed safely
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("execution context not found: %s", executionID)
		}

		return nil, fmt.Errorf("failed to read execution context %s: %w", executionID, err)
	}

	var execCtx models.ExecutionContext

	err = json.Unmarshal(data, &execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution context %s: %w", executionID, err)
	}

	return &execCtx, nil
}

// UpdateExecutionContext updates an existing execution context in the file system.
func (ecr *ExecutionContextRepository) UpdateExecutionContext(ctx context.Context, execCtx *models.ExecutionContext) error {
	// Check if execution context exists first
	_, err := ecr.GetExecutionContext(ctx, execCtx.ID)
	if err != nil {
		return err // Will return "execution context not found" if it doesn't exist
	}

	// Save the updated execution context (overwrites the existing file)
	return ecr.SaveExecutionContext(ctx, execCtx)
}

// GetExecutionsByWorkflow retrieves all execution contexts for a specific workflow.
func (ecr *ExecutionContextRepository) GetExecutionsByWorkflow(ctx context.Context, publishedWorkflowID string) ([]*models.ExecutionContext, error) {
	execContextsDir := filepath.Join(ecr.root, "execution_contexts")

	// Check if directory exists
	if _, err := os.Stat(execContextsDir); os.IsNotExist(err) {
		return []*models.ExecutionContext{}, nil
	}

	entries, err := os.ReadDir(execContextsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution contexts directory: %w", err)
	}

	var executions []*models.ExecutionContext

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			executionID := strings.TrimSuffix(entry.Name(), ".json")

			execCtx, err := ecr.GetExecutionContext(ctx, executionID)
			if err != nil {
				// Skip invalid files
				continue
			}

			if execCtx.PublishedWorkflowID == publishedWorkflowID {
				executions = append(executions, execCtx)
			}
		}
	}

	return executions, nil
}

// GetExecutionsByStatus retrieves all execution contexts with a specific status.
func (ecr *ExecutionContextRepository) GetExecutionsByStatus(ctx context.Context, status models.ExecutionStatus) ([]*models.ExecutionContext, error) {
	execContextsDir := filepath.Join(ecr.root, "execution_contexts")

	// Check if directory exists
	if _, err := os.Stat(execContextsDir); os.IsNotExist(err) {
		return []*models.ExecutionContext{}, nil
	}

	entries, err := os.ReadDir(execContextsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution contexts directory: %w", err)
	}

	var executions []*models.ExecutionContext

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			executionID := strings.TrimSuffix(entry.Name(), ".json")

			execCtx, err := ecr.GetExecutionContext(ctx, executionID)
			if err != nil {
				// Skip invalid files
				continue
			}

			if execCtx.Status == status {
				executions = append(executions, execCtx)
			}
		}
	}

	return executions, nil
}
