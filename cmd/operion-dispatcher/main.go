package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/log"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/google/uuid"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion-dispatcher",
		Usage:                 "Start the Operion dispatcher service",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			NewValidateCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dispatcher-id",
				Aliases: []string{"id"},
				Usage:   "Custom dispatcher ID (auto-generated if not provided)",
				Value:   "",
				Sources: cli.EnvVars("DISPATCHER_ID"),
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
			&cli.IntFlag{
				Name:     "webhook-port",
				Usage:    "Port for webhook HTTP server",
				Value:    8085,
				Required: false,
				Sources:  cli.EnvVars("WEBHOOK_PORT"),
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

			logger := log.WithModule("operion-dispatcher").With("dispatcher_id", dispatcherID)

			logger.Info("Initializing Operion Dispatcher", "dispatcher_id", dispatcherID)

			registry := cmd.NewRegistry(logger, command.String("plugins-path"))

			eventBus := cmd.NewEventBus(command.String("event-bus"), logger)
			defer func() {
				if err := eventBus.Close(); err != nil {
					logger.Error("Failed to close event bus", "error", err)
				}
			}()

			persistence := cmd.NewPersistence(logger, command.String("database-url"))
			defer func() {
				if err := persistence.Close(); err != nil {
					logger.Error("Failed to close persistence", "error", err)
				}
			}()

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

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
