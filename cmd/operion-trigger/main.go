package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion-trigger",
		Usage:                 "Manage workflow triggers and publish trigger events",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Start trigger listeners and event publishers",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "trigger-id",
						Aliases: []string{"id"},
						Usage:   "Custom trigger service ID (auto-generated if not provided)",
						Value:   "",
					},
					&cli.BoolFlag{
						Name:  "kafka",
						Usage: "Use Kafka as event bus",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "rabbitmq",
						Usage: "Use RabbitMQ as event bus",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "data-path",
						Usage: "Path to data directory",
						Value: "./data",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return RunTriggerService(cmd)
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all available triggers",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "data-path",
						Usage: "Path to workflow data directory",
						Value: "./data/workflows",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return ListTriggers(cmd)
				},
			},
			{
				Name:    "validate",
				Aliases: []string{"v"},
				Usage:   "Validate trigger configurations",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "data-path",
						Usage: "Path to workflow data directory",
						Value: "./data/workflows",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return ValidateTriggers(cmd)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
