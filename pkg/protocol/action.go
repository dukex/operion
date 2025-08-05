// Package protocol defines the interfaces and contracts for pluggable actions and triggers.
package protocol

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
)

type Action interface {
	Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (any, error)
	Validate(ctx context.Context) error
}

type ActionFactory interface {
	Create(ctx context.Context, config map[string]any) (Action, error)
	ID() string
	Name() string
	Description() string
	Schema() map[string]any
}
