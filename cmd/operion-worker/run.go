package main

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/log"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func NewRunCommand() *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Start workers to execute workflows",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "worker-id",
				Aliases: []string{"id"},
				Usage:   "Custom worker ID (auto-generated if not provided)",
				Value:   "",
				Sources: cli.EnvVars("WORKER_ID"),
			},
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "Database connection URL for persistence",
				Required: true,
				Sources:  cli.EnvVars("DATABASE_URL"),
			},
			&cli.StringFlag{
				Name:     "event-bus",
				Usage:    "Event bus type (kafka, rabbitmq, etc.)",
				Required: true,
				Sources:  cli.EnvVars("EVENT_BUS_TYPE"),
			},
			&cli.StringFlag{
				Name:     "plugins-path",
				Usage:    "Path to the directory containing action plugins",
				Value:    "./plugins",
				Required: false,
				Sources:  cli.EnvVars("PLUGINS_PATH"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			log.Setup(command.String("log-level"))

			// tracerProvider, err := trc.InitTracer(ctx, "operion-worker")
			// if err != nil {
			// 	return fmt.Errorf("failed to initialize tracer: %w", err)
			// }
			// defer func() {
			// 	if err := tracerProvider.Shutdown(ctx); err != nil {
			// 		slog.Error("Failed to shutdown tracer provider", "error", err)
			// 	}
			// }()

			workerID := command.String("worker-id")
			if workerID == "" {
				workerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
			}

			logger := log.WithModule("operion-worker").With("workerId", workerID)

			logger.Info("Initializing Operion Worker")

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

			worker := NewWorkerManager(
				workerID,
				persistence,
				eventBus,
				logger,
				registry,
			)

			if err := worker.Start(ctx); err != nil {
				logger.Error("Failed to start event-driven worker", "error", err)
			}

			return nil
		},
	}
}
