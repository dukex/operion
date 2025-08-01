package models

type ExecutionContext struct {
	ID          string         `json:"id"`
	WorkflowID  string         `json:"workflow_id"`
	TriggerData map[string]any `json:"trigger_data,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	StepResults map[string]any `json:"step_results,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
