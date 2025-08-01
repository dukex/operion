// Package transform provides data transformation action implementation using golang template expressions.
package transform

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/template"
)

func NewTransformActionFactory() *TransformActionFactory {
	return &TransformActionFactory{}
}

type TransformActionFactory struct{}

func (h *TransformActionFactory) Create(config map[string]any) (protocol.Action, error) {
	return NewTransformAction(config)
}

func (h *TransformActionFactory) ID() string {
	return "transform"
}

func (h *TransformActionFactory) Name() string {
	return "Transform"
}

func (h *TransformActionFactory) Description() string {
	return "Transforms data using a specified expression."
}

func (h *TransformActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"format":      "template",
				"description": "Go template expression to transform the data. Use Go template syntax with {{}} delimiters.",
				"examples": []string{
					"{{.name}}",
					"{{.users.0.email}}",
					"{\"fullName\": \"{{.firstName}} {{.lastName}}\", \"isActive\": {{eq .status \"active\"}}}",
					"{{range .data.users}}{\"user_id\": {{.id}}, \"display_name\": \"{{.name}}\"}{{end}}",
					"{{len .items}}",
					"{{with .orders}}{{range .}}{{if gt .total 100.0}}{{.}}{{end}}{{end}}{{end}}",
				},
			},
		},
		"required": []string{"expression"},
	}
}

type TransformAction struct {
	Expression string
}

func NewTransformAction(config map[string]any) (*TransformAction, error) {
	expression, _ := config["expression"].(string)

	return &TransformAction{
		Expression: expression,
	}, nil
}

func (a *TransformAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (any, error) {
	logger = logger.With(
		"module", "http_request_action",
	)
	logger.Info("Executing TransformAction")

	result, err := template.Render(a.Expression, executionCtx.StepResults)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	logger.Info("TransformAction completed successfully")
	return result, nil
}
