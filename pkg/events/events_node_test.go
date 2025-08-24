package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NodeActivation Event Tests

func TestNodeActivation_GetType(t *testing.T) {
	event := NodeActivation{}
	assert.Equal(t, NodeActivationEvent, event.GetType())
}

func TestNodeActivation_JSONSerialization(t *testing.T) {
	original := &NodeActivation{
		BaseEvent:           NewBaseEvent(NodeActivationEvent, "wf-123"),
		ExecutionID:         "exec-456",
		NodeID:              "http-request-node-1",
		PublishedWorkflowID: "wf-published-789",
		InputPort:           "default",
		InputData: map[string]any{
			"url":    "https://api.example.com/users",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer token-123",
				"Content-Type":  "application/json",
			},
		},
		SourceNode: "conditional-node-0",
		SourcePort: "true",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"node.activation"`)
	assert.Contains(t, string(jsonData), `"execution_id":"exec-456"`)
	assert.Contains(t, string(jsonData), `"node_id":"http-request-node-1"`)
	assert.Contains(t, string(jsonData), `"published_workflow_id":"wf-published-789"`)

	// Deserialize from JSON
	var deserialized NodeActivation

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Type, deserialized.Type)
	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.NodeID, deserialized.NodeID)
	assert.Equal(t, original.PublishedWorkflowID, deserialized.PublishedWorkflowID)
	assert.Equal(t, original.InputPort, deserialized.InputPort)
	assert.Equal(t, original.SourceNode, deserialized.SourceNode)
	assert.Equal(t, original.SourcePort, deserialized.SourcePort)

	// Verify input data
	originalInputData := original.InputData.(map[string]any)
	deserializedInputData := deserialized.InputData.(map[string]any)
	assert.Equal(t, originalInputData["url"], deserializedInputData["url"])
	assert.Equal(t, originalInputData["method"], deserializedInputData["method"])
	headers := deserializedInputData["headers"].(map[string]any)
	assert.Equal(t, "Bearer token-123", headers["Authorization"])
	assert.Equal(t, "application/json", headers["Content-Type"])
}

// NodeExecutionFinished Event Tests

func TestNodeExecutionFinished_GetType(t *testing.T) {
	event := NodeExecutionFinished{}
	assert.Equal(t, NodeExecutionFinishedEvent, event.GetType())
}

func TestNodeExecutionFinished_JSONSerialization(t *testing.T) {
	original := &NodeExecutionFinished{
		BaseEvent:   NewBaseEvent(NodeExecutionFinishedEvent, "wf-123"),
		ExecutionID: "exec-456",
		NodeID:      "transform-node-2",
		OutputData: map[string]any{
			"result": map[string]any{
				"processed": true,
				"data": map[string]any{
					"users":        []map[string]any{{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}},
					"total":        2,
					"processed_at": "2024-01-15T10:30:00Z",
				},
			},
		},
		Duration: 1500 * time.Millisecond,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"node.execution.finished"`)
	assert.Contains(t, string(jsonData), `"execution_id":"exec-456"`)
	assert.Contains(t, string(jsonData), `"node_id":"transform-node-2"`)

	// Deserialize from JSON
	var deserialized NodeExecutionFinished

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Type, deserialized.Type)
	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.NodeID, deserialized.NodeID)
	assert.Equal(t, original.Duration, deserialized.Duration)

	// Verify output data
	result := deserialized.OutputData["result"].(map[string]any)
	assert.Equal(t, true, result["processed"])

	data := result["data"].(map[string]any)
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, "2024-01-15T10:30:00Z", data["processed_at"])

	users := data["users"].([]any)
	assert.Len(t, users, 2)
	user1 := users[0].(map[string]any)
	assert.Equal(t, float64(1), user1["id"])
	assert.Equal(t, "John", user1["name"])
}

// NodeExecutionFailed Event Tests

func TestNodeExecutionFailed_GetType(t *testing.T) {
	event := NodeExecutionFailed{}
	assert.Equal(t, NodeExecutionFailedEvent, event.GetType())
}

