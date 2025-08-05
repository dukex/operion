// Package log provides logging action implementation for workflow steps.
package log

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"

	"github.com/dukex/operion/pkg/template"
)

type LogAction struct {
	Message string
	Level   string
}

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

var logLevelName = map[LogLevel]string{
	Debug: "debug",
	Info:  "info",
	Warn:  "warn",
	Error: "error",
}

// NewLogAction creates a new LogAction instance with the provided configuration.
func NewLogAction(config map[string]any) *LogAction {
	message, _ := config["message"].(string)
	level, _ := config["level"].(string)

	if level == "" {
		level = logLevelName[Info]
	}

	return &LogAction{
		Message: message,
		Level:   level,
	}
}

// Execute performs the logging action, rendering the message with templating if needed.
func (a *LogAction) Execute(
	ctx context.Context,
	executionCtx models.ExecutionContext,
	logger *slog.Logger,
) (any, error) {
	logger = logger.With("action_type", "log")

	// Render the message with templating if needed
	renderedMessage, err := template.RenderWithContext(a.Message, &executionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render log message template: %w", err)
	}

	message := fmt.Sprintf("%v", renderedMessage)

	// Log the message at the specified level
	switch a.Level {
	case logLevelName[Debug]:
		logger.DebugContext(ctx, message)
	case logLevelName[Info]:
		logger.InfoContext(ctx, message)
	case logLevelName[Warn]:
		logger.WarnContext(ctx, message)
	case logLevelName[Error]:
		logger.ErrorContext(ctx, message)
	default:
		logger.InfoContext(ctx, message)
	}

	result := map[string]any{
		"message": message,
		"level":   a.Level,
	}

	return result, nil
}

var errMessageInvalid = errors.New("message is required")
var errLevelInvalid = errors.New("level is required")

// Validate checks if the LogAction configuration is valid.
func (a *LogAction) Validate(_ context.Context) error {
	if a.Message == "" {
		return errMessageInvalid
	}

	if a.Level != logLevelName[Debug] &&
		a.Level != logLevelName[Info] &&
		a.Level != logLevelName[Warn] &&
		a.Level != logLevelName[Error] {
		return fmt.Errorf("invalid log level '%s': %w", a.Level, errLevelInvalid)
	}

	return nil
}
