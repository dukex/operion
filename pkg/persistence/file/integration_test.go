package file

import (
	"context"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeBasedWorkflowExecution_CompleteFlow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Helper to create string pointers
	stringPtr := func(s string) *string { return &s }

	// Create a complete workflow with trigger and action nodes
	workflow := &models.Workflow{
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

	// 1. Save the complete workflow
	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	// 2. Test NodeRepository operations
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

	// 3. Test ConnectionRepository operations
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

	// 4. Test ExecutionContext operations
	execRepo := persistence.ExecutionContextRepository()

	// Create execution context for workflow
	executionCtx := &models.ExecutionContext{
		ID:                  "exec-integration-test",
		PublishedWorkflowID: workflow.ID,
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
			"headers": map[string]string{
				"source":       "order-service",
				"content-type": "application/json",
			},
		},
		Variables: map[string]any{
			"api_timeout": 30,
			"retry_count": 3,
		},
		Metadata: map[string]any{
			"execution_start": time.Now().UTC().Format(time.RFC3339),
			"worker_id":       "worker-001",
		},
		CreatedAt: time.Now(),
	}

	// Save execution context
	err = execRepo.SaveExecutionContext(ctx, executionCtx)
	require.NoError(t, err)

	// Retrieve and verify execution context
	retrievedExec, err := execRepo.GetExecutionContext(ctx, "exec-integration-test")
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, retrievedExec.PublishedWorkflowID)
	assert.Equal(t, models.ExecutionStatusRunning, retrievedExec.Status)
	assert.Equal(t, "order-67890", retrievedExec.TriggerData["key"])
	assert.Equal(t, "cust-123", retrievedExec.TriggerData["message"].(map[string]any)["customer_id"])

	// 5. Simulate workflow execution progress
	// Update execution with transform node result
	executionCtx.NodeResults["transform-order::success"] = models.NodeResult{
		NodeID: "transform-order",
		Data: map[string]any{
			"orderId":     "order-67890",
			"customerId":  "cust-123",
			"totalAmount": 99.99,
			"processedAt": time.Now().UTC().Format(time.RFC3339),
		},
		Status: string(models.NodeStatusSuccess),
	}

	// Update execution with validation result
	executionCtx.NodeResults["validate-order::success"] = models.NodeResult{
		NodeID: "validate-order",
		Data: map[string]any{
			"status":        "validated",
			"validation_id": "val-54321",
			"approved":      true,
		},
		Status: string(models.NodeStatusSuccess),
	}

	// Update execution with final log result
	executionCtx.NodeResults["log-result::success"] = models.NodeResult{
		NodeID: "log-result",
		Data: map[string]any{
			"logged":  true,
			"message": "Order order-67890 processed with status: validated",
		},
		Status: string(models.NodeStatusSuccess),
	}

	// Mark execution as completed
	completedAt := time.Now()
	executionCtx.Status = models.ExecutionStatusCompleted
	executionCtx.CompletedAt = &completedAt

	// Update execution context
	err = execRepo.UpdateExecutionContext(ctx, executionCtx)
	require.NoError(t, err)

	// Verify final execution state
	finalExec, err := execRepo.GetExecutionContext(ctx, "exec-integration-test")
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, finalExec.Status)
	assert.NotNil(t, finalExec.CompletedAt)
	assert.Len(t, finalExec.NodeResults, 4) // All nodes executed

	// Verify each node result
	triggerResult := finalExec.NodeResults["kafka-trigger::success"]
	assert.Equal(t, "kafka-trigger", triggerResult.NodeID)
	assert.Equal(t, string(models.NodeStatusSuccess), triggerResult.Status)

	transformResult := finalExec.NodeResults["transform-order::success"]
	assert.Equal(t, "transform-order", transformResult.NodeID)
	assert.Equal(t, "order-67890", transformResult.Data["orderId"])

	validateResult := finalExec.NodeResults["validate-order::success"]
	assert.Equal(t, "validate-order", validateResult.NodeID)
	assert.Equal(t, "validated", validateResult.Data["status"])

	logResult := finalExec.NodeResults["log-result::success"]
	assert.Equal(t, "log-result", logResult.NodeID)
	assert.True(t, logResult.Data["logged"].(bool))

	// 6. Test querying executions by workflow
	workflowExecutions, err := execRepo.GetExecutionsByWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, workflowExecutions, 1)
	assert.Equal(t, "exec-integration-test", workflowExecutions[0].ID)

	// 7. Test querying executions by status
	completedExecutions, err := execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusCompleted)
	require.NoError(t, err)
	assert.Len(t, completedExecutions, 1)
	assert.Equal(t, "exec-integration-test", completedExecutions[0].ID)
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
				Config: map[string]any{
					"path": "/orders",
				},
				Enabled: true,
			},
			{
				ID:       "action-1",
				Name:     "Process in Workflow 1",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"message": "Processing in workflow 1: {{.trigger_data.webhook.body}}",
				},
				Enabled: true,
			},
		},
		Connections: []*models.Connection{
			{
				ID:         "conn-1",
				SourcePort: "trigger-1:success",
				TargetPort: "action-1:main",
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
				Config: map[string]any{
					"path": "/orders",
				},
				Enabled: true,
			},
			{
				ID:       "action-2",
				Name:     "Process in Workflow 2",
				NodeType: "log",
				Category: models.CategoryTypeAction,
				Config: map[string]any{
					"message": "Processing in workflow 2: {{.trigger_data.webhook.body}}",
				},
				Enabled: true,
			},
		},
		Connections: []*models.Connection{
			{
				ID:         "conn-2",
				SourcePort: "trigger-2:success",
				TargetPort: "action-2:main",
			},
		},
		Status: models.WorkflowStatusActive,
	}

	// Save both workflows
	err := persistence.WorkflowRepository().Save(ctx, workflow1)
	require.NoError(t, err)
	err = persistence.WorkflowRepository().Save(ctx, workflow2)
	require.NoError(t, err)

	// Test finding trigger nodes - should find both workflows
	nodeRepo := persistence.NodeRepository()
	matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
		ctx, "webhook-source", "webhook_received", "webhook", models.WorkflowStatusActive)
	require.NoError(t, err)
	assert.Len(t, matches, 2)

	// Verify both workflows are matched
	var matchedWorkflowIDs []string
	for _, match := range matches {
		matchedWorkflowIDs = append(matchedWorkflowIDs, match.WorkflowID)
	}

	assert.Contains(t, matchedWorkflowIDs, "workflow-1")
	assert.Contains(t, matchedWorkflowIDs, "workflow-2")

	// Create execution contexts for both workflows
	execRepo := persistence.ExecutionContextRepository()

	exec1 := &models.ExecutionContext{
		ID:                  "exec-workflow-1",
		PublishedWorkflowID: "workflow-1",
		Status:              models.ExecutionStatusRunning,
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"path": "/orders",
				"body": `{"order_id": "order-123", "amount": 50.00}`,
			},
		},
		CreatedAt: time.Now(),
	}

	exec2 := &models.ExecutionContext{
		ID:                  "exec-workflow-2",
		PublishedWorkflowID: "workflow-2",
		Status:              models.ExecutionStatusRunning,
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"path": "/orders",
				"body": `{"order_id": "order-123", "amount": 50.00}`,
			},
		},
		CreatedAt: time.Now(),
	}

	// Save both execution contexts
	err = execRepo.SaveExecutionContext(ctx, exec1)
	require.NoError(t, err)
	err = execRepo.SaveExecutionContext(ctx, exec2)
	require.NoError(t, err)

	// Test querying executions by status - should find both
	runningExecutions, err := execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusRunning)
	require.NoError(t, err)
	assert.Len(t, runningExecutions, 2)

	// Test querying executions by specific workflow
	workflow1Executions, err := execRepo.GetExecutionsByWorkflow(ctx, "workflow-1")
	require.NoError(t, err)
	assert.Len(t, workflow1Executions, 1)
	assert.Equal(t, "exec-workflow-1", workflow1Executions[0].ID)

	workflow2Executions, err := execRepo.GetExecutionsByWorkflow(ctx, "workflow-2")
	require.NoError(t, err)
	assert.Len(t, workflow2Executions, 1)
	assert.Equal(t, "exec-workflow-2", workflow2Executions[0].ID)
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
		Status:      models.WorkflowStatusActive,
	}

	// Save empty workflow
	err := persistence.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	nodeRepo := persistence.NodeRepository()
	connRepo := persistence.ConnectionRepository()

	// 1. Add nodes dynamically
	triggerNode := &models.WorkflowNode{
		ID:       "dynamic-trigger",
		Name:     "Dynamic Trigger",
		NodeType: "trigger:manual",
		Category: models.CategoryTypeTrigger,
		Config:   map[string]any{"manual": true},
		Enabled:  true,
	}

	actionNode := &models.WorkflowNode{
		ID:       "dynamic-action",
		Name:     "Dynamic Action",
		NodeType: "log",
		Category: models.CategoryTypeAction,
		Config:   map[string]any{"message": "Dynamic execution"},
		Enabled:  true,
	}

	// Save nodes
	err = nodeRepo.SaveNode(ctx, workflow.ID, triggerNode)
	require.NoError(t, err)
	err = nodeRepo.SaveNode(ctx, workflow.ID, actionNode)
	require.NoError(t, err)

	// Verify nodes were added
	nodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// 2. Add connection dynamically
	connection := &models.Connection{
		ID:         "dynamic-connection",
		SourcePort: "dynamic-trigger:success",
		TargetPort: "dynamic-action:main",
	}

	err = connRepo.SaveConnection(ctx, workflow.ID, connection)
	require.NoError(t, err)

	// Verify connection was added
	connections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, connections, 1)
	assert.Equal(t, "dynamic-connection", connections[0].ID)

	// 3. Update node configuration
	triggerNode.Config["manual"] = false
	triggerNode.Config["auto_trigger"] = true

	err = nodeRepo.SaveNode(ctx, workflow.ID, triggerNode)
	require.NoError(t, err)

	// Verify node was updated
	updatedNode, err := nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "dynamic-trigger")
	require.NoError(t, err)
	assert.False(t, updatedNode.Config["manual"].(bool))
	assert.True(t, updatedNode.Config["auto_trigger"].(bool))

	// 4. Update connection
	connection.TargetPort = "dynamic-action:secondary"
	err = connRepo.SaveConnection(ctx, workflow.ID, connection)
	require.NoError(t, err)

	// Verify connection was updated
	updatedConnections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, updatedConnections, 1)
	assert.Equal(t, "dynamic-action:secondary", updatedConnections[0].TargetPort)

	// 5. Delete connection
	err = connRepo.DeleteConnection(ctx, workflow.ID, "dynamic-connection")
	require.NoError(t, err)

	// Verify connection was deleted
	finalConnections, err := connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, finalConnections, 0)

	// 6. Delete node
	err = nodeRepo.DeleteNode(ctx, workflow.ID, "dynamic-action")
	require.NoError(t, err)

	// Verify node was deleted
	finalNodes, err := nodeRepo.GetNodesFromPublishedWorkflow(ctx, workflow.ID)
	require.NoError(t, err)
	assert.Len(t, finalNodes, 1)
	assert.Equal(t, "dynamic-trigger", finalNodes[0].ID)

	// Verify deleted node cannot be retrieved
	_, err = nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "dynamic-action")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node not found")
}

