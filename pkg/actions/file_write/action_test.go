package file_write_action

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileWriteAction(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *FileWriteAction
	}{
		{
			name: "basic file write action",
			config: map[string]interface{}{
				"id":        "test-file-1",
				"file_name": "test.json",
				"directory": "/tmp/test",
				"overwrite": true,
				"input":     "data",
			},
			expected: &FileWriteAction{
				ID:        "test-file-1",
				FileName:  "test.json",
				Directory: "/tmp/test",
				Overwrite: true,
				Input:     "data",
			},
		},
		{
			name: "default directory when not specified",
			config: map[string]interface{}{
				"id":        "test-file-2",
				"file_name": "output.json",
				"overwrite": false,
			},
			expected: &FileWriteAction{
				ID:        "test-file-2",
				FileName:  "output.json",
				Directory: "/tmp",
				Overwrite: false,
				Input:     "",
			},
		},
		{
			name:   "empty config with defaults",
			config: map[string]interface{}{},
			expected: &FileWriteAction{
				ID:        "",
				FileName:  "",
				Directory: "/tmp",
				Overwrite: false,
				Input:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewFileWriteAction(tt.config)

			require.NoError(t, err)
			assert.Equal(t, tt.expected.ID, action.ID)
			assert.Equal(t, tt.expected.FileName, action.FileName)
			assert.Equal(t, tt.expected.Directory, action.Directory)
			assert.Equal(t, tt.expected.Overwrite, action.Overwrite)
			assert.Equal(t, tt.expected.Input, action.Input)
		})
	}
}

func TestFileWriteAction_GetMethods(t *testing.T) {
	action := &FileWriteAction{
		ID:        "test-file",
		FileName:  "output.json",
		Directory: "/tmp/test",
		Overwrite: true,
		Input:     "data.results",
	}

	assert.Equal(t, "test-file", action.GetID())
	assert.Equal(t, "file_write", action.GetType())

	config := action.GetConfig()
	assert.Equal(t, "test-file", config["id"])
	assert.Equal(t, "output.json", config["file_name"])
	assert.Equal(t, "/tmp/test", config["directory"])
	assert.Equal(t, true, config["overwrite"])

	assert.NoError(t, action.Validate())
}

func TestFileWriteAction_Execute_Success(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	action := &FileWriteAction{
		ID:        "test-write",
		FileName:  "test_output.json",
		Directory: tempDir,
		Overwrite: true,
		Input:     "",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"name":  "John Doe",
			"age":   30,
			"email": "john@example.com",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	expectedPath := filepath.Join(tempDir, "test_output.json")
	assert.Equal(t, expectedPath, resultMap["file_path"])
	assert.Greater(t, resultMap["bytes_written"], 0)
	assert.True(t, resultMap["success"].(bool))

	// Verify the file was created and contains the correct data
	assert.FileExists(t, expectedPath)

	fileContent, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	var writtenData map[string]interface{}
	err = json.Unmarshal(fileContent, &writtenData)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", writtenData["name"])
	assert.Equal(t, float64(30), writtenData["age"])
	assert.Equal(t, "john@example.com", writtenData["email"])
}

func TestFileWriteAction_Execute_WithInputExpression(t *testing.T) {
	tempDir := t.TempDir()

	action := &FileWriteAction{
		ID:        "test-input-expr",
		FileName:  "filtered_output.json",
		Directory: tempDir,
		Overwrite: true,
		Input:     "users[0]",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{
					"name": "Alice",
					"age":  25,
				},
				map[string]interface{}{
					"name": "Bob",
					"age":  30,
				},
			},
		},
	}

	_, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "filtered_output.json")
	assert.FileExists(t, expectedPath)

	fileContent, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	var writtenData map[string]interface{}
	err = json.Unmarshal(fileContent, &writtenData)
	require.NoError(t, err)

	// Should only contain the first user
	assert.Equal(t, "Alice", writtenData["name"])
	assert.Equal(t, float64(25), writtenData["age"])
}

func TestFileWriteAction_Execute_OverwriteFalse_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	fileName := "existing_file.json"
	filePath := filepath.Join(tempDir, fileName)

	// Create an existing file
	existingData := map[string]interface{}{"existing": "data"}
	existingJSON, _ := json.Marshal(existingData)
	err := os.WriteFile(filePath, existingJSON, 0644)
	require.NoError(t, err)

	action := &FileWriteAction{
		ID:        "test-no-overwrite",
		FileName:  fileName,
		Directory: tempDir,
		Overwrite: false, // Don't overwrite
		Input:     "",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"new": "data",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists and overwrite is false")

	// Verify original file is unchanged
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var actualData map[string]interface{}
	err = json.Unmarshal(fileContent, &actualData)
	require.NoError(t, err)

	assert.Equal(t, "data", actualData["existing"])
	assert.NotContains(t, actualData, "new")
}

