// Package file provides file-based input coordination persistence.
package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
)

// FileInputCoordinationRepository implements input coordination persistence using JSON files.
type FileInputCoordinationRepository struct {
	baseDir string
}

// NewFileInputCoordinationRepository creates a new file-based input coordination repository.
func NewFileInputCoordinationRepository(baseDir string) *FileInputCoordinationRepository {
	return &FileInputCoordinationRepository{
		baseDir: baseDir,
	}
}

// validateNodeExecutionID validates that the node execution ID is safe for file operations.
func (r *FileInputCoordinationRepository) validateNodeExecutionID(nodeExecutionID string) error {
	if nodeExecutionID == "" {
		return errors.New("node execution ID cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(nodeExecutionID, "..") || strings.Contains(nodeExecutionID, "/") || strings.Contains(nodeExecutionID, "\\") {
		return errors.New("node execution ID contains invalid characters")
	}

	return nil
}

// SaveInputState persists the input state to a JSON file.
func (r *FileInputCoordinationRepository) SaveInputState(ctx context.Context, state *models.NodeInputState) error {
	// Validate node execution ID to prevent path traversal
	if err := r.validateNodeExecutionID(state.NodeExecutionID); err != nil {
		return fmt.Errorf("invalid node execution ID: %w", err)
	}

	filename := filepath.Join(r.baseDir, state.NodeExecutionID+"-input-state.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal input state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write input state file: %w", err)
	}

	return nil
}

// LoadInputState loads input state from a JSON file.
func (r *FileInputCoordinationRepository) LoadInputState(ctx context.Context, nodeExecutionID string) (*models.NodeInputState, error) {
	// Validate node execution ID to prevent path traversal
	if err := r.validateNodeExecutionID(nodeExecutionID); err != nil {
		return nil, fmt.Errorf("invalid node execution ID: %w", err)
	}

	filename := filepath.Join(r.baseDir, nodeExecutionID+"-input-state.json")

	data, err := os.ReadFile(filename) // #nosec G304 -- filename is validated and constructed safely
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("input state not found for node execution %s", nodeExecutionID)
		}

		return nil, fmt.Errorf("failed to read input state file: %w", err)
	}

	var state models.NodeInputState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input state: %w", err)
	}

	return &state, nil
}

// FindPendingNodeExecution finds the first pending execution for a node (for FIFO loop handling).
func (r *FileInputCoordinationRepository) FindPendingNodeExecution(ctx context.Context, nodeID, executionID string) (*models.NodeInputState, error) {
	// For file-based implementation, we need to scan all input state files
	// This is not very efficient but works for the file-based persistence layer
	pattern := filepath.Join(r.baseDir, "*-input-state.json")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob input state files: %w", err)
	}

	var (
		oldestPending *models.NodeInputState
		oldestTime    time.Time
	)

	for _, file := range files {
		// Ensure file is within base directory (additional safety check)
		if !strings.HasPrefix(file, r.baseDir) {
			continue
		}

		data, err := os.ReadFile(file) // #nosec G304 -- file is from filepath.Glob within baseDir
		if err != nil {
			continue // Skip files we can't read
		}

		var state models.NodeInputState
		if err := json.Unmarshal(data, &state); err != nil {
			continue // Skip files we can't parse
		}

		// Check if this matches our node and execution
		if state.NodeID == nodeID && state.ExecutionID == executionID {
			// This is a pending execution for our node - check if it's the oldest
			if oldestPending == nil || state.CreatedAt.Before(oldestTime) {
				oldestPending = &state
				oldestTime = state.CreatedAt
			}
		}
	}

	return oldestPending, nil
}

// DeleteInputState removes input state file.
func (r *FileInputCoordinationRepository) DeleteInputState(ctx context.Context, nodeExecutionID string) error {
	// Validate node execution ID to prevent path traversal
	if err := r.validateNodeExecutionID(nodeExecutionID); err != nil {
		return fmt.Errorf("invalid node execution ID: %w", err)
	}

	filename := filepath.Join(r.baseDir, nodeExecutionID+"-input-state.json")

	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, that's fine
		}

		return fmt.Errorf("failed to delete input state file: %w", err)
	}

	return nil
}

// CleanupExpiredStates removes old input state files.
func (r *FileInputCoordinationRepository) CleanupExpiredStates(ctx context.Context, maxAge time.Duration) error {
	pattern := filepath.Join(r.baseDir, "*-input-state.json")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob input state files: %w", err)
	}

	cutoffTime := time.Now().Add(-maxAge)
	removed := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue // Skip files we can't stat
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err == nil {
				removed++
			}
		}
	}

	return nil
}
