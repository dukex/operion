package events

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	WorkflowTriggeredEvent    EventType = "workflow.triggered"
	WorkflowStartedEvent      EventType = "workflow.started"
	WorkflowFinishedEvent     EventType = "workflow.finished"
	WorkflowFailedEvent       EventType = "workflow.failed"
	WorkflowStepStartedEvent  EventType = "workflow.step.started"
	WorkflowStepFinishedEvent EventType = "workflow.step.finished"
	WorkflowStepFailedEvent   EventType = "workflow.step.failed"
)

type BaseEvent struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	WorkflowID string                 `json:"workflow_id"`
	WorkerID   string                 `json:"worker_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type WorkflowTriggered struct {
	BaseEvent
	TriggerID   string                 `json:"trigger_id"`
	TriggerType string                 `json:"trigger_type"`
	TriggerData map[string]interface{} `json:"trigger_data,omitempty"`
}

type WorkflowStarted struct {
	BaseEvent
	ExecutionID string `json:"execution_id"`
}

type WorkflowFinished struct {
	BaseEvent
	ExecutionID string                 `json:"execution_id"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

type WorkflowFailed struct {
	BaseEvent
	ExecutionID string        `json:"execution_id"`
	Error       string        `json:"error"`
	Duration    time.Duration `json:"duration"`
}

type WorkflowStepStarted struct {
	BaseEvent
	ExecutionID string `json:"execution_id"`
	StepID      string `json:"step_id"`
	StepName    string `json:"step_name"`
	ActionType  string `json:"action_type"`
}

type WorkflowStepFinished struct {
	BaseEvent
	ExecutionID string                 `json:"execution_id"`
	StepID      string                 `json:"step_id"`
	StepName    string                 `json:"step_name"`
	ActionType  string                 `json:"action_type"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

type WorkflowStepFailed struct {
	BaseEvent
	ExecutionID string        `json:"execution_id"`
	StepID      string        `json:"step_id"`
	StepName    string        `json:"step_name"`
	ActionType  string        `json:"action_type"`
	Error       string        `json:"error"`
	Duration    time.Duration `json:"duration"`
}

func NewBaseEvent(eventType EventType, workflowID string) BaseEvent {
	return BaseEvent{
		ID:         uuid.New().String(),
		Type:       eventType,
		Timestamp:  time.Now(),
		WorkflowID: workflowID,
		Metadata:   make(map[string]interface{}),
	}
}
