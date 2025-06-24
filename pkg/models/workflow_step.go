package models

type StepType string

type WorkflowStep struct {
	ID            string                 `json:"id"`
	ActionID      string                 `json:"action_id" validate:"required"`
	Configuration map[string]interface{} `json:"configuration"`
	UID           string                 `json:"uid" validate:"required,lowercase,alphanum"`
	Name          string                 `json:"name" validate:"required"`
	Conditional   ConditionalExpression  `json:"conditional,omitempty"`
	OnSuccess     *string                `json:"on_success,omitempty"`
	OnFailure     *string                `json:"on_failure,omitempty"`
	Enabled       bool                   `json:"enabled" validate:"required"`
}

type ConditionalExpression struct {
	Language   string `json:"language" validate:"required"` // "javascript", "cel", "simple"
	Expression string `json:"expression" validate:"required"`
}
