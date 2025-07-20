// Package protocol defines the interfaces and contracts for pluggable actions and triggers.
package protocol

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
)

type Action interface {
	// GetID() string
	// GetType() string
	Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (interface{}, error)
	// Validate() error
	// GetConfig() map[string]any
}

type ActionFactory interface {
	Create(config map[string]any) (Action, error)
	ID() string
	Name() string
	Description() string
	Schema() map[string]any
}
