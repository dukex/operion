// Package transform provides data transformation action implementation using JSONata expressions.
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
	return "Transforms input data using a specified expression. The input can be a string or an expression that evaluates to data."
}

func (h *TransformActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "Input data source expression. If empty, uses all step results. Supports templating.",
				"examples": []string{
					"",
					"steps.fetch_users",
					"steps.api_call.response.data",
					"trigger.webhook.payload",
				},
			},
			"expression": map[string]any{
				"type":        "string",
				"format":      "code",
				"description": "JSONata expression to transform the input data. Use JSONata syntax for powerful data transformations.",
				"examples": []string{
					"$.name",
					"$.users[0].email",
					"{ \"fullName\": $.firstName & \" \" & $.lastName, \"isActive\": $.status = \"active\" }",
					"$.data.users.{ \"user_id\": id, \"display_name\": name, \"created\": $now() }",
					"$count($.items)",
					"$.orders[total > 100]",
				},
			},
		},
		"required": []string{"expression"},
	}
}

type TransformAction struct {
	Input      string
	Expression string
}

func NewTransformAction(config map[string]any) (*TransformAction, error) {
	input, _ := config["input"].(string)
	expression, _ := config["expression"].(string)

	return &TransformAction{
		Input:      input,
		Expression: expression,
	}, nil
}

func (a *TransformAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (any, error) {
	logger = logger.With(
		"module", "http_request_action",
	)
	logger.Info("Executing TransformAction")

	data, err := a.extract(executionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	result, err := template.Render(a.Expression, data)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	logger.Info("TransformAction completed successfully")
	return result, nil
}

func (a *TransformAction) extract(executionCtx models.ExecutionContext) (any, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	return template.Render(a.Input, executionCtx.StepResults)
}
