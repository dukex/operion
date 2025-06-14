package domain

type StepType string

type WorkflowStep struct {
    ID          string
    Type        StepType
    Action      Action
    Conditional Conditional
    OnSuccess   *string // ID of the next step
    OnFailure   *string // ID of the next step
    Enabled     bool
}