// Package events defines event types and structures for workflow lifecycle notifications.
package events

import (
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/google/uuid"
)

type EventType string

// Kafka topics.
const Topic = "operion.events"                               // Legacy topic for workflow events
const NodeActivationTopic = "operion.node.activations"       // Topic for node activations
const WorkflowExecutionTopic = "operion.workflow.executions" // Topic for workflow execution events

const EventMetadataKey = "key"
const EventTypeMetadataKey = "event_type"

const (
	// Workflow lifecycle events.
	WorkflowTriggeredEvent EventType = "workflow.triggered"
	WorkflowFinishedEvent  EventType = "workflow.finished"
	WorkflowFailedEvent    EventType = "workflow.failed"

	// Node-based workflow events.
	NodeActivationEvent        EventType = "node.activation"
	NodeCompletionEvent        EventType = "node.completion"
	NodeExecutionFinishedEvent EventType = "node.execution.finished"
	NodeExecutionFailedEvent   EventType = "node.execution.failed"

	// Workflow execution lifecycle events.
	WorkflowExecutionStartedEvent   EventType = "workflow.execution.started"
	WorkflowExecutionCompletedEvent EventType = "workflow.execution.completed"
	WorkflowExecutionFailedEvent    EventType = "workflow.execution.failed"
	WorkflowExecutionCancelledEvent EventType = "workflow.execution.cancelled"
	WorkflowExecutionTimeoutEvent   EventType = "workflow.execution.timeout"
	WorkflowExecutionPausedEvent    EventType = "workflow.execution.paused"
	WorkflowExecutionResumedEvent   EventType = "workflow.execution.resumed"
	WorkflowVariablesUpdatedEvent   EventType = "workflow.variables.updated"
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

// Node-based workflow events

type NodeActivation struct {
	BaseEvent

	ExecutionID         string `json:"execution_id"`
	NodeID              string `json:"node_id"`
	PublishedWorkflowID string `json:"published_workflow_id"`
	InputPort           string `json:"input_port"`
	InputData           any    `json:"input_data"`
	SourceNode          string `json:"source_node"`
	SourcePort          string `json:"source_port"`
}

func (n NodeActivation) GetType() EventType {
	return NodeActivationEvent
}

// NodeCompletion represents the completion of a node execution.
type NodeCompletion struct {
	BaseEvent

	ExecutionID         string            `json:"execution_id"`
	NodeID              string            `json:"node_id"`
	PublishedWorkflowID string            `json:"published_workflow_id"`
	Status              models.NodeStatus `json:"status"`
	OutputData          map[string]any    `json:"output_data"`
	ErrorMessage        string            `json:"error_message,omitempty"`
	DurationMs          int64             `json:"duration_ms"`
	CompletedAt         time.Time         `json:"completed_at"`
}

func (n NodeCompletion) GetType() EventType {
	return NodeCompletionEvent
}

type NodeExecutionFinished struct {
	BaseEvent

	ExecutionID string         `json:"execution_id"`
	NodeID      string         `json:"node_id"`
	OutputData  map[string]any `json:"output_data"`
	Duration    time.Duration  `json:"duration"`
}

func (n NodeExecutionFinished) GetType() EventType {
	return NodeExecutionFinishedEvent
}

type NodeExecutionFailed struct {
	BaseEvent

	ExecutionID string        `json:"execution_id"`
	NodeID      string        `json:"node_id"`
	Error       string        `json:"error"`
	Duration    time.Duration `json:"duration"`
}

func (n NodeExecutionFailed) GetType() EventType {
	return NodeExecutionFailedEvent
}

// Workflow execution lifecycle events

type WorkflowExecutionStarted struct {
	BaseEvent

	ExecutionID  string         `json:"execution_id"`
	WorkflowName string         `json:"workflow_name"`
	TriggerType  string         `json:"trigger_type"`
	TriggerData  map[string]any `json:"trigger_data"`
	Variables    map[string]any `json:"variables"`
	Initiator    string         `json:"initiator"`
}

func (w WorkflowExecutionStarted) GetType() EventType {
	return WorkflowExecutionStartedEvent
}

type WorkflowExecutionCompleted struct {
	BaseEvent

	ExecutionID   string         `json:"execution_id"`
	Status        string         `json:"status"`
	DurationMs    int64          `json:"duration_ms"`
	NodesExecuted int            `json:"nodes_executed"`
	FinalResults  map[string]any `json:"final_results"`
}

func (w WorkflowExecutionCompleted) GetType() EventType {
	return WorkflowExecutionCompletedEvent
}

type WorkflowExecutionFailed struct {
	BaseEvent

	ExecutionID    string         `json:"execution_id"`
	Status         string         `json:"status"`
	DurationMs     int64          `json:"duration_ms"`
	Error          WorkflowError  `json:"error"`
	NodesExecuted  int            `json:"nodes_executed"`
	PartialResults map[string]any `json:"partial_results"`
}

type WorkflowError struct {
	NodeID  string         `json:"node_id"`
	Message string         `json:"message"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details"`
}

func (w WorkflowExecutionFailed) GetType() EventType {
	return WorkflowExecutionFailedEvent
}

type WorkflowExecutionCancelled struct {
	BaseEvent

	ExecutionID   string `json:"execution_id"`
	Status        string `json:"status"`
	DurationMs    int64  `json:"duration_ms"`
	Reason        string `json:"reason"`
	CancelledBy   string `json:"cancelled_by"`
	NodesExecuted int    `json:"nodes_executed"`
}

func (w WorkflowExecutionCancelled) GetType() EventType {
	return WorkflowExecutionCancelledEvent
}

type WorkflowExecutionTimeout struct {
	BaseEvent

	ExecutionID    string         `json:"execution_id"`
	Status         string         `json:"status"`
	DurationMs     int64          `json:"duration_ms"`
	TimeoutLimitMs int64          `json:"timeout_limit_ms"`
	NodesExecuted  int            `json:"nodes_executed"`
	StuckNode      string         `json:"stuck_node"`
	PartialResults map[string]any `json:"partial_results"`
}

func (w WorkflowExecutionTimeout) GetType() EventType {
	return WorkflowExecutionTimeoutEvent
}

type WorkflowVariablesUpdated struct {
	BaseEvent

	ExecutionID      string         `json:"execution_id"`
	UpdatedVariables map[string]any `json:"updated_variables"`
	UpdatedBy        string         `json:"updated_by"`
}

func (w WorkflowVariablesUpdated) GetType() EventType {
	return WorkflowVariablesUpdatedEvent
}

type WorkflowExecutionPaused struct {
	BaseEvent

	ExecutionID  string         `json:"execution_id"`
	Status       string         `json:"status"`
	PauseReason  string         `json:"pause_reason"`
	PausedAtNode string         `json:"paused_at_node"`
	ApprovalData map[string]any `json:"approval_data"`
}

func (w WorkflowExecutionPaused) GetType() EventType {
	return WorkflowExecutionPausedEvent
}

type WorkflowExecutionResumed struct {
	BaseEvent

	ExecutionID     string `json:"execution_id"`
	Status          string `json:"status"`
	ResumedBy       string `json:"resumed_by"`
	PauseDurationMs int64  `json:"pause_duration_ms"`
	ApprovalResult  string `json:"approval_result"`
}

func (w WorkflowExecutionResumed) GetType() EventType {
	return WorkflowExecutionResumedEvent
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
