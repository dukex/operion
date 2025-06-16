package log_action

import (
	"context"
	"fmt"
	"log"

	"github.com/dukex/operion/internal/domain"
)

type LogAction struct {
	ID      string
	Message string
	Level   string
}

func NewLogAction(config map[string]interface{}) (*LogAction, error) {
	id, _ := config["id"].(string)
	message, _ := config["message"].(string)
	level, _ := config["level"].(string)

	return &LogAction{
		ID:      id,
		Message: message,
		Level:   level,
	}, nil
}

func (a *LogAction) GetID() string   { return a.ID }
func (a *LogAction) GetType() string { return "log" }
func (a *LogAction) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":      a.ID,
		"message": a.Message,
		"level":   a.Level,
	}
}
func (a *LogAction) Validate() error { return nil }

func (a *LogAction) Execute(ctx context.Context, executionCtx domain.ExecutionContext) (interface{}, error) {
	logMessage := fmt.Sprintf("[%s] %s", a.Level, a.Message)
	log.Printf("LogAction '%s': %s", a.ID, logMessage)

	result := map[string]interface{}{
		"logged_message": logMessage,
		"level":          a.Level,
	}

	return result, nil
}
