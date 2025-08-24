package file

import (
	"context"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeRepository_GetNodesFromPublishedWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with nodes
	workflow := &models.Workflow{
		ID:   "test-workflow-nodes",
		Name: "Test Workflow with Nodes",
		Nodes: []*models.WorkflowNode{
			{
				ID:       "node1",
				Name:     "First Node",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"message": "test1"},
				Enabled:  true,
			},
			{
				ID:       "node2",
				Name:     "Second Node",
				NodeType: "transform",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"expression": "test2"},
				Enabled:  true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	// Save workflow
	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetNodesFromPublishedWorkflow
	nodeRepo := persistence.NodeRepository()
	nodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)

	// Verify
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "node1", nodes[0].ID)
	assert.Equal(t, "First Node", nodes[0].Name)
	assert.Equal(t, "node2", nodes[1].ID)
	assert.Equal(t, "Second Node", nodes[1].Name)
}

func TestNodeRepository_GetNodeFromPublishedWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow
	workflow := &models.Workflow{
		ID:   "test-workflow-single-node",
		Name: "Test Workflow Single Node",
		Nodes: []*models.WorkflowNode{
			{
				ID:       "target-node",
				Name:     "Target Node",
				NodeType: "httprequest",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"url":    "https://api.example.com",
					"method": "GET",
				},
				Enabled: true,
			},
			{
				ID:       "other-node",
				Name:     "Other Node",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"message": "other"},
				Enabled:  true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetNodeFromPublishedWorkflow
	nodeRepo := persistence.NodeRepository()
	node, err := nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "target-node")

	// Verify
	require.NoError(t, err)
	require.NotNil(t, node)
	assert.Equal(t, "target-node", node.ID)
	assert.Equal(t, "Target Node", node.Name)
	assert.Equal(t, "httprequest", node.NodeType)
	assert.Equal(t, "https://api.example.com", node.Config["url"])
}

func TestNodeRepository_GetNodeFromPublishedWorkflow_NotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow
	workflow := &models.Workflow{
		ID:     "test-workflow-not-found",
		Name:   "Test Workflow",
		Nodes:  []*models.WorkflowNode{},
		Status: models.WorkflowStatusActive,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetNodeFromPublishedWorkflow with non-existent node
	nodeRepo := persistence.NodeRepository()
	node, err := nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "non-existent-node")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "node not found")
}

func TestNodeRepository_SaveNode_NewNode(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow
	workflow := &models.Workflow{
		ID:     "test-workflow-save-node",
		Name:   "Test Workflow Save Node",
		Nodes:  []*models.WorkflowNode{},
		Status: models.WorkflowStatusActive,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Create new node to save
	newNode := &models.WorkflowNode{
		ID:       "new-node",
		Name:     "New Node",
		NodeType: "transform",
		Category: models.CategoryTypeAction,
		Config:   map[string]any{"expression": "{{.data}}"},
		Enabled:  true,
	}

	// Test SaveNode
	nodeRepo := persistence.NodeRepository()
	err = nodeRepo.SaveNode(ctx, workflow.ID, newNode)
	require.NoError(t, err)

	// Verify node was added
	nodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "new-node", nodes[0].ID)
	assert.Equal(t, "New Node", nodes[0].Name)
}

func TestNodeRepository_SaveNode_UpdateExisting(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with existing node
	existingNode := &models.WorkflowNode{
		ID:       "existing-node",
		Name:     "Original Name",
		NodeType: "log",
		Category: models.CategoryTypeAction,
		Config:   map[string]any{"message": "original"},
		Enabled:  true,
	}

	workflow := &models.Workflow{
		ID:     "test-workflow-update-node",
		Name:   "Test Workflow Update Node",
		Nodes:  []*models.WorkflowNode{existingNode},
		Status: models.WorkflowStatusActive,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Update the node
	updatedNode := &models.WorkflowNode{
		ID:       "existing-node",
		Name:     "Updated Name",
		NodeType: "log",
		Category: models.CategoryTypeAction,
		Config:   map[string]any{"message": "updated"},
		Enabled:  false,
	}

	// Test SaveNode (update)
	nodeRepo := persistence.NodeRepository()
	err = nodeRepo.SaveNode(ctx, workflow.ID, updatedNode)
	require.NoError(t, err)

	// Verify node was updated
	node, err := nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "existing-node")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", node.Name)
	assert.Equal(t, "updated", node.Config["message"])
	assert.False(t, node.Enabled)
}

