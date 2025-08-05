package main

import (
	"context"
	"os"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/log"
	"github.com/google/uuid"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion-worker",
		EnableShellCompletion: true,
		Usage:                 "Start workers to execute workflows",
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

			workerID := command.String("worker-id")
			if workerID == "" {
				workerID = "worker-" + uuid.New().String()[:8]
			}

			logger := log.WithModule("operion-worker").With("workerId", workerID)

			logger.InfoContext(ctx, "Initializing Operion Worker")

			registry := cmd.NewRegistry(ctx, logger, command.String("plugins-path"))

			eventBus := cmd.NewEventBus(command.String("event-bus"), logger)
			defer func() {
				err := eventBus.Close()
				if err != nil {
					logger.ErrorContext(ctx, "Failed to close event bus", "error", err)
				}
			}()

			persistence := cmd.NewPersistence(ctx, logger, command.String("database-url"))
			defer func() {
				err := persistence.Close(ctx)
				if err != nil {
					logger.ErrorContext(ctx, "Failed to close persistence", "error", err)
				}
			}()

			worker := NewWorkerManager(
				workerID,
				persistence,
				eventBus,
				logger,
				registry,
			)

			err := worker.Start(ctx)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to start event-driven worker", "error", err)
			}

			return nil
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		panic(err)
	}
}
