package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
        Name:  "operion",
        Usage: "Create and manage workflows",
		EnableShellCompletion: true,
        Commands: []*cli.Command{
			{
				Name: "workers",
				Aliases: []string{"w"},
				Usage: "Manage workflow workers",
				Commands: []*cli.Command{
					{
						Name:  "run",
						Aliases: []string{"r"},
						Usage: "Start workers to execute workflows",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "worker-id",
								Aliases: []string{"id"},
								Usage:   "Custom worker ID (auto-generated if not provided)",
								Value:   "", // Default to empty, will be auto-generated if not set
							},
							&cli.StringFlag{
								Name:    "workflows-file",
								Aliases: []string{"f"},
								Usage:   "Path to workflows file",
								Value:   "./data/workflows/index.json",
							},
							&cli.StringFlag{
								Name:    "filter",
								Aliases: []string{"t"},
								Usage:   "Comma-separated tags to filter workflows",
								Value:   "",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							workerId := cmd.String("worker-id")
							if workerId == "" {
								workerId = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
							}

							workflowsPath := cmd.String("workflows-file")
							filterTags := cmd.String("filter")

							runWorkers(workflowsPath, filterTags, workerId)
							return nil
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

