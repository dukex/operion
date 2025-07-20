package main

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/urfave/cli/v3"
)

func NewListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all workflow triggers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "Database connection URL for persistence",
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
			persistence := cmd.NewPersistence(command.String("database-url"))
			defer func() {
				if err := persistence.Close(); err != nil {
					return
				}
			}()

			workflowRepository := workflow.NewRepository(persistence)

			workflows, err := workflowRepository.FetchAll()
			if err != nil {
				return fmt.Errorf("failed to fetch workflows: %w", err)
			}

			fmt.Println("Available Triggers:")
			fmt.Println("==================")

			totalTriggers := 0
			for _, workflow := range workflows {
				fmt.Printf("\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)
				fmt.Printf("Status: %s\n", workflow.Status)
				fmt.Printf("Workflow Triggers:\n")

				for _, trigger := range workflow.WorkflowTriggers {
					fmt.Printf("  - ID: %s\n", trigger.ID)
					fmt.Printf("    Trigger ID: %s\n", trigger.TriggerID)
					fmt.Printf("    Config: %v\n", trigger.Configuration)
					totalTriggers++
				}
			}

			fmt.Printf("\nTotal triggers: %d\n", totalTriggers)
			return nil
		},
	}
}
