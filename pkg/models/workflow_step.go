package models

type StepType string

type WorkflowStep struct {
	ID            string         `json:"id"`
	ActionID      string         `json:"action_id" validate:"required"`
	Configuration map[string]any `json:"configuration"`
	UID           string         `json:"uid" validate:"required,lowercase,alphanum"`
	Name          string         `json:"name" validate:"required"`
	OnSuccess     *string        `json:"on_success,omitempty"`
	OnFailure     *string        `json:"on_failure,omitempty"`
	Enabled       bool           `json:"enabled" validate:"required"`
}
