package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

type LogAction struct {
	config map[string]interface{}
}

func (l *LogAction) Execute(ctx context.Context, ectx models.ExecutionContext, logger *slog.Logger) (interface{}, error) {
	log.Default().Println("Executing custom log action")
	return nil, nil
}

type LogActionFactory struct{}

func (l LogActionFactory) Schema() map[string]any {
	return map[string]any{}
}

func (l LogActionFactory) ID() string {
	return "custom-log"
}

func (l LogActionFactory) Name() string {
	return "Custom Log"
}

func (l LogActionFactory) Description() string {
	return "Logs a message to the console or a file based on the configuration provided."
}

func (l LogActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	return &LogAction{
		config: config,
	}, nil
}

var _ protocol.ActionFactory = LogActionFactory{}

var Action = LogActionFactory{}
