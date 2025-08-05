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
		Name:                  "operion-activator",
		Usage:                 "Start the Operion activator service",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			NewValidateCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "activator-id",
				Aliases: []string{"id"},
				Usage:   "Custom activator ID (auto-generated if not provided)",
				Value:   "",
				Sources: cli.EnvVars("ACTIVATOR_ID"),
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
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			log.Setup(command.String("log-level"))

			tracerProvider, err := trc.InitTracer(ctx, "operion-activator")
			if err != nil {
				return fmt.Errorf("failed to initialize tracer: %w", err)
			}
			defer func() {
				if err := tracerProvider.Shutdown(ctx); err != nil {
					slog.Error("Failed to shutdown tracer provider", "error", err)
				}
			}()

			activatorID := command.String("activator-id")
			if activatorID == "" {
				activatorID = fmt.Sprintf("activator-%s", uuid.New().String()[:8])
			}

			logger := log.WithModule("operion-activator").With("activator_id", activatorID)

			logger.Info("Initializing Operion Activator", "activator_id", activatorID)

			eventBus := cmd.NewEventBus(command.String("event-bus"), logger)
			defer func() {
				if err := eventBus.Close(); err != nil {
					logger.Error("Failed to close workflow event bus", "error", err)
				}
			}()

			sourceEventBus := cmd.NewSourceEventBus(logger)
			defer func() {
				if err := sourceEventBus.Close(); err != nil {
					logger.Error("Failed to close source event bus", "error", err)
				}
			}()

			persistence := cmd.NewPersistence(logger, command.String("database-url"))
			defer func() {
				if err := persistence.Close(); err != nil {
					logger.Error("Failed to close persistence", "error", err)
				}
			}()

			activator := NewActivator(
				activatorID,
				persistence,
				eventBus,
				sourceEventBus,
				logger,
			)

			activator.Start(ctx)

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
