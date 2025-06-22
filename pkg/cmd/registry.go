package cmd

import (
	"log/slog"

	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/triggers/schedule"
)

func NewRegistry(log *slog.Logger, pluginsPath string) *registry.Registry {
	reg := registry.NewRegistry(log)

	actionPlugins, err := reg.LoadActionPlugins(pluginsPath)
	if err != nil {
		panic(err)

	}
	for _, plugin := range actionPlugins {
		reg.RegisterAction(plugin)
	}

	scheduleTrigger := schedule.NewScheduleTriggerFactory()
	reg.RegisterTrigger(scheduleTrigger)

	triggerPlugins, err := reg.LoadTriggerPlugins(pluginsPath)
	if err != nil {
		panic(err)
	}
	for _, plugin := range triggerPlugins {
		reg.RegisterTrigger(plugin)
	}

	return reg
}
