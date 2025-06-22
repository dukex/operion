package protocol

import (
	"context"
	"log/slog"
)

type TriggerCallback func(ctx context.Context, data map[string]interface{}) error

type Trigger interface {
	Start(ctx context.Context, callback TriggerCallback) error
	Stop(ctx context.Context) error
	Validate() error
}

type TriggerFactory interface {
	Create(config map[string]interface{}, logger *slog.Logger) (Trigger, error)
	ID() string
}
