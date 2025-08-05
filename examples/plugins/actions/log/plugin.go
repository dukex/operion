package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

type LogAction struct {
	config map[string]any
}

func (l *LogAction) Execute(ctx context.Context, ectx models.ExecutionContext, logger *slog.Logger) (any, error) {
	log.Default().Println("Executing custom log action")
	return nil, nil
}

type ActionFactory struct{}

func (l ActionFactory) Schema() map[string]any {
	return map[string]any{}
}

func (l ActionFactory) ID() string {
	return "custom-log"
}

func (l ActionFactory) Name() string {
	return "Custom Log"
}

func (l ActionFactory) Description() string {
	return "Logs a message to the console or a file based on the configuration provided."
}

func (l ActionFactory) Create(config map[string]any) (protocol.Action, error) {
	return &LogAction{
		config: config,
	}, nil
}

var _ protocol.ActionFactory = ActionFactory{}

var Action = ActionFactory{}
