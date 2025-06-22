package models

type ExecutionContext struct {
	ID          string
	WorkflowID  string
	TriggerData map[string]interface{}
	Variables   map[string]interface{}
	StepResults map[string]interface{}
	Metadata    map[string]interface{}
}

func (ex *ExecutionContext) WithLogger() *ExecutionContext {
	return &ExecutionContext{
		WorkflowID:  ex.WorkflowID,
		ID:          ex.ID,
		TriggerData: ex.TriggerData,
		Variables:   ex.Variables,
		StepResults: ex.StepResults,
		Metadata:    ex.Metadata,
	}
}
