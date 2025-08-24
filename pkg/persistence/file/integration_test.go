package file

import (
	"context"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestWorkflow creates a complete workflow for integration testing.
func createTestWorkflow() *models.Workflow {
	stringPtr := func(s string) *string { return &s }

	return &models.Workflow{
		ID:   "integration-workflow-complete",
		Name: "Integration Test Complete Workflow",
		Nodes: []*models.WorkflowNode{
			{
				ID:         "kafka-trigger",
				Name:       "Kafka Order Trigger",
				NodeType:   "trigger:kafka",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("kafka-source"),
				EventType:  stringPtr("kafka_message"),
				ProviderID: stringPtr("kafka"),
				Config: map[string]any{
					"topic":          "orders",
					"consumer_group": "order-processor",
				},
				Enabled: true,
			},
			{
				ID:       "transform-order",
				Name:     "Transform Order Data",
				NodeType: "transform",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"expression": `{
						"orderId": "{{.trigger_data.message.id}}",
						"customerId": "{{.trigger_data.message.customer_id}}",
						"totalAmount": {{.trigger_data.message.total}},
						"processedAt": "{{now}}"
					}`,
				},
				Enabled: true,
			},
			{
				ID:       "validate-order",
				Name:     "Validate Order",
				NodeType: "httprequest",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"url":    "https://api.example.com/validate",
					"method": "POST",
					"headers": map[string]any{
						"Content-Type": "application/json",
					},
					"body": `{{.step_results.transform_order}}`,
					"retries": map[string]any{
						"attempts": 3,
						"delay":    1000,
					},
				},
				Enabled: true,
			},
			{
				ID:       "log-result",
				Name:     "Log Processing Result",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"message": "Order {{.step_results.transform_order.orderId}} processed with status: {{.step_results.validate_order.status}}",
					"level":   "info",
				},
				Enabled: true,
			},
		},
		Connections: []*models.Connection{
			{
				ID:         "trigger-to-transform",
				SourcePort: "kafka-trigger:success",
				TargetPort: "transform-order:main",
			},
			{
				ID:         "transform-to-validate",
				SourcePort: "transform-order:success",
				TargetPort: "validate-order:main",
			},
			{
				ID:         "validate-to-log",
				SourcePort: "validate-order:success",
				TargetPort: "log-result:main",
			},
		},
		Variables: map[string]any{
			"api_timeout": 30,
			"retry_count": 3,
		},
		Metadata: map[string]any{
			"version":     "1.0",
			"description": "Processes incoming Kafka order messages",
		},
		Status: models.WorkflowStatusActive,
	}
}

// testNodeRepositoryOperations tests node repository functionality.
func testNodeRepositoryOperations(t *testing.T, persistence *Persistence, workflow *models.Workflow, ctx context.Context) {
	t.Helper()

	nodeRepo := persistence.NodeRepository()

	// Get all nodes
	allNodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, allNodes, 4)

	// Get specific trigger node
	triggerNode, err := nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "kafka-trigger")
	require.NoError(t, err)
	assert.Equal(t, "kafka-trigger", triggerNode.ID)
	assert.Equal(t, models.CategoryTypeTrigger, triggerNode.Category)
	assert.Equal(t, "kafka-source", *triggerNode.SourceID)
	assert.Equal(t, "kafka_message", *triggerNode.EventType)
	assert.Equal(t, "kafka", *triggerNode.ProviderID)

	// Find trigger nodes by source/event/provider
	triggerMatches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "kafka-source", "kafka_message", "kafka", models.WorkflowStatusActive)
	require.NoError(t, err)
	assert.Len(t, triggerMatches, 1)
	assert.Equal(t, workflow.ID, triggerMatches[0].WorkflowID)
	assert.Equal(t, "kafka-trigger", triggerMatches[0].TriggerNode.ID)
}

