// Package cmd provides common initialization functions for command-line applications.
package cmd

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/actions/httprequest"
	logaction "github.com/dukex/operion/pkg/actions/log"
	"github.com/dukex/operion/pkg/actions/transform"
	kafkaProvider "github.com/dukex/operion/pkg/providers/kafka"
	"github.com/dukex/operion/pkg/providers/scheduler"
	webhookSource "github.com/dukex/operion/pkg/providers/webhook"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/triggers/kafka"
	"github.com/dukex/operion/pkg/triggers/queue"
	"github.com/dukex/operion/pkg/triggers/schedule"
	"github.com/dukex/operion/pkg/triggers/webhook"
)

func registerActionPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	actionPlugins, err := reg.LoadActionPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range actionPlugins {
		reg.RegisterAction(plugin)
	}
}

func registerTriggerPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	triggerPlugins, err := reg.LoadTriggerPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range triggerPlugins {
		reg.RegisterTrigger(plugin)
	}
}

func registerProviderPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	sourceProviderPlugins, err := reg.LoadProviderPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range sourceProviderPlugins {
		reg.RegisterProvider(plugin)
	}
}

func registerNativeActions(reg *registry.Registry) {
	reg.RegisterAction(httprequest.NewActionFactory())
	reg.RegisterAction(transform.NewActionFactory())
	reg.RegisterAction(logaction.NewActionFactory())
}

func registerNativeTriggers(reg *registry.Registry) {
	reg.RegisterTrigger(schedule.NewTriggerFactory())
	reg.RegisterTrigger(webhook.NewTriggerFactory())
	reg.RegisterTrigger(queue.NewTriggerFactory())
	reg.RegisterTrigger(kafka.NewTriggerFactory())
}

func registerNativeProviders(reg *registry.Registry) {
	schedulerProvider := scheduler.NewSchedulerProviderFactory()
	reg.RegisterProvider(schedulerProvider)

	webhookProvider := webhookSource.NewWebhookProviderFactory()
	reg.RegisterProvider(webhookProvider)

	kafkaSourceProvider := kafkaProvider.NewKafkaProviderFactory()
	reg.RegisterProvider(kafkaSourceProvider)
}

func NewRegistry(ctx context.Context, log *slog.Logger, pluginsPath string) *registry.Registry {
	reg := registry.NewRegistry(log)

	registerActionPlugins(ctx, reg, pluginsPath)
	registerTriggerPlugins(ctx, reg, pluginsPath)
	registerProviderPlugins(ctx, reg, pluginsPath)

	registerNativeTriggers(reg)
	registerNativeActions(reg)
	registerNativeProviders(reg)

	return reg
}
