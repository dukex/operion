package registry

import (
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/triggers/kafka"
	"github.com/dukex/operion/pkg/triggers/schedule"
)

// RegisterAllComponents registers all available actions and triggers with their schemas
// This function should be updated when new actions or triggers are added
func RegisterAllComponents(registry *Registry) {
	registerActions(registry)
	registerTriggers(registry)
}

// registerActions registers all available actions
// TODO: Add action registrations here when actions are migrated to pkg structure
func registerActions(registry *Registry) {
	// Example when actions are available:
	// registry.RegisterAction(
	// 	http_action.GetHTTPRequestActionSchema(),
	// 	func(config map[string]interface{}) (models.Action, error) {
	// 		return http_action.NewHTTPRequestAction(config)
	// 	},
	// )
}

// registerTriggers registers all available triggers  
func registerTriggers(registry *Registry) {
	// Register Schedule Trigger
	registry.RegisterTrigger(
		schedule.GetScheduleTriggerSchema(),
		func(config map[string]interface{}) (models.Trigger, error) {
			trigger, err := schedule.NewScheduleTrigger(config)
			if err != nil {
				return nil, err
			}
			return trigger, nil
		},
	)
	
	// Register Kafka Trigger
	registry.RegisterTrigger(
		kafka.GetKafkaTriggerSchema(),
		func(config map[string]interface{}) (models.Trigger, error) {
			trigger, err := kafka.NewKafkaTrigger(config)
			if err != nil {
				return nil, err
			}
			return trigger, nil
		},
	)
}

// GetDefaultRegistry returns a registry with all components registered
func GetDefaultRegistry() *Registry {
	registry := NewRegistry()
	RegisterAllComponents(registry)
	return registry
}
