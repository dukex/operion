package models

// TriggerMatch represents a workflow trigger that matches a source event.
// This is used by the activator to identify which workflow triggers should
// be activated when a source event is received.
type TriggerMatch struct {
	// WorkflowID identifies the workflow that contains the matching trigger
	WorkflowID string `json:"workflow_id"`

	// Trigger contains the full workflow trigger configuration that matched
	Trigger *WorkflowTrigger `json:"trigger"`
}
