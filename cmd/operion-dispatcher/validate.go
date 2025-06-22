package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/urfave/cli/v3"
)

func NewValidateCommand() *cli.Command {
	return &cli.Command{
		Name:    "validate",
		Aliases: []string{"v"},
		Usage:   "Validate trigger configurations",
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
			logger := slog.With(
				"module", "operion-dispatcher",
				"action", "validate",
			)

			registry := cmd.NewRegistry(logger, command.String("plugins-path"))

			persistence := cmd.NewPersistence(command.String("database-url"))
			defer persistence.Close()

			workflowRepository := workflow.NewRepository(persistence)

			workflows, err := workflowRepository.FetchAll()
			if err != nil {
				return fmt.Errorf("failed to fetch workflows: %w", err)
			}

			logger.Info("Validating triggers", "workflows", len(workflows))

			fmt.Println("Trigger Validation Results:")
			fmt.Println("===========================")

			totalTriggers := 0
			validTriggers := 0
			invalidTriggers := 0

			for _, workflow := range workflows {
				fmt.Printf("\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)
				if len(workflow.WorkflowTriggers) == 0 {
					fmt.Printf("    ❌ INVALID: No trigger found for this workflow.\n")
					invalidTriggers++
					continue
				}

				for _, workflowTrigger := range workflow.WorkflowTriggers {
					totalTriggers++
					fmt.Printf("  WorkflowTrigger: %s (%s)\n", workflowTrigger.ID, workflowTrigger.TriggerID)

					config := make(map[string]interface{})
					maps.Copy(config, workflowTrigger.Configuration)
					config["workflow_id"] = workflow.ID
					config["trigger_id"] = workflowTrigger.ID
					config["id"] = workflowTrigger.ID

					trigger, err := registry.CreateTrigger(workflowTrigger.TriggerID, config)
					if err != nil {
						fmt.Printf("    ❌ INVALID: %v\n", err)
						invalidTriggers++
					}

					err = trigger.Validate()

					if err != nil {
						fmt.Printf("    ❌ INVALID: %v\n", err)
						invalidTriggers++
					} else {
						fmt.Printf("    ✅ VALID\n")
						validTriggers++
					}
				}
			}

			fmt.Printf("\nValidation Summary:\n")
			fmt.Printf("  Total triggers: %d\n", totalTriggers)
			fmt.Printf("  Valid triggers: %d\n", validTriggers)
			fmt.Printf("  Invalid triggers: %d\n", invalidTriggers)

			if invalidTriggers > 0 {
				return fmt.Errorf("found %d invalid triggers", invalidTriggers)
			}

			fmt.Println("All triggers are valid! ✅")
			return nil
		},
	}
}
