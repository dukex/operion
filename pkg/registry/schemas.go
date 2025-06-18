package registry

// RegisterAllComponents registers all available actions and triggers with their schemas
// This function should be updated when new actions or triggers are added
func RegisterAllComponents(registry *Registry) {
	// TODO: Uncomment and update imports when actions and triggers are migrated
	// registerActions(registry)
	// registerTriggers(registry)
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
// TODO: Add trigger registrations here when triggers are migrated to pkg structure
func registerTriggers(registry *Registry) {
	// Example when triggers are available:
	// registry.RegisterTrigger(
	// 	schedule_trigger.GetScheduleTriggerSchema(),
	// 	func(config map[string]interface{}) (models.Trigger, error) {
	// 		return schedule_trigger.NewScheduleTrigger(config)
	// 	},
	// )
}

// GetDefaultRegistry returns a registry with all components registered
func GetDefaultRegistry() *Registry {
	registry := NewRegistry()
	RegisterAllComponents(registry)
	return registry
}
