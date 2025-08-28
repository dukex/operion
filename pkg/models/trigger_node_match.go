package models

// TriggerNodeMatch represents a trigger node that matches a source event.
// This is used by the activator to identify which trigger nodes should
// be activated when a source event is received.
type TriggerNodeMatch struct {
	// WorkflowID identifies the workflow that contains the matching trigger node
	WorkflowID string `json:"workflow_id"`

	// TriggerNode contains the full workflow node configuration that matched
	TriggerNode *WorkflowNode `json:"trigger_node"`
}
