package application

import (
	"github.com/dukex/operion/internal/domain"
	file_write_action "github.com/dukex/operion/internal/actions/file_write"
	http_action "github.com/dukex/operion/internal/actions/http_request"
	log_action "github.com/dukex/operion/internal/actions/log"
	transform_action "github.com/dukex/operion/internal/actions/transform"
	triggers "github.com/dukex/operion/internal/triggers/schedule"
)

// SetupTriggerRegistry creates and configures a trigger registry with default triggers
func SetupTriggerRegistry() *TriggerRegistry {
	registry := NewTriggerRegistry()
	
	// Register schedule trigger
	registry.Register("schedule", func(config map[string]interface{}) (domain.Trigger, error) {
		return triggers.NewScheduleTrigger(config)
	})
	
	return registry
}

// SetupActionRegistry creates and configures an action registry with default actions
func SetupActionRegistry() *ActionRegistry {
	registry := NewActionRegistry()
	
	// Register HTTP request action
	registry.Register("http_request", func(config map[string]interface{}) (domain.Action, error) {
		return http_action.NewHTTPRequestAction(config)
	})
	
	// Register log action
	registry.Register("log", func(config map[string]interface{}) (domain.Action, error) {
		return log_action.NewLogAction(config)
	})
	
	// Register transform action
	registry.Register("transform", func(config map[string]interface{}) (domain.Action, error) {
		return transform_action.NewTransformAction(config)
	})
	
	// Register file write action
	registry.Register("file_write", func(config map[string]interface{}) (domain.Action, error) {
		return file_write_action.NewFileWriteAction(config)
	})
	
	return registry
}