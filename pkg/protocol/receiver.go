package protocol

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
)

// ReceiverConfig defines configuration for receivers
type ReceiverConfig struct {
	Sources      []SourceConfig      `json:"sources"`
	TriggerTopic string              `json:"trigger_topic"`
	Transformers []TransformerConfig `json:"transformers"`
}

// SourceConfig defines configuration for a data source
type SourceConfig struct {
	Type          string                 `json:"type"`          // "kafka", "webhook", "schedule"
	Name          string                 `json:"name"`          // identifier
	Configuration map[string]interface{} `json:"configuration"` // source-specific config
	Schema        map[string]interface{} `json:"schema"`        // expected payload schema
}

// TransformerConfig defines data transformation configuration
type TransformerConfig struct {
	Type   string                 `json:"type"`   // "json_path", "jira_transformer", etc.
	Name   string                 `json:"name"`   // identifier
	Config map[string]interface{} `json:"config"` // transformation-specific config
}

// Receiver interface for listening to external data sources
type Receiver interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Configure(config ReceiverConfig) error
	Validate() error
}

// ReceiverFactory interface for creating receivers
type ReceiverFactory interface {
	Create(config ReceiverConfig, eventBus eventbus.EventBus, logger *slog.Logger) (Receiver, error)
	Type() string
	Name() string
	Description() string
}