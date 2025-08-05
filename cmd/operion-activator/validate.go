package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v3"
)

var validate *validator.Validate

func NewValidateCommand() *cli.Command {
	return &cli.Command{
		Name:    "validate",
		Aliases: []string{"v"},
		Usage:   "Validate source configurations and trigger mappings",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "Database connection URL for persistence",
				Required: true,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			validate = validator.New(validator.WithRequiredStructEnabled())

			logger := slog.With(
				"module", "operion-activator",
				"action", "validate",
			)

			persistence := cmd.NewPersistence(logger, command.String("database-url"))

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

			logger.Info("Validating source triggers", "workflows", len(workflows))

			fmt.Println("Source Trigger Validation Results:")
			fmt.Println("==================================")

			validTriggers := 0
			invalidTriggers := 0
			validSteps := 0
			invalidSteps := 0

			for _, workflow := range workflows {
				fmt.Printf("\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)
				if len(workflow.WorkflowTriggers) == 0 {
					fmt.Printf("    ❌ INVALID: No triggers found for this workflow.\n")
					invalidTriggers++
					continue
				}

				for _, workflowTrigger := range workflow.WorkflowTriggers {
					fmt.Printf("  WorkflowTrigger: %s (SourceID: %s)\n", workflowTrigger.ID, workflowTrigger.SourceID)

					// Validate struct fields
					err = validate.Struct(workflowTrigger)
					if err != nil {
						validationErrors := err.(validator.ValidationErrors)
						fmt.Printf("    ❌ INVALID: %v\n", validationErrors)
						invalidTriggers++
						continue
					}

					// Validate that SourceID is not empty (required for activator)
					if workflowTrigger.SourceID == "" {
						fmt.Printf("    ❌ INVALID: SourceID is required for activator-based triggers\n")
						invalidTriggers++
						continue
					}

					// TODO: Add validation for source event types once source events are defined
					// This could validate that the trigger configuration includes valid event types
					// that the activator can process

					fmt.Printf("    ✅ VALID\n")
					validTriggers++
				}

				for _, step := range workflow.Steps {
					fmt.Printf("  Step: %s\n", step.Name)

					err = validate.Struct(step)

					if err != nil {
						validationErrors := err.(validator.ValidationErrors)

						fmt.Printf("    ❌ INVALID: %v\n", validationErrors)
						invalidSteps++
					} else {
						validSteps++
						fmt.Printf("    ✅ VALID\n")
					}
				}
			}

			fmt.Printf("\nValidation Summary:\n")
			fmt.Printf("  Total triggers: %d\n", invalidTriggers+validTriggers)
			fmt.Printf("  Valid triggers: %d\n", validTriggers)
			fmt.Printf("  Invalid triggers: %d\n", invalidTriggers)
			fmt.Printf("  Total steps: %d\n", invalidSteps+validSteps)
			fmt.Printf("  Valid steps: %d\n", validSteps)
			fmt.Printf("  Invalid steps: %d\n", invalidSteps)

			if invalidTriggers > 0 {
				return fmt.Errorf("found %d invalid triggers", invalidTriggers)
			}

			if invalidSteps > 0 {
				return fmt.Errorf("found %d invalid steps", invalidSteps)
			}

			fmt.Println("All triggers and steps are valid for activator processing! ✅")
			return nil
		},
	}
}