// testConnectionRepositoryOperations tests connection repository functionality.
func testConnectionRepositoryOperations(t *testing.T, persistence *Persistence, workflow *models.Workflow, ctx context.Context) {
	t.Helper()

	connRepo := persistence.ConnectionRepository()

	// Get all connections
	allConnections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, allConnections, 3)

	// Get connections from trigger node
	triggerConnections, err := connRepo.GetConnectionsFromPublishedWorkflow(ctx, workflow.ID, "kafka-trigger")
	require.NoError(t, err)
	assert.Len(t, triggerConnections, 1)
	assert.Equal(t, "kafka-trigger:success", triggerConnections[0].SourcePort)
	assert.Equal(t, "transform-order:main", triggerConnections[0].TargetPort)

	// Get connections to transform node
	transformConnections, err := connRepo.GetConnectionsByTargetNode(ctx, workflow.ID, "transform-order")
	require.NoError(t, err)
	assert.Len(t, transformConnections, 1)
	assert.Equal(t, "trigger-to-transform", transformConnections[0].ID)
}

// createTestExecutionContext creates test execution context.
func createTestExecutionContext(workflowID string) *models.ExecutionContext {
	return &models.ExecutionContext{
		ID:                  "exec-integration-test",
		PublishedWorkflowID: workflowID,
		Status:              models.ExecutionStatusRunning,
		NodeResults: map[string]models.NodeResult{
			"kafka-trigger::success": {
				NodeID: "kafka-trigger",
				Data: map[string]any{
					"trigger_executed": true,
					"message_id":       "msg-12345",
				},
				Status: string(models.NodeStatusSuccess),
			},
		},
		TriggerData: map[string]any{
			"topic":     "orders",
			"partition": 0,
			"offset":    12345,
			"key":       "order-67890",
			"message": map[string]any{
				"id":          "order-67890",
				"customer_id": "cust-123",
				"total":       99.99,
				"items": []any{
					map[string]any{
						"sku":      "ITEM-001",
						"quantity": 2,
						"price":    29.99,
					},
					map[string]any{
						"sku":      "ITEM-002",
						"quantity": 1,
						"price":    39.99,
					},
				},
			},
		},
		Variables: map[string]any{
			"api_timeout": 30,
			"retry_count": 3,
		},
		Metadata: map[string]any{
			"processing_start": time.Now().Unix(),
		},
		CreatedAt: time.Now().Add(-time.Minute),
	}
}

// testExecutionContextOperations tests execution context repository functionality.
func testExecutionContextOperations(t *testing.T, persistence *Persistence, workflow *models.Workflow, ctx context.Context) {
	t.Helper()

	execRepo := persistence.ExecutionContextRepository()

	// Create execution context for workflow
	executionCtx := createTestExecutionContext(workflow.ID)

	// Save execution context
	err := execRepo.SaveExecutionContext(ctx, executionCtx)
	require.NoError(t, err)

	// Get execution context by ID
	retrievedCtx, err := execRepo.GetExecutionContext(ctx, executionCtx.ID)
	require.NoError(t, err)
	assert.Equal(t, executionCtx.ID, retrievedCtx.ID)
	assert.Equal(t, executionCtx.PublishedWorkflowID, retrievedCtx.PublishedWorkflowID)
	assert.Equal(t, executionCtx.Status, retrievedCtx.Status)

	// Update execution status
	executionCtx.Status = models.ExecutionStatusCompleted
	err = execRepo.UpdateExecutionContext(ctx, executionCtx)
	require.NoError(t, err)

	// Verify status update
	updatedCtx, err := execRepo.GetExecutionContext(ctx, executionCtx.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, updatedCtx.Status)

	// Test querying executions by workflow
	workflowExecutions, err := execRepo.GetExecutionsByWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, workflowExecutions, 1)
	assert.Equal(t, "exec-integration-test", workflowExecutions[0].ID)

	// Test querying executions by status
	completedExecutions, err := execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusCompleted)
	require.NoError(t, err)
	assert.Len(t, completedExecutions, 1)
	assert.Equal(t, "exec-integration-test", completedExecutions[0].ID)
}

func TestNodeBasedWorkflowExecution_CompleteFlow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create test workflow
	workflow := createTestWorkflow()

	// Save the complete workflow
	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Test all repository operations
	testNodeRepositoryOperations(t, persistence.(*Persistence), workflow, ctx)
	testConnectionRepositoryOperations(t, persistence.(*Persistence), workflow, ctx)
	testExecutionContextOperations(t, persistence.(*Persistence), workflow, ctx)
}

