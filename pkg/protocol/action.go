package protocol

import (
	"context"

	"github.com/dukex/operion/pkg/models"
)

type Action interface {
	GetID() string
	GetType() string
	Execute(ctx context.Context, ectx models.ExecutionContext) (interface{}, error)
	Validate() error
	GetConfig() map[string]interface{}
}

type ActionFactory interface {
	Create(config map[string]interface{}) (Action, error)
	Type() string
}