func TestNodeRepository_DeleteNode(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with multiple nodes
	workflow := &models.Workflow{
		ID:   "test-workflow-delete-node",
		Name: "Test Workflow Delete Node",
		Nodes: []*models.WorkflowNode{
			{
				ID:       "node-to-delete",
				Name:     "Node To Delete",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"message": "delete me"},
				Enabled:  true,
			},
			{
				ID:       "node-to-keep",
				Name:     "Node To Keep",
				NodeType: "transform",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"expression": "keep me"},
				Enabled:  true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test DeleteNode
	nodeRepo := persistence.NodeRepository()
	err = nodeRepo.DeleteNode(ctx, workflow.ID, "node-to-delete")
	require.NoError(t, err)

	// Verify node was deleted
	nodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "node-to-keep", nodes[0].ID)

	// Verify deleted node is not found
	_, err = nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "node-to-delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node not found")
}

func TestNodeRepository_FindTriggerNodesBySourceEventAndProvider(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Helper to create string pointers
	stringPtr := func(s string) *string { return &s }

	// Create workflows with trigger nodes
	workflow1 := &models.Workflow{
		ID:   "workflow-1",
		Name: "Workflow 1",
		Nodes: []*models.WorkflowNode{
			{
				ID:         "trigger-1",
				Name:       "Kafka Trigger",
				NodeType:   "trigger:kafka",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("source-123"),
				EventType:  stringPtr("kafka_message"),
				ProviderID: stringPtr("kafka"),
				Config:     map[string]any{"topic": "orders"},
				Enabled:    true,
			},
			{
				ID:       "action-1",
				Name:     "Process Order",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config:   map[string]any{"message": "processing"},
				Enabled:  true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	workflow2 := &models.Workflow{
		ID:   "workflow-2",
		Name: "Workflow 2",
		Nodes: []*models.WorkflowNode{
			{
				ID:         "trigger-2",
				Name:       "Another Kafka Trigger",
				NodeType:   "trigger:kafka",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("source-123"),
				EventType:  stringPtr("kafka_message"),
				ProviderID: stringPtr("kafka"),
				Config:     map[string]any{"topic": "payments"},
				Enabled:    true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	workflow3 := &models.Workflow{
		ID:   "workflow-3",
		Name: "Workflow 3 - Different Provider",
		Nodes: []*models.WorkflowNode{
			{
				ID:         "trigger-3",
				Name:       "Schedule Trigger",
				NodeType:   "trigger:schedule",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("source-456"),
				EventType:  stringPtr("schedule_due"),
				ProviderID: stringPtr("scheduler"),
				Config:     map[string]any{"cron": "0 * * * *"},
				Enabled:    true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	// Save all workflows
	err := persistence.WorkflowRepository().Save(ctx, workflow1)
	require.NoError(t, err)
	err = persistence.WorkflowRepository().Save(ctx, workflow2)
	require.NoError(t, err)
	err = persistence.WorkflowRepository().Save(ctx, workflow3)
	require.NoError(t, err)

	// Test FindTriggerNodesBySourceEventAndProvider
	nodeRepo := persistence.NodeRepository()
	matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "source-123", "kafka_message", "kafka", models.WorkflowStatusActive)

	// Verify
	require.NoError(t, err)
	assert.Len(t, matches, 2)

	// Check that we got the right matches
	matchedWorkflowIDs := make([]string, 0, len(matches))
	for _, match := range matches {
		matchedWorkflowIDs = append(matchedWorkflowIDs, match.WorkflowID)
	}

	assert.Contains(t, matchedWorkflowIDs, "workflow-1")
	assert.Contains(t, matchedWorkflowIDs, "workflow-2")

	// Verify trigger node details
	for _, match := range matches {
		assert.NotNil(t, match.TriggerNode)
		assert.Equal(t, models.CategoryTypeTrigger, match.TriggerNode.Category)
		assert.Equal(t, "source-123", *match.TriggerNode.SourceID)
		assert.Equal(t, "kafka_message", *match.TriggerNode.EventType)
		assert.Equal(t, "kafka", *match.TriggerNode.ProviderID)
	}
}

func TestNodeRepository_FindTriggerNodesBySourceEventAndProvider_NoMatches(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Test with empty persistence
	nodeRepo := persistence.NodeRepository()
	matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "non-existent", "non-existent", "non-existent", models.WorkflowStatusActive)

	// Verify
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestNodeRepository_WorkflowNotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	nodeRepo := persistence.NodeRepository()

	// Test operations on non-existent workflow
	_, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, "non-existent-workflow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	_, err = nodeRepo.GetNodeFromPublishedWorkflow(ctx, "non-existent-workflow", "some-node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	node := &models.WorkflowNode{ID: "test", Name: "Test", NodeType: "log"}
	err = nodeRepo.SaveNode(ctx, "non-existent-workflow", node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	err = nodeRepo.DeleteNode(ctx, "non-existent-workflow", "some-node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}