func TestNodeBasedWorkflowExecution_MultipleWorkflows(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Helper to create string pointers
	stringPtr := func(s string) *string { return &s }

	// Create two workflows with the same trigger configuration
	workflow1 := &models.Workflow{
		ID:   "workflow-1",
		Name: "Workflow 1",
		Nodes: []*models.WorkflowNode{
			{
				ID:         "trigger-1",
				Name:       "Webhook Trigger 1",
				NodeType:   "trigger:webhook",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("webhook-source"),
				EventType:  stringPtr("webhook_received"),
				ProviderID: stringPtr("webhook"),
				Config:     map[string]any{},
				Enabled:    true,
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
				Name:       "Webhook Trigger 2",
				NodeType:   "trigger:webhook",
				Category:   models.CategoryTypeTrigger,
				SourceID:   stringPtr("webhook-source"),
				EventType:  stringPtr("webhook_received"),
				ProviderID: stringPtr("webhook"),
				Config:     map[string]any{},
				Enabled:    true,
			},
		},
		Status: models.WorkflowStatusActive,
	}

	// Save both workflows
	err := persistence.WorkflowRepository().Save(ctx, workflow1)
	require.NoError(t, err)

	err = persistence.WorkflowRepository().Save(ctx, workflow2)
	require.NoError(t, err)

	// Test finding trigger nodes by source/event/provider
	nodeRepo := persistence.NodeRepository()
	matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "webhook-source", "webhook_received", "webhook", models.WorkflowStatusActive)
	require.NoError(t, err)
	assert.Len(t, matches, 2)

	// Verify both workflows are matched
	matchedWorkflowIDs := make([]string, 0, len(matches))
	for _, match := range matches {
		matchedWorkflowIDs = append(matchedWorkflowIDs, match.WorkflowID)
	}

	assert.Contains(t, matchedWorkflowIDs, "workflow-1")
	assert.Contains(t, matchedWorkflowIDs, "workflow-2")
}

func TestNodeBasedWorkflowExecution_DynamicNodeOperations(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create a basic workflow
	workflow := &models.Workflow{
		ID:          "dynamic-workflow",
		Name:        "Dynamic Workflow Operations",
		Nodes:       []*models.WorkflowNode{},
		Connections: []*models.Connection{},
		Variables:   map[string]any{},
		Metadata:    map[string]any{},
		Status:      models.WorkflowStatusActive,
	}

	// Save the workflow
	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Add a new node dynamically
	newNode := &models.WorkflowNode{
		ID:       "dynamic-node-1",
		Name:     "Dynamic Node",
		NodeType: "log",
		Category: models.CategoryTypeAction,
		Config: map[string]any{
			"message": "Dynamic node test",
			"level":   "info",
		},
		Enabled: true,
	}

	// Add the node to the workflow
	workflow.Nodes = append(workflow.Nodes, newNode)
	err = persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// Verify the node was added
	retrievedWorkflow, err := persistence.WorkflowRepository().GetByID(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, retrievedWorkflow.Nodes, 1)
	assert.Equal(t, "dynamic-node-1", retrievedWorkflow.Nodes[0].ID)
}

func TestNodeBasedWorkflowExecution_ErrorScenarios(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Test error scenarios with non-existent workflows and nodes
	nodeRepo := persistence.NodeRepository()

	// Try to get nodes from non-existent workflow
	_, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, "non-existent-workflow")
	assert.Error(t, err)

	// Try to get specific node from non-existent workflow
	_, err = nodeRepo.GetNodeFromPublishedWorkflow(ctx, "non-existent-workflow", "some-node")
	assert.Error(t, err)

	// Test empty repository operations
	matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "any-source", "any-event", "any-provider", models.WorkflowStatusActive)
	require.NoError(t, err)
	assert.Empty(t, matches)

	// Test execution context repository with empty data
	execRepo := persistence.ExecutionContextRepository()
	executions, err := execRepo.GetExecutionsByWorkflow(ctx, "any-workflow")
	require.NoError(t, err)
	assert.Empty(t, executions)

	executions, err = execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusRunning)
	require.NoError(t, err)
	assert.Empty(t, executions)
}
