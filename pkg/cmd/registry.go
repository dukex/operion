package cmd

import (
	"log/slog"

	"github.com/dukex/operion/pkg/actions/http_request"
	log_action "github.com/dukex/operion/pkg/actions/log"
	"github.com/dukex/operion/pkg/actions/transform"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/triggers/schedule"
	"github.com/dukex/operion/pkg/triggers/webhook"
)

func registreActionPlugins(reg *registry.Registry, pluginsPath string) {
	actionPlugins, err := reg.LoadActionPlugins(pluginsPath)
	if err != nil {
		panic(err)

	}
	for _, plugin := range actionPlugins {
		reg.RegisterAction(plugin)
	}
}

func registreTriggerPlugins(reg *registry.Registry, pluginsPath string) {
	triggerPlugins, err := reg.LoadTriggerPlugins(pluginsPath)
	if err != nil {
		panic(err)
	}
	for _, plugin := range triggerPlugins {
		reg.RegisterTrigger(plugin)
	}
}

func registerNativeActions(reg *registry.Registry) {
	reg.RegisterAction(http_request.NewHTTPRequestActionFactory())
	reg.RegisterAction(transform.NewTransformActionFactory())
	reg.RegisterAction(log_action.NewLogActionFactory())
}

func registerNativeTriggers(reg *registry.Registry) {
	scheduleTrigger := schedule.NewScheduleTriggerFactory()
	reg.RegisterTrigger(scheduleTrigger)

	webhookTrigger := webhook.NewWebhookTriggerFactory()
	reg.RegisterTrigger(webhookTrigger)
}

func NewRegistry(log *slog.Logger, pluginsPath string) *registry.Registry {
	reg := registry.NewRegistry(log)

	registreActionPlugins(reg, pluginsPath)
	registreTriggerPlugins(reg, pluginsPath)

	registerNativeTriggers(reg)
	registerNativeActions(reg)

	return reg
}
