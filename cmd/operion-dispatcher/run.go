package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/cmd"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func NewRunCommand() *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Start the Operion dispatcher service",
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
			&cli.StringFlag{
				Name:     "plugins-path",
				Usage:    "Path to the directory containing action plugins",
				Value:    "./plugins",
				Required: false,
			},
			&cli.IntFlag{
				Name:     "webhook-port",
				Usage:    "Port for webhook HTTP server",
				Value:    8085,
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

			registry := cmd.NewRegistry(logger, command.String("plugins-path"))

			eventBus := cmd.NewEventBus(command.String("event-bus"), logger)
			defer eventBus.Close()

			persistence := cmd.NewPersistence(command.String("database-url"))
			defer persistence.Close()

			NewDispatcherManager(
				dispatcherID,
				persistence,
				eventBus,
				logger,
				registry,
				command.Int("webhook-port"),
			).Start(ctx)

			return nil
		},
	}
}
