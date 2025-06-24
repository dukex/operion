package log_action

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

func NewLogActionFactory() *LogActionFactory {
	return &LogActionFactory{}
}

type LogActionFactory struct {
}

func (*LogActionFactory) ID() string {
	return "log"
}

func (f *LogActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	if config == nil {
		config = map[string]interface{}{}
	}

	return NewLogAction(config), nil
}

type LogAction struct {
}

func NewLogAction(config map[string]interface{}) *LogAction {
	return &LogAction{}
}

func (a *LogAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (interface{}, error) {
	logger = logger.With("action_type", "log")

	logger.Info("Executing log action")
	logger.Info("Log message", "message", executionCtx.StepResults)

	result := map[string]interface{}{}

	return result, nil
}
