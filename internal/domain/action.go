package domain

import "context"

type Action interface {
    GetID() string
    GetType() string
    Execute(ctx context.Context, input ExecutionContext) (ExecutionContext, error)
    Validate() error
    GetConfig() map[string]interface{}
}