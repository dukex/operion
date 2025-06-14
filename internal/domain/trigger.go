package domain

import "context"

type TriggerCallback func(ctx context.Context, data map[string]interface{}) error

type Trigger interface {
    GetID() string
    GetType() string
    Start(ctx context.Context, callback TriggerCallback) error
    Stop(ctx context.Context) error
    Validate() error
    GetConfig() map[string]interface{}
}