func TestFileWriteAction_Execute_OverwriteTrue_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	fileName := "overwrite_file.json"
	filePath := filepath.Join(tempDir, fileName)

	// Create an existing file
	existingData := map[string]interface{}{"old": "data"}
	existingJSON, _ := json.Marshal(existingData)
	err := os.WriteFile(filePath, existingJSON, 0644)
	require.NoError(t, err)

	action := &FileWriteAction{
		ID:        "test-overwrite",
		FileName:  fileName,
		Directory: tempDir,
		Overwrite: true, // Allow overwrite
		Input:     "",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"new": "data",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify file was overwritten
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var actualData map[string]interface{}
	err = json.Unmarshal(fileContent, &actualData)
	require.NoError(t, err)

	assert.Equal(t, "data", actualData["new"])
	assert.NotContains(t, actualData, "old")
}

func TestFileWriteAction_Execute_CreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "deep", "directory")

	action := &FileWriteAction{
		ID:        "test-create-dir",
		FileName:  "deep_file.json",
		Directory: nestedDir,
		Overwrite: true,
		Input:     "",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"message": "created in deep directory",
		},
	}

	_, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)

	expectedPath := filepath.Join(nestedDir, "deep_file.json")
	assert.FileExists(t, expectedPath)

	// Verify directory was created
	assert.DirExists(t, nestedDir)
}

func TestFileWriteAction_Execute_InvalidInputExpression(t *testing.T) {
	tempDir := t.TempDir()

	action := &FileWriteAction{
		ID:        "test-invalid-input",
		FileName:  "output.json",
		Directory: tempDir,
		Overwrite: true,
		Input:     "invalid((expression",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"data": "test",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to extract input data")
}

func TestFileWriteAction_Execute_ComplexData(t *testing.T) {
	tempDir := t.TempDir()

	action := &FileWriteAction{
		ID:        "test-complex",
		FileName:  "complex_data.json",
		Directory: tempDir,
		Overwrite: true,
		Input:     "",
	}

	logger := log.WithField("test", "file_write_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{
					"id":     1,
					"name":   "John",
					"active": true,
					"scores": []interface{}{85, 92, 78},
					"profile": map[string]interface{}{
						"email": "john@example.com",
						"age":   30,
					},
				},
				map[string]interface{}{
					"id":     2,
					"name":   "Jane",
					"active": false,
					"scores": []interface{}{90, 88, 95},
					"profile": map[string]interface{}{
						"email": "jane@example.com",
						"age":   28,
					},
				},
			},
			"metadata": map[string]interface{}{
				"total_users": 2,
				"generated":   "2024-01-15T10:30:00Z",
			},
		},
	}

	_, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "complex_data.json")
	assert.FileExists(t, expectedPath)

	// Verify the file contains properly formatted JSON
	fileContent, err := os.ReadFile(expectedPath)
	require.NoError(t, err)

	var writtenData map[string]interface{}
	err = json.Unmarshal(fileContent, &writtenData)
	require.NoError(t, err)

	// Verify structure is preserved
	users := writtenData["users"].([]interface{})
	assert.Len(t, users, 2)

	firstUser := users[0].(map[string]interface{})
	assert.Equal(t, "John", firstUser["name"])
	assert.Equal(t, true, firstUser["active"])

	metadata := writtenData["metadata"].(map[string]interface{})
	assert.Equal(t, float64(2), metadata["total_users"])
}

func TestFileWriteAction_GetConfig_Consistency(t *testing.T) {
	config := map[string]interface{}{
		"id":        "config-test",
		"file_name": "test.json",
		"directory": "/tmp/test",
		"overwrite": true,
		"input":     "data.items",
	}

	action, err := NewFileWriteAction(config)
	require.NoError(t, err)

	retrievedConfig := action.GetConfig()

	// Config should match the original action properties (note: input is not included in GetConfig)
	assert.Equal(t, action.ID, retrievedConfig["id"])
	assert.Equal(t, action.FileName, retrievedConfig["file_name"])
	assert.Equal(t, action.Directory, retrievedConfig["directory"])
	assert.Equal(t, action.Overwrite, retrievedConfig["overwrite"])
}
