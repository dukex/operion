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

func (h *TransformActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	return NewTransformAction(config)
}

func (h *TransformActionFactory) ID() string {
	return "transform"
}

type TransformAction struct {
	ID         string
	Input      string
	Expression string
}

func NewTransformAction(config map[string]interface{}) (*TransformAction, error) {
	id, _ := config["id"].(string)
	input, _ := config["input"].(string)
	expression, _ := config["expression"].(string)

	return &TransformAction{
		ID:         id,
		Input:      input,
		Expression: expression,
	}, nil
}

func (a *TransformAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (interface{}, error) {
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

func (a *TransformAction) extract(executionCtx models.ExecutionContext) (interface{}, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	return template.Render(a.Input, executionCtx.StepResults)
}
