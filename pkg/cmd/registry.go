// Package cmd provides common initialization functions for command-line applications.
package cmd

import (
	"context"
	"log/slog"

	kafkaProvider "github.com/dukex/operion/pkg/providers/kafka"
	"github.com/dukex/operion/pkg/providers/scheduler"
	webhookSource "github.com/dukex/operion/pkg/providers/webhook"
	"github.com/dukex/operion/pkg/registry"
)

func registerProviderPlugins(ctx context.Context, reg *registry.Registry, pluginsPath string) {
	sourceProviderPlugins, err := reg.LoadProviderPlugins(ctx, pluginsPath)
	if err != nil {
		panic(err)
	}

	for _, plugin := range sourceProviderPlugins {
		reg.RegisterProvider(plugin)
	}
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

	registerProviderPlugins(ctx, reg, pluginsPath)
	registerNativeProviders(reg)

	// Register built-in node factories for node-based workflow execution
	reg.RegisterDefaultNodes()

	return reg
}