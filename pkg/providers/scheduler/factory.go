package scheduler

import (
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

// SchedulerProviderFactory creates instances of SchedulerProvider.
type SchedulerProviderFactory struct{}

// NewSchedulerProviderFactory creates a new factory instance.
func NewSchedulerProviderFactory() *SchedulerProviderFactory {
	return &SchedulerProviderFactory{}
}

// Create instantiates a new centralized SchedulerProvider orchestrator.
func (f *SchedulerProviderFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Provider, error) {
	// Create single orchestrator instance (no source-specific configuration required)
	// Persistence will be initialized during the Initialize lifecycle method
	return &SchedulerProvider{
		config: config,
		logger: logger.With("module", "centralized_scheduler"),
	}, nil
}

// ID returns the unique identifier for this source provider type.
func (f *SchedulerProviderFactory) ID() string {
	return "scheduler"
}

// Name returns a human-readable name for this source provider.
func (f *SchedulerProviderFactory) Name() string {
	return "Centralized Scheduler"
}

// Description returns a detailed description of what this source provider does.
func (f *SchedulerProviderFactory) Description() string {
	return "A centralized scheduler orchestrator that polls the database for due schedules and processes them regardless of their individual cron expressions. Schedules are created when workflows with scheduler triggers are registered."
}

// Schema returns a JSON Schema that describes the orchestrator configuration.
func (f *SchedulerProviderFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"polling_interval": map[string]any{
				"type":        "string",
				"description": "How often the orchestrator polls for due schedules (default: 1 minute)",
				"examples":    []string{"1m", "30s", "2m"},
				"default":     "1m",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
		"description":          "Centralized scheduler orchestrator configuration. Individual schedule cron expressions are defined in workflow triggers, not here.",
	}
}

// EventTypes returns a list of event types that this source provider can emit.
func (f *SchedulerProviderFactory) EventTypes() []string {
	return []string{"ScheduleDue"}
}

// Ensure interface compliance.
var _ protocol.ProviderFactory = (*SchedulerProviderFactory)(nil)
