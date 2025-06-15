package domain

type StepType string

type WorkflowStep struct {
	ID          string                `json:"id"`
	Action      ActionItem            `json:"action" validate:"required"`
	Conditional ConditionalExpression `json:"conditional,omitempty"`
	OnSuccess   *string               `json:"on_success,omitempty"`        // ID of the next step
	OnFailure   *string               `json:"on_failure,omitempty"`        // ID of the next step
	Enabled     bool                  `json:"enabled" validate:"required"` // Whether the step is enabled
}

type ConditionalExpression struct {
	Language   string `json:"language" validate:"required"` // "javascript", "cel", "simple"
	Expression string `json:"expression" validate:"required"`
}
