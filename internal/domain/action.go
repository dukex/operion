package domain

import "context"

type Action interface {
	GetID() string
	GetType() string
	Execute(ctx context.Context, input ExecutionContext) (ExecutionContext, error)
	Validate() error
	GetConfig() map[string]interface{}
}

type ActionItem struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Name  string                 `json:"name"`
	Description string                 `json:"description"`
	Configuration  map[string]interface{} `json:"configuration"`
}