package events

const (
	// TriggerDetectedEvent is published when a receiver detects a trigger event
	TriggerDetectedEvent EventType = "trigger.detected"
)

// TriggerEvent represents a standardized trigger event from receivers
type TriggerEvent struct {
	BaseEvent
	TriggerType  string                 `json:"trigger_type"`  // "kafka", "webhook", "schedule"
	Source       string                 `json:"source"`        // source identifier
	TriggerData  map[string]interface{} `json:"trigger_data"`  // transformed payload
	OriginalData map[string]interface{} `json:"original_data"` // raw payload
}

func (t TriggerEvent) GetType() EventType {
	return TriggerDetectedEvent
}

// NewTriggerEvent creates a new trigger event
func NewTriggerEvent(triggerType, source string, triggerData, originalData map[string]interface{}) TriggerEvent {
	return TriggerEvent{
		BaseEvent:    NewBaseEvent(TriggerDetectedEvent, ""),
		TriggerType:  triggerType,
		Source:       source,
		TriggerData:  triggerData,
		OriginalData: originalData,
	}
}

// NewTriggerEventWithID creates a new trigger event with a custom event ID
func NewTriggerEventWithID(eventID, triggerType, source string, triggerData, originalData map[string]interface{}) TriggerEvent {
	baseEvent := NewBaseEvent(TriggerDetectedEvent, "")
	baseEvent.ID = eventID
	
	return TriggerEvent{
		BaseEvent:    baseEvent,
		TriggerType:  triggerType,
		Source:       source,
		TriggerData:  triggerData,
		OriginalData: originalData,
	}
}