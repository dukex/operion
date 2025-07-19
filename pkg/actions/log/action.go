// Package log provides logging action implementation for workflow steps.
package log

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"

	"github.com/dukex/operion/pkg/template"
)

type LogAction struct {
	Message string
	Level   string
}

func NewLogAction(config map[string]any) *LogAction {
	message, _ := config["message"].(string)
	level, _ := config["level"].(string)

	if level == "" {
		level = "info"
	}

	return &LogAction{
		Message: message,
		Level:   level,
	}
}

func (a *LogAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (interface{}, error) {
	logger = logger.With("action_type", "log")

	// Render the message with templating if needed
	message := a.Message
	if template.NeedsTemplating(a.Message) {
		renderedMessage, err := template.RenderWithContext(a.Message, &executionCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render log message template: %w", err)
		}
		message = fmt.Sprintf("%v", renderedMessage)
	}

	// Log the message at the specified level
	switch a.Level {
	case "debug":
		logger.Debug(message)
	case "info":
		logger.Info(message)
	case "warn", "warning":
		logger.Warn(message)
	case "error":
		logger.Error(message)
	default:
		logger.Info(message)
	}

	result := map[string]any{
		"message": message,
		"level":   a.Level,
	}

	return result, nil
}
