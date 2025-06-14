package domain

type ExecutionContext struct {
    WorkflowID  string
    ExecutionID string
    TriggerData map[string]interface{}
    Variables   map[string]interface{}
    StepResults map[string]interface{}
    Metadata    map[string]interface{}
}
