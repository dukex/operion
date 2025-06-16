package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "operion",
		Usage:                 "Create and manage workflows",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "Manage workflow workers",
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
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runWorkers(cmd)
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
