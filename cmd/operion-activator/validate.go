package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v3"
)

var validate *validator.Validate

// Static error variables for linter compliance
var (
	ErrInvalidTriggers = errors.New("invalid triggers found")
	ErrInvalidSteps    = errors.New("invalid steps found")
)

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

			persistence := cmd.NewPersistence(ctx, logger, command.String("database-url"))

			defer func() {
				if err := persistence.Close(ctx); err != nil {
					return
				}
			}()

			workflowRepository := workflow.NewRepository(persistence)

			workflows, err := workflowRepository.FetchAll(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch workflows: %w", err)
			}

			logger.Info("Validating source triggers", "workflows", len(workflows))

			fmt.Fprintln(os.Stdout, "Source Trigger Validation Results:")
			fmt.Fprintln(os.Stdout, "==================================")

			validTriggers := 0
			invalidTriggers := 0
			validSteps := 0
			invalidSteps := 0

			for _, workflow := range workflows {
				fmt.Fprintf(os.Stdout, "\nWorkflow: %s (%s)\n", workflow.Name, workflow.ID)
				if len(workflow.WorkflowTriggers) == 0 {
					fmt.Fprintf(os.Stdout, "    ❌ INVALID: No triggers found for this workflow.\n")
					invalidTriggers++

					continue
				}

				for _, workflowTrigger := range workflow.WorkflowTriggers {
					fmt.Fprintf(os.Stdout, "  WorkflowTrigger: %s (SourceID: %s)\n", workflowTrigger.ID, workflowTrigger.SourceID)

					// Validate struct fields
					err = validate.Struct(workflowTrigger)
					if err != nil {
						var validationErrors validator.ValidationErrors
						if errors.As(err, &validationErrors) {
							fmt.Fprintf(os.Stdout, "    ❌ INVALID: %v\n", validationErrors)
						} else {
							fmt.Fprintf(os.Stdout, "    ❌ INVALID: %v\n", err)
						}
						invalidTriggers++

						continue
					}

					// Validate that SourceID is not empty (required for activator)
					if workflowTrigger.SourceID == "" {
						fmt.Fprintf(os.Stdout, "    ❌ INVALID: SourceID is required for activator-based triggers\n")
						invalidTriggers++

						continue
					}

					// TODO: Add validation for source event types once source events are defined
					// This could validate that the trigger configuration includes valid event types
					// that the activator can process

					fmt.Fprintf(os.Stdout, "    ✅ VALID\n")
					validTriggers++
				}

				for _, step := range workflow.Steps {
					fmt.Fprintf(os.Stdout, "  Step: %s\n", step.Name)

					err = validate.Struct(step)

					if err != nil {
						var validationErrors validator.ValidationErrors
						if errors.As(err, &validationErrors) {
							fmt.Fprintf(os.Stdout, "    ❌ INVALID: %v\n", validationErrors)
						} else {
							fmt.Fprintf(os.Stdout, "    ❌ INVALID: %v\n", err)
						}
						invalidSteps++
					} else {
						validSteps++
						fmt.Fprintf(os.Stdout, "    ✅ VALID\n")
					}
				}
			}

			fmt.Fprintf(os.Stdout, "\nValidation Summary:\n")
			fmt.Fprintf(os.Stdout, "  Total triggers: %d\n", invalidTriggers+validTriggers)
			fmt.Fprintf(os.Stdout, "  Valid triggers: %d\n", validTriggers)
			fmt.Fprintf(os.Stdout, "  Invalid triggers: %d\n", invalidTriggers)
			fmt.Fprintf(os.Stdout, "  Total steps: %d\n", invalidSteps+validSteps)
			fmt.Fprintf(os.Stdout, "  Valid steps: %d\n", validSteps)
			fmt.Fprintf(os.Stdout, "  Invalid steps: %d\n", invalidSteps)

			if invalidTriggers > 0 {
				return fmt.Errorf("%w: %d", ErrInvalidTriggers, invalidTriggers)
			}

			if invalidSteps > 0 {
				return fmt.Errorf("%w: %d", ErrInvalidSteps, invalidSteps)
			}

			fmt.Fprintln(os.Stdout, "All triggers and steps are valid for activator processing! ✅")
			return nil
		},
	}
}
