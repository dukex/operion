package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constructor Tests

func TestNewSourceEvent_WithValidData(t *testing.T) {
	eventData := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	event := NewSourceEvent("source-123", "scheduler", "ScheduleDue", eventData)

	assert.Equal(t, "source-123", event.SourceID)
	assert.Equal(t, "scheduler", event.ProviderID)
	assert.Equal(t, "ScheduleDue", event.EventType)
	assert.Equal(t, eventData, event.EventData)
}

func TestNewSourceEvent_WithNilEventData(t *testing.T) {
	event := NewSourceEvent("source-123", "scheduler", "ScheduleDue", nil)

	assert.Equal(t, "source-123", event.SourceID)
	assert.Equal(t, "scheduler", event.ProviderID)
	assert.Equal(t, "ScheduleDue", event.EventType)
	assert.NotNil(t, event.EventData)
	assert.Empty(t, event.EventData)
}

func TestNewSourceEvent_EmptyEventData(t *testing.T) {
	eventData := map[string]any{}
	event := NewSourceEvent("source-123", "webhook", "RequestReceived", eventData)

	assert.Equal(t, "source-123", event.SourceID)
	assert.Equal(t, "webhook", event.ProviderID)
	assert.Equal(t, "RequestReceived", event.EventType)
	assert.Equal(t, eventData, event.EventData)
	assert.Empty(t, event.EventData)
}

// Validation Tests

func TestSourceEvent_Validate_Success(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	err := event.Validate()
	assert.NoError(t, err)
}

func TestSourceEvent_Validate_MissingSourceID(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	err := event.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source_id is required")
}

func TestSourceEvent_Validate_MissingProviderID(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	err := event.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider_id is required")
}

func TestSourceEvent_Validate_MissingEventType(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "",
		EventData:  map[string]any{},
	}

	err := event.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event_type is required")
}

// Trigger Matching Tests

func TestSourceEvent_MatchesTrigger_ExactSourceIDMatch(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	matches := event.MatchesTrigger("source-123", "")
	assert.True(t, matches)
}

func TestSourceEvent_MatchesTrigger_SourceIDMismatch(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	matches := event.MatchesTrigger("source-456", "")
	assert.False(t, matches)
}

func TestSourceEvent_MatchesTrigger_WithEventTypeFilter(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	matches := event.MatchesTrigger("source-123", "ScheduleDue")
	assert.True(t, matches)
}

func TestSourceEvent_MatchesTrigger_EventTypeMismatch(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	matches := event.MatchesTrigger("source-123", "ScheduleOverdue")
	assert.False(t, matches)
}

func TestSourceEvent_MatchesTrigger_EmptyEventTypeFilter(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "RequestReceived",
		EventData:  map[string]any{},
	}

	// Empty event type filter should match any event type for the same source
	matches := event.MatchesTrigger("source-123", "")
	assert.True(t, matches)
}

// Data Extraction Tests

func TestSourceEvent_GetEventDataString_ValidString(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "RequestReceived",
		EventData: map[string]any{
			"message": "Hello World",
			"user":    "john_doe",
		},
	}

	value, ok := event.GetEventDataString("message")
	assert.True(t, ok)
	assert.Equal(t, "Hello World", value)

	value, ok = event.GetEventDataString("user")
	assert.True(t, ok)
	assert.Equal(t, "john_doe", value)
}

func TestSourceEvent_GetEventDataString_NonExistentKey(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "RequestReceived",
		EventData:  map[string]any{},
	}

	value, ok := event.GetEventDataString("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestSourceEvent_GetEventDataString_WrongType(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "RequestReceived",
		EventData: map[string]any{
			"count": 42,
			"flag":  true,
		},
	}

	value, ok := event.GetEventDataString("count")
	assert.False(t, ok)
	assert.Equal(t, "", value)

	value, ok = event.GetEventDataString("flag")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestSourceEvent_GetEventDataInt_ValidInt(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData: map[string]any{
			"count":     42,
			"attempts":  3,
			"timestamp": 1609459200,
		},
	}

	value, ok := event.GetEventDataInt("count")
	assert.True(t, ok)
	assert.Equal(t, 42, value)

	value, ok = event.GetEventDataInt("attempts")
	assert.True(t, ok)
	assert.Equal(t, 3, value)
}

func TestSourceEvent_GetEventDataInt_ValidFloat64(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData: map[string]any{
			"duration": 42.7, // float64
			"timeout":  15.0, // float64
		},
	}

	value, ok := event.GetEventDataInt("duration")
	assert.True(t, ok)
	assert.Equal(t, 42, value)

	value, ok = event.GetEventDataInt("timeout")
	assert.True(t, ok)
	assert.Equal(t, 15, value)
}

func TestSourceEvent_GetEventDataInt_ValidFloat32(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData: map[string]any{
			"ratio": float32(3.14),
			"score": float32(100.0),
		},
	}

	value, ok := event.GetEventDataInt("ratio")
	assert.True(t, ok)
	assert.Equal(t, 3, value)

	value, ok = event.GetEventDataInt("score")
	assert.True(t, ok)
	assert.Equal(t, 100, value)
}

func TestSourceEvent_GetEventDataInt_NonExistentKey(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{},
	}

	value, ok := event.GetEventDataInt("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, 0, value)
}

func TestSourceEvent_GetEventDataInt_WrongType(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "RequestReceived",
		EventData: map[string]any{
			"message": "hello",
			"flag":    true,
		},
	}

	value, ok := event.GetEventDataInt("message")
	assert.False(t, ok)
	assert.Equal(t, 0, value)

	value, ok = event.GetEventDataInt("flag")
	assert.False(t, ok)
	assert.Equal(t, 0, value)
}

func TestSourceEvent_GetEventDataMap_ValidMap(t *testing.T) {
	nestedData := map[string]any{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "UserCreated",
		EventData: map[string]any{
			"user":     nestedData,
			"metadata": map[string]any{"version": "1.0"},
		},
	}

	value, ok := event.GetEventDataMap("user")
	assert.True(t, ok)
	assert.Equal(t, nestedData, value)

	// Verify nested data access
	require.Contains(t, value, "name")
	assert.Equal(t, "John Doe", value["name"])

	value, ok = event.GetEventDataMap("metadata")
	assert.True(t, ok)
	require.Contains(t, value, "version")
	assert.Equal(t, "1.0", value["version"])
}

func TestSourceEvent_GetEventDataMap_NonExistentKey(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "UserCreated",
		EventData:  map[string]any{},
	}

	value, ok := event.GetEventDataMap("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, value)
}

func TestSourceEvent_GetEventDataMap_WrongType(t *testing.T) {
	event := &SourceEvent{
		SourceID:   "source-123",
		ProviderID: "webhook",
		EventType:  "UserCreated",
		EventData: map[string]any{
			"message": "hello",
			"count":   42,
			"flag":    true,
		},
	}

	value, ok := event.GetEventDataMap("message")
	assert.False(t, ok)
	assert.Nil(t, value)

	value, ok = event.GetEventDataMap("count")
	assert.False(t, ok)
	assert.Nil(t, value)

	value, ok = event.GetEventDataMap("flag")
	assert.False(t, ok)
	assert.Nil(t, value)
}
