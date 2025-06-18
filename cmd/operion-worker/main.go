package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion-worker",
		Usage:                 "Create and manage workflows",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Start workers to execute workflows",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "worker-id",
						Aliases: []string{"id"},
						Usage:   "Custom worker ID (auto-generated if not provided)",
						Value:   "",
					},
					&cli.BoolFlag{
						Name:  "kafka",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "rabbitmq",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "mysql",
						Value: false,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return RunWorkers(cmd)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
