package log_action

import (
	"context"
	"fmt"
	"log"

	"github.com/dukex/operion/pkg/models"
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

// GetSchema returns the JSON Schema for Log Action configuration
func GetLogActionSchema() *models.RegisteredComponent {
	return &models.RegisteredComponent{
		Type:        "log",
		Name:        "Log Message",
		Description: "Log a message with configurable level",
		Schema: &models.JSONSchema{
			Type:        "object",
			Title:       "Log Action Configuration",
			Description: "Configuration for logging messages",
			Properties: map[string]*models.Property{
				"message": {
					Type:        "string",
					Description: "Message to log",
				},
				"level": {
					Type:        "string",
					Description: "Log level",
					Enum:        []interface{}{"debug", "info", "warn", "error"},
					Default:     "info",
				},
			},
			Required: []string{"message"},
		},
	}
}

func (a *LogAction) Execute(ctx context.Context, executionCtx models.ExecutionContext) (interface{}, error) {
	logMessage := fmt.Sprintf("[%s] %s", a.Level, a.Message)
	log.Printf("LogAction '%s': %s", a.ID, logMessage)

	result := map[string]interface{}{
		"logged_message": logMessage,
		"level":          a.Level,
	}

	return result, nil
}
