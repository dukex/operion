// Package cmd provides common initialization functions for command-line applications.
package cmd

import (
	"context"
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

func registreActionPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	actionPlugins, err := reg.LoadActionPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range actionPlugins {
		reg.RegisterAction(plugin)
	}
}

func registreTriggerPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	triggerPlugins, err := reg.LoadTriggerPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range triggerPlugins {
		reg.RegisterTrigger(plugin)
	}
}

func registerNativeActions(ctx context.Context, reg *registry.Registry) {
	reg.RegisterAction(httprequest.NewActionFactory())
	reg.RegisterAction(transform.NewActionFactory())
	reg.RegisterAction(logaction.NewActionFactory())
}

func registerNativeTriggers(ctx context.Context, reg *registry.Registry) {
	reg.RegisterTrigger(schedule.NewTriggerFactory())
	reg.RegisterTrigger(webhook.NewTriggerFactory())
	reg.RegisterTrigger(queue.NewTriggerFactory())
	reg.RegisterTrigger(kafka.NewTriggerFactory())
}

func NewRegistry(ctx context.Context, log *slog.Logger, pluginsPath string) *registry.Registry {
	reg := registry.NewRegistry(log)

	registreActionPlugins(ctx, reg, pluginsPath)
	registreTriggerPlugins(ctx, reg, pluginsPath)

	registerNativeTriggers(ctx, reg)
	registerNativeActions(ctx, reg)

	return reg
}
