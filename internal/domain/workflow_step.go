package domain

type StepType string

type WorkflowStep struct {
	ID          string                `json:"id"`
	Action      ActionItem            `json:"action" validate:"required"`
	Name        string                `json:"name" validate:"required"`
	Conditional ConditionalExpression `json:"conditional,omitempty"`
	OnSuccess   *string               `json:"on_success,omitempty"`
	OnFailure   *string               `json:"on_failure,omitempty"`
	Enabled     bool                  `json:"enabled" validate:"required"`
}

type ConditionalExpression struct {
	Language   string `json:"language" validate:"required"` // "javascript", "cel", "simple"
	Expression string `json:"expression" validate:"required"`
}