func TestNodeExecutionFailed_JSONSerialization(t *testing.T) {
	original := &NodeExecutionFailed{
		BaseEvent:   NewBaseEvent(NodeExecutionFailedEvent, "wf-123"),
		ExecutionID: "exec-456",
		NodeID:      "http-request-node-1",
		Error:       "HTTP request failed: 500 Internal Server Error - Database connection timeout",
		Duration:    2300 * time.Millisecond,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"node.execution.failed"`)
	assert.Contains(t, string(jsonData), `"error":"HTTP request failed: 500 Internal Server Error - Database connection timeout"`)

	// Deserialize from JSON
	var deserialized NodeExecutionFailed

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.Type, deserialized.Type)
	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.NodeID, deserialized.NodeID)
	assert.Equal(t, original.Error, deserialized.Error)
	assert.Equal(t, original.Duration, deserialized.Duration)
}

// WorkflowExecutionStarted Event Tests

func TestWorkflowExecutionStarted_GetType(t *testing.T) {
	event := WorkflowExecutionStarted{}
	assert.Equal(t, WorkflowExecutionStartedEvent, event.GetType())
}

func TestWorkflowExecutionStarted_JSONSerialization(t *testing.T) {
	original := &WorkflowExecutionStarted{
		BaseEvent:    NewBaseEvent(WorkflowExecutionStartedEvent, "wf-123"),
		ExecutionID:  "exec-7f3d2a1b-4c5e-6789-ab01-23456789cdef",
		WorkflowName: "User Registration Flow",
		TriggerType:  "webhook",
		TriggerData: map[string]any{
			"webhook": map[string]any{
				"user_email":          "user@example.com",
				"registration_source": "mobile_app",
				"timestamp":           "2024-01-15T10:30:00Z",
			},
		},
		Variables: map[string]any{
			"environment": "production",
			"retry_limit": 3,
			"api_timeout": 30,
		},
		Initiator: "webhook-trigger-node-1",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"workflow.execution.started"`)
	assert.Contains(t, string(jsonData), `"workflow_name":"User Registration Flow"`)
	assert.Contains(t, string(jsonData), `"trigger_type":"webhook"`)

	// Deserialize from JSON
	var deserialized WorkflowExecutionStarted

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.WorkflowName, deserialized.WorkflowName)
	assert.Equal(t, original.TriggerType, deserialized.TriggerType)
	assert.Equal(t, original.Initiator, deserialized.Initiator)

	// Verify trigger data
	webhookData := deserialized.TriggerData["webhook"].(map[string]any)
	assert.Equal(t, "user@example.com", webhookData["user_email"])
	assert.Equal(t, "mobile_app", webhookData["registration_source"])

	// Verify variables
	assert.Equal(t, "production", deserialized.Variables["environment"])
	assert.Equal(t, float64(3), deserialized.Variables["retry_limit"])
	assert.Equal(t, float64(30), deserialized.Variables["api_timeout"])
}

// WorkflowExecutionCompleted Event Tests

func TestWorkflowExecutionCompleted_GetType(t *testing.T) {
	event := WorkflowExecutionCompleted{}
	assert.Equal(t, WorkflowExecutionCompletedEvent, event.GetType())
}

