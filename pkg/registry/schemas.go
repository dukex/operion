package registry

// func RegisterAllComponents() {
// 	registerActions(DefaultRegistry)
// 	registerTriggers(DefaultRegistry)
// }

// func registerActions(registry *Registry) {
// 	registry.RegisterAction(
// 		http_request_action.GetHTTPRequestActionSchema(),
// 		func(config map[string]interface{}) (models.Action, error) {
// 			return http_request_action.NewHTTPRequestAction(config)
// 		},
// 	)

// 	registry.RegisterAction(
// 		transform_action.GetTransformActionSchema(),
// 		func(config map[string]interface{}) (models.Action, error) {
// 			return transform_action.NewTransformAction(config)
// 		},
// 	)

// 	registry.RegisterAction(
// 		file_write_action.GetFileWriteActionSchema(),
// 		func(config map[string]interface{}) (models.Action, error) {
// 			return file_write_action.NewFileWriteAction(config)
// 		},
// 	)
// }

// func registerTriggers(registry *Registry) {
// 	registry.RegisterTrigger(
// 		schedule.GetScheduleTriggerSchema(),
// 		func(config map[string]interface{}) (models.Trigger, error) {
// 			trigger, err := schedule.NewScheduleTrigger(config)
// 			if err != nil {
// 				return nil, err
// 			}
// 			return trigger, nil
// 		},
// 	)

// 	registry.RegisterTrigger(
// 		kafka.GetKafkaTriggerSchema(),
// 		func(config map[string]interface{}) (models.Trigger, error) {
// 			trigger, err := kafka.NewKafkaTrigger(config)
// 			if err != nil {
// 				return nil, err
// 			}
// 			return trigger, nil
// 		},
// 	)
// }
