package protocol

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
)

// ProviderLifecycle defines the lifecycle management interface for source providers.
// This interface enables providers to handle their own initialization, configuration,
// and preparation phases before starting.
type ProviderLifecycle interface {
	// Initialize sets up the provider with required dependencies.
	// Called once when the source manager starts the provider.
	Initialize(ctx context.Context, deps Dependencies) error

	// Configure configures the provider based on current workflow definitions.
	// Called after Initialize() and whenever workflows change.
	// Returns a map of triggerID -> sourceID for workflows that were configured.
	Configure(workflows []*models.Workflow) (map[string]string, error)

	// Prepare performs final preparation before starting the provider.
	// Called after Configure(), just before Start().
	Prepare(ctx context.Context) error
}

// Dependencies contains the common dependencies that providers need.
type Dependencies struct {
	Logger *slog.Logger
	// Note: No shared persistence - providers manage their own data
}
