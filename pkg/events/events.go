// Package events defines event types and structures for workflow lifecycle notifications.
package events

import (
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/google/uuid"
)

type EventType string

const Topic = "operion.events"
const EventMetadataKey = "key"
const EventTypeMetadataKey = "event_type"

const (
	WorkflowTriggeredEvent     EventType = "workflow.triggered"
	WorkflowFinishedEvent      EventType = "workflow.finished"
	WorkflowFailedEvent        EventType = "workflow.failed"
	WorkflowStepAvailableEvent EventType = "workflow.step.available"
	WorkflowStepFinishedEvent  EventType = "workflow.step.finished"
	WorkflowStepFailedEvent    EventType = "workflow.step.failed"
)

type BaseEvent struct {
	ID         string         `json:"id"`
	Type       EventType      `json:"type"`
	Timestamp  time.Time      `json:"timestamp"`
	WorkflowID string         `json:"workflow_id"`
	WorkerID   string         `json:"worker_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type WorkflowTriggered struct {
	BaseEvent

	TriggerID   string         `json:"trigger_id"`
	TriggerData map[string]any `json:"trigger_data,omitempty"`
}

func (w WorkflowTriggered) GetType() EventType {
	return WorkflowTriggeredEvent
}

type WorkflowFinished struct {
	BaseEvent

	ExecutionID string         `json:"execution_id"`
	Result      map[string]any `json:"result,omitempty"`
	Duration    time.Duration  `json:"duration"`
}

func (w WorkflowFinished) GetType() EventType {
	return WorkflowFinishedEvent
}

type WorkflowFailed struct {
	BaseEvent

	ExecutionID string        `json:"execution_id"`
	Error       string        `json:"error"`
	Duration    time.Duration `json:"duration"`
}

func (w WorkflowFailed) GetType() EventType {
	return WorkflowFailedEvent
}

type WorkflowStepAvailable struct {
	BaseEvent

	ExecutionID      string                   `json:"execution_id"`
	StepID           string                   `json:"step_id"`
	ExecutionContext *models.ExecutionContext `json:"execution_context,omitempty"`
}

func (w WorkflowStepAvailable) GetType() EventType {
	return WorkflowStepAvailableEvent
}

type WorkflowStepFinished struct {
	BaseEvent

	ExecutionID string        `json:"execution_id"`
	StepID      string        `json:"step_id"`
	ActionID    string        `json:"action_id"`
	Result      any           `json:"result,omitempty"`
	Duration    time.Duration `json:"duration"`
}

func (w WorkflowStepFinished) GetType() EventType {
	return WorkflowStepFinishedEvent
}

type WorkflowStepFailed struct {
	BaseEvent

	ExecutionID string        `json:"execution_id"`
	StepID      string        `json:"step_id"`
	ActionID    string        `json:"action_id"`
	Error       string        `json:"error"`
	Duration    time.Duration `json:"duration"`
}

func (w WorkflowStepFailed) GetType() EventType {
	return WorkflowStepFailedEvent
}

func NewBaseEvent(eventType EventType, workflowID string) BaseEvent {
	return BaseEvent{
		ID:         uuid.New().String(),
		Type:       eventType,
		Timestamp:  time.Now().UTC(),
		WorkflowID: workflowID,
		Metadata:   make(map[string]any),
	}
}
