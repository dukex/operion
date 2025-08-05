package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/cmd"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/go-playground/validator/v10"
	"github.com/urfave/cli/v3"
)

var validate *validator.Validate

// Static error variables for linter compliance
var (
	ErrInvalidTriggers              = errors.New("invalid triggers found")
	ErrInvalidSourceProviderConfigs = errors.New("invalid source provider configurations found")
)

func NewValidateCommand() *cli.Command {
	return &cli.Command{
		Name:    "validate",
		Aliases: []string{"v"},
		Usage:   "Validate source provider configurations and workflow triggers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "database-url",
				Usage:    "Database connection URL for persistence",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "plugins-path",
				Usage:    "Path to the directory containing source provider plugins",
				Value:    "./plugins",
				Required: false,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			validate = validator.New(validator.WithRequiredStructEnabled())

			logger := slog.With(
				"module", "operion-source-manager",
				"action", "validate",
			)

			persistence := cmd.NewPersistence(logger, command.String("database-url"))
			defer func() {
				if err := persistence.Close(); err != nil {
					return
				}
			}()

			registry := cmd.NewRegistry(logger, command.String("plugins-path"))

			workflowRepository := workflow.NewRepository(persistence)

			workflows, err := workflowRepository.FetchAll()
			if err != nil {
				return fmt.Errorf("failed to fetch workflows: %w", err)
			}

			logger.Info("Validating source provider configurations", "workflows", len(workflows))

			fmt.Println("Source Provider Validation Results:")
			fmt.Println("===================================")

			validTriggers := 0
			invalidTriggers := 0
			validProviders := 0
			invalidProviders := 0

			// Get available source providers from registry
			sourceProviders := registry.GetSourceProviders()
			fmt.Printf("Available source providers: %d\n", len(sourceProviders))
			for name, factory := range sourceProviders {
				fmt.Printf("  - %s: %s\n", name, factory.Description())
			}

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

					// Validate that SourceID is not empty
					if workflowTrigger.SourceID == "" {
						fmt.Printf("    ❌ INVALID: SourceID is required for source-based triggers\n")
						invalidTriggers++
						continue
					}

					// Try to determine source provider type from configuration
					// This is a heuristic approach since we don't have explicit provider type in trigger
					sourceProviderType := ""
					if cronExpr, exists := workflowTrigger.Configuration["cron_expression"]; exists && cronExpr != nil {
						sourceProviderType = "scheduler"
					}
					// Add more heuristics for other provider types as needed

					if sourceProviderType != "" {
						// Validate that the source provider exists
						if factory, exists := sourceProviders[sourceProviderType]; exists {
							fmt.Printf("    ✅ VALID: Source provider '%s' found\n", sourceProviderType)

							// Validate configuration schema if possible
							schema := factory.Schema()
							if schema != nil {
								fmt.Printf("    📋 Configuration schema available\n")
							}

							validProviders++
						} else {
							fmt.Printf("    ❌ INVALID: Source provider '%s' not found\n", sourceProviderType)
							invalidProviders++
						}
					} else {
						fmt.Printf("    ⚠️  WARNING: Could not determine source provider type from configuration\n")
					}

					validTriggers++
				}
			}

			fmt.Printf("\nValidation Summary:\n")
			fmt.Printf("  Total triggers: %d\n", invalidTriggers+validTriggers)
			fmt.Printf("  Valid triggers: %d\n", validTriggers)
			fmt.Printf("  Invalid triggers: %d\n", invalidTriggers)
			fmt.Printf("  Valid source providers: %d\n", validProviders)
			fmt.Printf("  Invalid source providers: %d\n", invalidProviders)

			if invalidTriggers > 0 {
				return fmt.Errorf("%w: %d", ErrInvalidTriggers, invalidTriggers)
			}

			if invalidProviders > 0 {
				return fmt.Errorf("%w: %d", ErrInvalidSourceProviderConfigs, invalidProviders)
			}

			fmt.Println("All source provider configurations are valid! ✅")
			return nil
		},
	}
}
