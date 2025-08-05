package scheduler

import (
	"log/slog"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
)

// SchedulerSourceProviderFactory creates instances of SchedulerSourceProvider.
type SchedulerSourceProviderFactory struct{}

// NewSchedulerSourceProviderFactory creates a new factory instance.
func NewSchedulerSourceProviderFactory() *SchedulerSourceProviderFactory {
	return &SchedulerSourceProviderFactory{}
}

// Create instantiates a new centralized SchedulerSourceProvider orchestrator.
func (f *SchedulerSourceProviderFactory) Create(config map[string]any, logger *slog.Logger) (protocol.SourceProvider, error) {
	// Get persistence from config (passed by source provider manager)
	persistence, ok := config["persistence"].(persistence.Persistence)
	if !ok || persistence == nil {
		return nil, events.ErrInvalidEventData
	}

	// Create single orchestrator instance (no source-specific configuration required)
	return &SchedulerSourceProvider{
		config:      config,
		logger:      logger.With("module", "centralized_scheduler"),
		persistence: persistence,
	}, nil
}

// ID returns the unique identifier for this source provider type.
func (f *SchedulerSourceProviderFactory) ID() string {
	return "scheduler"
}

// Name returns a human-readable name for this source provider.
func (f *SchedulerSourceProviderFactory) Name() string {
	return "Centralized Scheduler"
}

// Description returns a detailed description of what this source provider does.
func (f *SchedulerSourceProviderFactory) Description() string {
	return "A centralized scheduler orchestrator that polls the database for due schedules and processes them regardless of their individual cron expressions. Schedules are created when workflows with scheduler triggers are registered."
}

// Schema returns a JSON Schema that describes the orchestrator configuration.
func (f *SchedulerSourceProviderFactory) Schema() map[string]any {
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
func (f *SchedulerSourceProviderFactory) EventTypes() []string {
	return []string{"ScheduleDue"}
}

// Ensure interface compliance.
var _ protocol.SourceProviderFactory = (*SchedulerSourceProviderFactory)(nil)
