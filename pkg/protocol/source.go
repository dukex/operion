package protocol

import (
	"context"
	"log/slog"
)

// SourceEventCallback is called when a source provider emits an event.
// The callback should publish the event to the event bus for activator consumption.
type SourceEventCallback func(ctx context.Context, sourceID, providerID, eventType string, eventData map[string]any) error

// Provider represents a running instance of a source provider that can emit events.
// Source providers are long-running processes that monitor external systems and emit
// events when specific conditions are met (e.g., scheduled time, webhook received, etc.).
type Provider interface {
	// Start begins the source provider's operation, monitoring for events to emit.
	// The callback function should be called whenever an event occurs.
	Start(ctx context.Context, callback SourceEventCallback) error

	// Stop gracefully shuts down the source provider.
	Stop(ctx context.Context) error

	// Validate checks if the source provider configuration is valid.
	Validate() error
}

// ProviderFactory creates instances of Provider with specific configurations.
// This interface is implemented by source provider plugins to enable dynamic loading.
type ProviderFactory interface {
	// Create instantiates a new Provider with the given configuration.
	// The config map contains source-specific settings defined by the plugin's schema.
	Create(config map[string]any, logger *slog.Logger) (Provider, error)

	// ID returns the unique identifier for this source provider type.
	// This ID is used in the Source.ProviderID field to reference this provider.
	ID() string

	// Name returns a human-readable name for this source provider.
	Name() string

	// Description returns a detailed description of what this source provider does.
	Description() string

	// Schema returns a JSON Schema that describes the configuration structure
	// required by this source provider. This is used for validation and UI generation.
	Schema() map[string]any

	// EventTypes returns a list of event types that this source provider can emit.
	// This helps users understand what events they can configure triggers for.
	EventTypes() []string
}
