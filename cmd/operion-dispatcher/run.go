package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/config"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func NewRunCommand() *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Start the Operion receiver-based dispatcher service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dispatcher-id",
				Aliases: []string{"id"},
				Usage:   "Custom dispatcher ID (auto-generated if not provided)",
				Value:   "",
			},
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "Database connection URL for persistence",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "event-bus",
				Usage:    "Event bus type (kafka, rabbitmq, etc.)",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "webhook-port",
				Usage:    "Port for webhook HTTP server",
				Value:    8085,
				Required: false,
			},
			&cli.StringFlag{
				Name:     "receiver-config",
				Usage:    "Path to receiver configuration file",
				Value:    "./configs/receivers.yaml",
				Required: false,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			tracerProvider, err := trc.InitTracer(ctx, "operion-trigger")
			if err != nil {
				return fmt.Errorf("failed to initialize tracer: %w", err)
			}
			defer func() {
				if err := tracerProvider.Shutdown(ctx); err != nil {
					slog.Error("Failed to shutdown tracer provider", "error", err)
				}
			}()

			dispatcherID := command.String("dispatcher-id")
			if dispatcherID == "" {
				dispatcherID = fmt.Sprintf("dispatcher-%s", uuid.New().String()[:8])
			}

			logger := slog.With(
				"module", "operion-dispatcher",
				"dispatcher_id", dispatcherID,
			)

			logger.Info("Initializing Operion Dispatcher", "dispatcher_id", dispatcherID)

			eventBus := cmd.NewEventBus(command.String("event-bus"), logger)
			defer func() {
				if err := eventBus.Close(); err != nil {
					logger.Error("Failed to close event bus", "error", err)
				}
			}()

			persistence := cmd.NewPersistence(command.String("database-url"))
			defer func() {
				if err := persistence.Close(); err != nil {
					logger.Error("Failed to close persistence", "error", err)
				}
			}()

			// Load receiver configuration
			receiverConfig, err := config.LoadReceiverConfig(command.String("receiver-config"))
			if err != nil {
				logger.Warn("Failed to load receiver config, using default", "error", err)
				receiverConfig = config.LoadReceiverConfigOrDefault(command.String("receiver-config"))
			}

			// Validate receiver configuration
			if err := config.ValidateReceiverConfig(receiverConfig); err != nil {
				return fmt.Errorf("invalid receiver configuration: %w", err)
			}

			logger.Info("Loaded receiver configuration", 
				"sources_count", len(receiverConfig.Sources),
				"trigger_topic", receiverConfig.TriggerTopic)

			// Create and start receiver manager
			receiverManager := NewReceiverManager(
				dispatcherID,
				persistence,
				eventBus,
				logger,
				command.Int("webhook-port"),
			)

			if err := receiverManager.Configure(receiverConfig); err != nil {
				return fmt.Errorf("failed to configure receiver manager: %w", err)
			}

			receiverManager.Start(ctx)

			return nil
		},
	}
}
