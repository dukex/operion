package main

import (
	"context"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

type LogAction struct {
	config map[string]interface{}
}

// Execute implements protocol.Action.
func (l *LogAction) Execute(ctx context.Context, ectx models.ExecutionContext) (interface{}, error) {
	panic("unimplemented")
}

// GetConfig implements protocol.Action.
func (l *LogAction) GetConfig() map[string]interface{} {
	panic("unimplemented")
}

// GetID implements protocol.Action.
func (l *LogAction) GetID() string {
	panic("unimplemented")
}

// GetType implements protocol.Action.
func (l *LogAction) GetType() string {
	panic("unimplemented")
}

// Validate implements protocol.Action.
func (l *LogAction) Validate() error {
	panic("unimplemented")
}

type LogActionFactory struct{}

// Type implements protocol.ActionFactory.
func (l LogActionFactory) Type() string {
	return "log"
}

func (l LogActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	return &LogAction{
		config: config,
	}, nil
}

var _ protocol.ActionFactory = LogActionFactory{}

var Action = LogActionFactory{}