func TestWorkflowExecutionCompleted_JSONSerialization(t *testing.T) {
	original := &WorkflowExecutionCompleted{
		BaseEvent:     NewBaseEvent(WorkflowExecutionCompletedEvent, "wf-123"),
		ExecutionID:   "exec-7f3d2a1b-4c5e-6789-ab01-23456789cdef",
		Status:        "success",
		DurationMs:    8450,
		NodesExecuted: 5,
		FinalResults: map[string]any{
			"user_created":       true,
			"email_sent":         true,
			"user_id":            "usr-987654",
			"verification_token": "token-abc123",
			"workflow_metadata": map[string]any{
				"nodes_executed": 5,
				"execution_time": "8.45s",
				"success_rate":   1.0,
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"workflow.execution.completed"`)
	assert.Contains(t, string(jsonData), `"status":"success"`)
	assert.Contains(t, string(jsonData), `"duration_ms":8450`)

	// Deserialize from JSON
	var deserialized WorkflowExecutionCompleted

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.Status, deserialized.Status)
	assert.Equal(t, original.DurationMs, deserialized.DurationMs)
	assert.Equal(t, original.NodesExecuted, deserialized.NodesExecuted)

	// Verify final results
	assert.Equal(t, true, deserialized.FinalResults["user_created"])
	assert.Equal(t, true, deserialized.FinalResults["email_sent"])
	assert.Equal(t, "usr-987654", deserialized.FinalResults["user_id"])

	metadata := deserialized.FinalResults["workflow_metadata"].(map[string]any)
	assert.Equal(t, float64(5), metadata["nodes_executed"])
	assert.Equal(t, "8.45s", metadata["execution_time"])
	assert.Equal(t, 1.0, metadata["success_rate"])
}

// WorkflowExecutionFailed Event Tests

func TestWorkflowExecutionFailed_GetType(t *testing.T) {
	event := WorkflowExecutionFailed{}
	assert.Equal(t, WorkflowExecutionFailedEvent, event.GetType())
}

func TestWorkflowExecutionFailed_JSONSerialization(t *testing.T) {
	original := &WorkflowExecutionFailed{
		BaseEvent:   NewBaseEvent(WorkflowExecutionFailedEvent, "wf-123"),
		ExecutionID: "exec-8g4e3b2c-5d6f-7890-bc12-34567890deff",
		Status:      "failed",
		DurationMs:  3200,
		Error: WorkflowError{
			NodeID:  "http-request-node-3",
			Message: "HTTP request failed: 500 Internal Server Error",
			Code:    "HTTP_ERROR",
			Details: map[string]any{
				"url":           "https://api.example.com/users",
				"status_code":   500,
				"response_body": "Database connection timeout",
				"retry_count":   3,
				"timeout":       true,
			},
		},
		NodesExecuted: 3,
		PartialResults: map[string]any{
			"validation_passed":    true,
			"user_data_prepared":   true,
			"email_prepared":       false,
			"last_successful_node": "validate-input-node-1",
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"workflow.execution.failed"`)
	assert.Contains(t, string(jsonData), `"status":"failed"`)
	assert.Contains(t, string(jsonData), `"duration_ms":3200`)

	// Deserialize from JSON
	var deserialized WorkflowExecutionFailed

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.Status, deserialized.Status)
	assert.Equal(t, original.DurationMs, deserialized.DurationMs)
	assert.Equal(t, original.NodesExecuted, deserialized.NodesExecuted)

	// Verify error details
	assert.Equal(t, original.Error.NodeID, deserialized.Error.NodeID)
	assert.Equal(t, original.Error.Message, deserialized.Error.Message)
	assert.Equal(t, original.Error.Code, deserialized.Error.Code)

	errorDetails := deserialized.Error.Details
	assert.Equal(t, "https://api.example.com/users", errorDetails["url"])
	assert.Equal(t, float64(500), errorDetails["status_code"])
	assert.Equal(t, "Database connection timeout", errorDetails["response_body"])
	assert.Equal(t, float64(3), errorDetails["retry_count"])
	assert.Equal(t, true, errorDetails["timeout"])

	// Verify partial results
	assert.Equal(t, true, deserialized.PartialResults["validation_passed"])
	assert.Equal(t, true, deserialized.PartialResults["user_data_prepared"])
	assert.Equal(t, false, deserialized.PartialResults["email_prepared"])
	assert.Equal(t, "validate-input-node-1", deserialized.PartialResults["last_successful_node"])
}

// WorkflowVariablesUpdated Event Tests

func TestWorkflowVariablesUpdated_GetType(t *testing.T) {
	event := WorkflowVariablesUpdated{}
	assert.Equal(t, WorkflowVariablesUpdatedEvent, event.GetType())
}

func TestWorkflowVariablesUpdated_JSONSerialization(t *testing.T) {
	original := &WorkflowVariablesUpdated{
		BaseEvent:   NewBaseEvent(WorkflowVariablesUpdatedEvent, "wf-123"),
		ExecutionID: "exec-2j7h6e5f-8g9i-0123-ef45-678901234567",
		UpdatedVariables: map[string]any{
			"current_step":      "email_verification",
			"retry_count":       2,
			"user_preference":   "sms_notifications",
			"verification_code": "123456",
			"last_updated":      "2024-01-15T10:50:15Z",
			"metadata": map[string]any{
				"source":  "conditional_node",
				"trigger": "user_action",
			},
		},
		UpdatedBy: "conditional-node-7",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"type":"workflow.variables.updated"`)
	assert.Contains(t, string(jsonData), `"updated_by":"conditional-node-7"`)

	// Deserialize from JSON
	var deserialized WorkflowVariablesUpdated

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.ExecutionID, deserialized.ExecutionID)
	assert.Equal(t, original.UpdatedBy, deserialized.UpdatedBy)

	// Verify updated variables
	assert.Equal(t, "email_verification", deserialized.UpdatedVariables["current_step"])
	assert.Equal(t, float64(2), deserialized.UpdatedVariables["retry_count"])
	assert.Equal(t, "sms_notifications", deserialized.UpdatedVariables["user_preference"])
	assert.Equal(t, "123456", deserialized.UpdatedVariables["verification_code"])

	metadata := deserialized.UpdatedVariables["metadata"].(map[string]any)
	assert.Equal(t, "conditional_node", metadata["source"])
	assert.Equal(t, "user_action", metadata["trigger"])
}

// Event Topic Constants Tests

func TestEventTopicConstants(t *testing.T) {
	// Test that the topic constants are defined correctly
	assert.Equal(t, "operion.events", Topic)
	assert.Equal(t, "operion.node.activations", NodeActivationTopic)
	assert.Equal(t, "operion.workflow.executions", WorkflowExecutionTopic)
	assert.Equal(t, "key", EventMetadataKey)
	assert.Equal(t, "event_type", EventTypeMetadataKey)
}

// Event Type Constants Tests

func TestEventTypeConstants(t *testing.T) {
	// Test node-based event types
	assert.Equal(t, EventType("node.activation"), NodeActivationEvent)
	assert.Equal(t, EventType("node.execution.finished"), NodeExecutionFinishedEvent)
	assert.Equal(t, EventType("node.execution.failed"), NodeExecutionFailedEvent)

	// Test workflow execution event types
	assert.Equal(t, EventType("workflow.execution.started"), WorkflowExecutionStartedEvent)
	assert.Equal(t, EventType("workflow.execution.completed"), WorkflowExecutionCompletedEvent)
	assert.Equal(t, EventType("workflow.execution.failed"), WorkflowExecutionFailedEvent)
	assert.Equal(t, EventType("workflow.execution.cancelled"), WorkflowExecutionCancelledEvent)
	assert.Equal(t, EventType("workflow.execution.timeout"), WorkflowExecutionTimeoutEvent)
	assert.Equal(t, EventType("workflow.execution.paused"), WorkflowExecutionPausedEvent)
	assert.Equal(t, EventType("workflow.execution.resumed"), WorkflowExecutionResumedEvent)
	assert.Equal(t, EventType("workflow.variables.updated"), WorkflowVariablesUpdatedEvent)
}

// BaseEvent Tests with Node Events

func TestNewBaseEvent_NodeEvents(t *testing.T) {
	workflowID := "wf-test-123"

	testCases := []struct {
		name      string
		eventType EventType
	}{
		{"node activation", NodeActivationEvent},
		{"node execution finished", NodeExecutionFinishedEvent},
		{"node execution failed", NodeExecutionFailedEvent},
		{"workflow execution started", WorkflowExecutionStartedEvent},
		{"workflow execution completed", WorkflowExecutionCompletedEvent},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseEvent := NewBaseEvent(tc.eventType, workflowID)

			assert.NotEmpty(t, baseEvent.ID)
			assert.Equal(t, tc.eventType, baseEvent.Type)
			assert.Equal(t, workflowID, baseEvent.WorkflowID)
			assert.WithinDuration(t, time.Now().UTC(), baseEvent.Timestamp, time.Second)
			assert.NotNil(t, baseEvent.Metadata)
			assert.Empty(t, baseEvent.WorkerID) // Should be empty by default
		})
	}
}
