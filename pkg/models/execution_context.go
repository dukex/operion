package models

type ExecutionContext struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	TriggerData map[string]interface{} `json:"trigger_data,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	StepResults map[string]interface{} `json:"step_results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
