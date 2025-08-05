package protocol

import (
	"context"
	"log/slog"
)

type TriggerCallback func(ctx context.Context, data map[string]any) error

type Trigger interface {
	Start(ctx context.Context, callback TriggerCallback) error
	Stop(ctx context.Context) error
	Validate(ctx context.Context) error
}

type TriggerFactory interface {
	Create(ctx context.Context, config map[string]any, logger *slog.Logger) (Trigger, error)
	ID() string
	Name() string
	Description() string
	Schema() map[string]any
}
