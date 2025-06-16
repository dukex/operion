package transform_action

import (
	"context"
	"fmt"

	"github.com/dukex/operion/internal/domain"
	log "github.com/sirupsen/logrus"
	jsonata "github.com/xiatechs/jsonata-go"
)

type TransformAction struct {
	ID         string
	Input      string
	Expression string
}

func NewTransformAction(config map[string]interface{}) (*TransformAction, error) {
	id, _ := config["id"].(string)
	input, _ := config["input"].(string)
	expression, _ := config["exp"].(string)

	return &TransformAction{
		ID:         id,
		Input:      input,
		Expression: expression,
	}, nil
}

func (a *TransformAction) GetID() string   { return a.ID }
func (a *TransformAction) GetType() string { return "transform" }
func (a *TransformAction) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":    a.ID,
		"input": a.Input,
		"exp":   a.Expression,
	}
}
func (a *TransformAction) Validate() error { return nil }

func (a *TransformAction) Execute(ctx context.Context, executionCtx domain.ExecutionContext) (interface{}, error) {

	logger := executionCtx.Logger.WithFields(log.Fields{
		"module": "http_request_action",
	})
	logger.Info("Executing TransformAction")

	data, err := a.extract(executionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	result, err := a.transform(data)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	logger.Info("TransformAction completed successfully")
	return result, nil
}

func (a *TransformAction) extract(executionCtx domain.ExecutionContext) (interface{}, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	e := jsonata.MustCompile(a.Input)
	results, err := e.Eval(executionCtx.StepResults)

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input expression '%s': %w", a.Input, err)
	}

	return results, nil
}

func (a *TransformAction) transform(data interface{}) (interface{}, error) {
	e := jsonata.MustCompile(a.Expression)

	result, err := e.Eval(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", a.Expression, err)
	}

	return result, nil
}
