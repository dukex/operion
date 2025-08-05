// Package transform provides data transformation action implementation using golang template expressions.
package transform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/template"
)

// Action represents a data transformation action that applies a template expression to the execution context.
type Action struct {
	Expression string
}

// NewAction creates a new transform Action instance with the provided configuration.
func NewAction(config map[string]any) (*Action, error) {
	expression, _ := config["expression"].(string)

	return &Action{
		Expression: expression,
	}, nil
}

// Execute performs the transformation action, rendering the expression with templating if needed.
func (a *Action) Execute(
	ctx context.Context,
	executionCtx models.ExecutionContext,
	logger *slog.Logger,
) (any, error) {
	logger = logger.With(
		"module", "http_request_action",
	)
	logger.InfoContext(ctx, "Executing TransformAction")

	result, err := template.RenderWithContext(a.Expression, &executionCtx)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	logger.InfoContext(ctx, "TransformAction completed successfully")

	return result, nil
}

var errMessageInvalid = errors.New("invalid expression")

// Validate checks if the TransformAction has a valid expression.
func (a *Action) Validate(_ context.Context) error {
	if a.Expression == "" {
		return fmt.Errorf("expression cannot be empty: %w", errMessageInvalid)
	}

	_, err := template.Parse(a.Expression)
	if err != nil {
		return fmt.Errorf("invalid expression '%s': %w", a.Expression, errMessageInvalid)
	}

	return nil
}
