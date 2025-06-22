package models

// type TriggerCallback func(ctx context.Context, data map[string]interface{}) error

// type Trigger interface {
// 	GetID() string
// 	GetType() string
// 	Start(ctx context.Context, callback TriggerCallback) error
// 	Stop(ctx context.Context) error
// 	Validate() error
// 	GetConfig() map[string]interface{}
// }

type WorkflowTrigger struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name" validate:"required,min=3"`
	Description   string                 `json:"description" validate:"required"`
	TriggerID     string                 `json:"trigger_id"`
	Configuration map[string]interface{} `json:"configuration"`
}

type Trigger struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"`
}