func TestNodeBasedWorkflowExecution_ErrorScenarios(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	nodeRepo := persistence.NodeRepository()
	connRepo := persistence.ConnectionRepository()
	execRepo := persistence.ExecutionContextRepository()

	// Test operations on non-existent workflow
	t.Run("NonExistentWorkflow", func(t *testing.T) {
		// Node operations
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

		// Connection operations
		_, err = connRepo.GetAllConnectionsFromPublishedWorkflow(ctx, "non-existent-workflow")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")

		_, err = connRepo.GetConnectionsFromPublishedWorkflow(ctx, "non-existent-workflow", "some-node")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")

		connection := &models.Connection{ID: "test", SourcePort: "a:out", TargetPort: "b:in"}
		err = connRepo.SaveConnection(ctx, "non-existent-workflow", connection)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")

		err = connRepo.DeleteConnection(ctx, "non-existent-workflow", "some-connection")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")
	})

	// Test operations on non-existent entities
	t.Run("NonExistentEntities", func(t *testing.T) {
		// Create a workflow for testing
		workflow := &models.Workflow{
			ID:     "test-error-workflow",
			Name:   "Test Error Workflow",
			Nodes:  []*models.WorkflowNode{},
			Status: models.WorkflowStatusActive,
		}

		err := persistence.WorkflowRepository().Save(ctx, workflow)
		require.NoError(t, err)

		// Test non-existent node
		_, err = nodeRepo.GetNodeFromPublishedWorkflow(ctx, workflow.ID, "non-existent-node")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node not found")

		// Test non-existent execution context
		_, err = execRepo.GetExecutionContext(ctx, "non-existent-execution")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution context not found")

		// Test updating non-existent execution context
		execCtx := &models.ExecutionContext{
			ID:                  "non-existent-execution",
			PublishedWorkflowID: workflow.ID,
			Status:              models.ExecutionStatusRunning,
			CreatedAt:           time.Now(),
		}
		err = execRepo.UpdateExecutionContext(ctx, execCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution context not found")

		// Test deleting non-existent connection
		err = connRepo.DeleteConnection(ctx, workflow.ID, "non-existent-connection")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	// Test empty repository operations
	t.Run("EmptyRepositoryOperations", func(t *testing.T) {
		// Test finding trigger nodes in empty repository
		matches, err := nodeRepo.FindTriggerNodesBySourceEventAndProvider(
			ctx, "any-source", "any-event", "any-provider", models.WorkflowStatusActive)
		require.NoError(t, err)
		assert.Empty(t, matches)

		// Test getting executions by workflow/status in empty repository
		executions, err := execRepo.GetExecutionsByWorkflow(ctx, "any-workflow")
		require.NoError(t, err)
		assert.Empty(t, executions)

		executions, err = execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusRunning)
		require.NoError(t, err)
		assert.Empty(t, executions)
	})
}
