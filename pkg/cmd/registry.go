// Package cmd provides common initialization functions for command-line applications.
package cmd

import (
	"log/slog"

	"github.com/dukex/operion/pkg/actions/httprequest"
	logaction "github.com/dukex/operion/pkg/actions/log"
	"github.com/dukex/operion/pkg/actions/transform"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/triggers/kafka"
	"github.com/dukex/operion/pkg/triggers/queue"
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
	reg.RegisterAction(httprequest.NewHTTPRequestActionFactory())
	reg.RegisterAction(transform.NewTransformActionFactory())
	reg.RegisterAction(logaction.NewLogActionFactory())
}

func registerNativeTriggers(reg *registry.Registry) {
	scheduleTrigger := schedule.NewScheduleTriggerFactory()
	reg.RegisterTrigger(scheduleTrigger)

	webhookTrigger := webhook.NewWebhookTriggerFactory()
	reg.RegisterTrigger(webhookTrigger)

	queueTrigger := queue.NewQueueTriggerFactory()
	reg.RegisterTrigger(queueTrigger)

	kafkaTrigger := kafka.NewKafkaTriggerFactory()
	reg.RegisterTrigger(kafkaTrigger)
}

func NewRegistry(log *slog.Logger, pluginsPath string) *registry.Registry {
	reg := registry.NewRegistry(log)

	registreActionPlugins(reg, pluginsPath)
	registreTriggerPlugins(reg, pluginsPath)

	registerNativeTriggers(reg)
	registerNativeActions(reg)

	return reg
}
