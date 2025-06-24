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
	// GetConfig() map[string]interface{}
}

type ActionFactory interface {
	Create(config map[string]interface{}) (Action, error)
	ID() string
}
