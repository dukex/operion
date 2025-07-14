package queue

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

func NewQueueTriggerFactory() protocol.TriggerFactory {
	return &QueueTriggerFactory{}
}

type QueueTriggerFactory struct{}

func (f *QueueTriggerFactory) ID() string {
	return "queue"
}

func (f *QueueTriggerFactory) Create(config map[string]interface{}, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	trigger, err := NewQueueTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue trigger: %w", err)
	}
	return trigger, nil
}