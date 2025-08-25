package file

import (
	"context"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionRepository_GetConnectionsFromPublishedWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with connections
	workflow := &models.Workflow{
		ID:   "test-workflow-connections",
		Name: "Test Workflow with Connections",
		Nodes: []*models.WorkflowNode{
			{ID: "node1", Name: "Node 1", NodeType: "log", Category: models.CategoryTypeAction},
			{ID: "node2", Name: "Node 2", NodeType: "transform", Category: models.CategoryTypeAction},
			{ID: "node3", Name: "Node 3", NodeType: "httprequest", Category: models.CategoryTypeAction},
		},
		Connections: []*models.Connection{
			{
				ID:         "conn1",
				SourcePort: "node1:success",
				TargetPort: "node2:main",
			},
			{
				ID:         "conn2",
				SourcePort: "node1:error",
				TargetPort: "node3:main",
			},
			{
				ID:         "conn3",
				SourcePort: "node2:success",
				TargetPort: "node3:secondary",
			},
		},
		Status: models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetConnectionsFromPublishedWorkflow for node1
	connRepo := persistence.ConnectionRepository()
	connections, err := connRepo.GetConnectionsFromPublishedWorkflow(ctx, workflow.ID, "node1")

	// Verify - should get 2 connections from node1
	require.NoError(t, err)
	assert.Len(t, connections, 2)

	sourcePortNames := make([]string, 0, len(connections))
	for _, conn := range connections {
		sourcePortNames = append(sourcePortNames, conn.SourcePort)
	}

	assert.Contains(t, sourcePortNames, "node1:success")
	assert.Contains(t, sourcePortNames, "node1:error")
}

func TestConnectionRepository_GetConnectionsByTargetNode(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with connections
	workflow := &models.Workflow{
		ID:   "test-workflow-target-connections",
		Name: "Test Workflow Target Connections",
		Nodes: []*models.WorkflowNode{
			{ID: "source1", Name: "Source 1", NodeType: "log"},
			{ID: "source2", Name: "Source 2", NodeType: "transform"},
			{ID: "target1", Name: "Target 1", NodeType: "httprequest"},
		},
		Connections: []*models.Connection{
			{
				ID:         "conn1",
				SourcePort: "source1:success",
				TargetPort: "target1:main",
			},
			{
				ID:         "conn2",
				SourcePort: "source2:success",
				TargetPort: "target1:secondary",
			},
			{
				ID:         "conn3",
				SourcePort: "source1:error",
				TargetPort: "source2:main", // Different target
			},
		},
		Status: models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetConnectionsByTargetNode for target1
	connRepo := persistence.ConnectionRepository()
	connections, err := connRepo.GetConnectionsByTargetNode(ctx, workflow.ID, "target1")

	// Verify - should get 2 connections to target1
	require.NoError(t, err)
	assert.Len(t, connections, 2)

	targetPortNames := make([]string, 0, len(connections))
	for _, conn := range connections {
		targetPortNames = append(targetPortNames, conn.TargetPort)
	}

	assert.Contains(t, targetPortNames, "target1:main")
	assert.Contains(t, targetPortNames, "target1:secondary")
}

func TestConnectionRepository_GetAllConnectionsFromPublishedWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with connections
	workflow := &models.Workflow{
		ID:   "test-workflow-all-connections",
		Name: "Test Workflow All Connections",
		Nodes: []*models.WorkflowNode{
			{ID: "nodeA", Name: "Node A", NodeType: "log"},
			{ID: "nodeB", Name: "Node B", NodeType: "transform"},
		},
		Connections: []*models.Connection{
			{
				ID:         "conn-all-1",
				SourcePort: "nodeA:success",
				TargetPort: "nodeB:main",
			},
			{
				ID:         "conn-all-2",
				SourcePort: "nodeA:error",
				TargetPort: "nodeB:error",
			},
		},
		Status: models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetAllConnectionsFromPublishedWorkflow
	connRepo := persistence.ConnectionRepository()
	connections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)

	// Verify
	require.NoError(t, err)
	assert.Len(t, connections, 2)

	connectionIDs := make([]string, 0, len(connections))
	for _, conn := range connections {
		connectionIDs = append(connectionIDs, conn.ID)
	}

	assert.Contains(t, connectionIDs, "conn-all-1")
	assert.Contains(t, connectionIDs, "conn-all-2")
}

func TestConnectionRepository_SaveConnection_NewConnection(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with nodes but no connections
	workflow := &models.Workflow{
		ID:   "test-workflow-save-connection",
		Name: "Test Workflow Save Connection",
		Nodes: []*models.WorkflowNode{
			{ID: "nodeX", Name: "Node X", NodeType: "log"},
			{ID: "nodeY", Name: "Node Y", NodeType: "transform"},
		},
		Connections: []*models.Connection{},
		Status:      models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Create new connection to save
	newConnection := &models.Connection{
		ID:         "new-connection",
		SourcePort: "nodeX:success",
		TargetPort: "nodeY:main",
	}

	// Test SaveConnection
	connRepo := persistence.ConnectionRepository()
	err = connRepo.SaveConnection(ctx, workflow.ID, newConnection)
	require.NoError(t, err)

	// Verify connection was added
	connections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, connections, 1)
	assert.Equal(t, "new-connection", connections[0].ID)
	assert.Equal(t, "nodeX:success", connections[0].SourcePort)
	assert.Equal(t, "nodeY:main", connections[0].TargetPort)
}

func TestConnectionRepository_SaveConnection_UpdateExisting(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with existing connection
	existingConnection := &models.Connection{
		ID:         "existing-connection",
		SourcePort: "nodeA:success",
		TargetPort: "nodeB:main",
	}

	workflow := &models.Workflow{
		ID:          "test-workflow-update-connection",
		Name:        "Test Workflow Update Connection",
		Nodes:       []*models.WorkflowNode{},
		Connections: []*models.Connection{existingConnection},
		Status:      models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Update the connection
	updatedConnection := &models.Connection{
		ID:         "existing-connection",
		SourcePort: "nodeA:error", // Changed
		TargetPort: "nodeB:error", // Changed
	}

	// Test SaveConnection (update)
	connRepo := persistence.ConnectionRepository()
	err = connRepo.SaveConnection(ctx, workflow.ID, updatedConnection)
	require.NoError(t, err)

	// Verify connection was updated
	connections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, connections, 1)
	assert.Equal(t, "existing-connection", connections[0].ID)
	assert.Equal(t, "nodeA:error", connections[0].SourcePort)
	assert.Equal(t, "nodeB:error", connections[0].TargetPort)
}

func TestConnectionRepository_DeleteConnection(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with multiple connections
	workflow := &models.Workflow{
		ID:    "test-workflow-delete-connection",
		Name:  "Test Workflow Delete Connection",
		Nodes: []*models.WorkflowNode{},
		Connections: []*models.Connection{
			{
				ID:         "connection-to-delete",
				SourcePort: "nodeA:success",
				TargetPort: "nodeB:main",
			},
			{
				ID:         "connection-to-keep",
				SourcePort: "nodeB:success",
				TargetPort: "nodeC:main",
			},
		},
		Status: models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test DeleteConnection
	connRepo := persistence.ConnectionRepository()
	err = connRepo.DeleteConnection(ctx, workflow.ID, "connection-to-delete")
	require.NoError(t, err)

	// Verify connection was deleted
	connections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, connections, 1)
	assert.Equal(t, "connection-to-keep", connections[0].ID)
}

func TestConnectionRepository_DeleteConnection_NotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow with no connections
	workflow := &models.Workflow{
		ID:          "test-workflow-delete-not-found",
		Name:        "Test Workflow Delete Not Found",
		Connections: []*models.Connection{},
		Status:      models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test DeleteConnection with non-existent connection
	connRepo := persistence.ConnectionRepository()
	err = connRepo.DeleteConnection(ctx, workflow.ID, "non-existent-connection")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

func TestConnectionRepository_GetConnectionsByWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a test workflow
	workflow := &models.Workflow{
		ID:   "test-workflow-by-workflow",
		Name: "Test Workflow By Workflow",
		Connections: []*models.Connection{
			{ID: "conn1", SourcePort: "a:out", TargetPort: "b:in"},
			{ID: "conn2", SourcePort: "b:out", TargetPort: "c:in"},
		},
		Status: models.WorkflowStatusPublished,
	}

	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test GetConnectionsByWorkflow
	connRepo := persistence.ConnectionRepository()
	connections, err := connRepo.GetConnectionsByWorkflow(ctx, workflow.ID)

	// Verify
	require.NoError(t, err)
	assert.Len(t, connections, 2)
}

func TestConnectionRepository_WorkflowNotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	connRepo := persistence.ConnectionRepository()

	// Test operations on non-existent workflow
	_, err := connRepo.GetConnectionsFromPublishedWorkflow(ctx, "non-existent-workflow", "some-node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	_, err = connRepo.GetConnectionsByTargetNode(ctx, "non-existent-workflow", "some-node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	_, err = connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, "non-existent-workflow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	connection := &models.Connection{ID: "test", SourcePort: "a:out", TargetPort: "b:in"}
	err = connRepo.SaveConnection(ctx, "non-existent-workflow", connection)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	err = connRepo.DeleteConnection(ctx, "non-existent-workflow", "some-connection")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}
