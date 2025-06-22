package schedule

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

func NewScheduleTriggerFactory() protocol.TriggerFactory {
	return &ScheduleTriggerFactory{}
}

type ScheduleTriggerFactory struct{}

func (f *ScheduleTriggerFactory) ID() string {
	return "schedule"
}

func (f *ScheduleTriggerFactory) Create(config map[string]interface{}, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	trigger, err := NewScheduleTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule trigger: %w", err)
	}
	return trigger, nil
}
