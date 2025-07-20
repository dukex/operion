package main

import (
	"context"
	"log/slog"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/urfave/cli/v3"
)

func RunAPICommand() *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Start api",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Port to run the API server on",
				Value:   9091,
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
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			// tracerProvider, err := trc.InitTracer(ctx, "operion-worker")
			// if err != nil {
			// 	return fmt.Errorf("failed to initialize tracer: %w", err)
			// }
			// defer func() {
			// 	if err := tracerProvider.Shutdown(ctx); err != nil {
			// 		slog.Error("Failed to shutdown tracer provider", "error", err)
			// 	}
			// }()

			port := command.Int("port")

			logger := slog.With(
				"module", "api",
			)

			logger.Info("Initializing Operion API")

			registry := cmd.NewRegistry(logger, command.String("plugins-path"))
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

			api := NewAPI(
				persistence,
				eventBus,
				logger,
				registry,
			)

			if err := api.Start(port); err != nil {
				logger.Error("Failed to start event-driven worker", "error", err)
			}
			return nil
		},
	}
}
