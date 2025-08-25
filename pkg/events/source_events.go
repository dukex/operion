package events

import "errors"

// ErrInvalidEventData is returned when source event data cannot be parsed or is invalid.
var ErrInvalidEventData = errors.New("invalid event data")

// SourceEvent represents an event emitted by external source providers that can trigger workflows.
// These events are consumed by the operion-activator service to determine which workflows
// should be triggered based on source ID and event type matching.
//
// Source events are published by source provider plugins and consumed by the activator
// to create WorkflowTriggered events for matching workflow triggers.
type SourceEvent struct {
	// SourceID uniquely identifies the source instance that generated this event.
	// This corresponds to the SourceID field in WorkflowTrigger configurations.
	SourceID string `json:"source_id" validate:"required"`

	// ProviderID identifies the type of provider that generated this event.
	// This corresponds to the ProviderID in the Source model and the plugin ID.
	// Examples: "scheduler", "webhook", "github", "gitlab", "slack", etc.
	ProviderID string `json:"provider_id" validate:"required"`

	// EventType specifies the specific type of event that occurred within the provider.
	// This allows fine-grained filtering of which events should trigger workflows.
	// The available event types are defined by each source provider plugin.
	// Examples: "ScheduleDue", "PushReceived", "IssueOpened", "MessageReceived"
	EventType string `json:"event_type" validate:"required"`

	// EventData contains provider-specific data associated with the event.
	// The structure of this data depends on the ProviderID and EventType.
	// This data will be passed to triggered workflows as trigger data.
	// Each source provider plugin defines its own event data schema.
	EventData map[string]any `json:"event_data"`
}

// NewSourceEvent creates a new SourceEvent with the provided parameters.
func NewSourceEvent(sourceID, providerID, eventType string, eventData map[string]any) *SourceEvent {
	if eventData == nil {
		eventData = make(map[string]any)
	}

	return &SourceEvent{
		SourceID:   sourceID,
		ProviderID: providerID,
		EventType:  eventType,
		EventData:  eventData,
	}
}

// GetEventDataString safely extracts a string value from the event data.
// Returns the string value and true if the key exists and is a string, otherwise empty string and false.
func (se *SourceEvent) GetEventDataString(key string) (string, bool) {
	value, exists := se.EventData[key]
	if !exists {
		return "", false
	}

	strValue, ok := value.(string)

	return strValue, ok
}

// GetEventDataInt safely extracts an integer value from the event data.
// Returns the int value and true if the key exists and is numeric, otherwise 0 and false.
func (se *SourceEvent) GetEventDataInt(key string) (int, bool) {
	value, exists := se.EventData[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	default:
		return 0, false
	}
}

// GetEventDataMap safely extracts a nested map from the event data.
// Returns the map and true if the key exists and is a map, otherwise nil and false.
func (se *SourceEvent) GetEventDataMap(key string) (map[string]any, bool) {
	value, exists := se.EventData[key]
	if !exists {
		return nil, false
	}

	mapValue, ok := value.(map[string]any)

	return mapValue, ok
}

// Validate performs basic validation on the source event structure.
func (se *SourceEvent) Validate() error {
	if se.SourceID == "" {
		return errors.New("source_id is required")
	}

	if se.ProviderID == "" {
		return errors.New("provider_id is required")
	}

	if se.EventType == "" {
		return errors.New("event_type is required")
	}

	return nil
}
