package transform_action

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/models"
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
	expression, _ := config["expression"].(string)

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
		"id":         a.ID,
		"input":      a.Input,
		"expression": a.Expression,
	}
}
func (a *TransformAction) Validate() error { return nil }

// GetSchema returns the JSON Schema for Transform Action configuration
func GetTransformActionSchema() *models.RegisteredComponent {
	return &models.RegisteredComponent{
		Type:        "transform",
		Name:        "Transform Data",
		Description: "Transform data using JSONata expressions",
		Schema: &models.JSONSchema{
			Type:        "object",
			Title:       "Transform Action Configuration",
			Description: "Configuration for data transformation using JSONata",
			Properties: map[string]*models.Property{
				"expression": {
					Type:        "string",
					Description: "JSONata expression for data transformation",
				},
				"input": {
					Type:        "string",
					Description: "JSONata expression to extract input data (optional)",
				},
			},
			Required: []string{"expression"},
		},
	}
}

func (a *TransformAction) Execute(ctx context.Context, executionCtx models.ExecutionContext) (interface{}, error) {

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

func (a *TransformAction) extract(executionCtx models.ExecutionContext) (interface{}, error) {
	if a.Input == "" {
		return executionCtx.StepResults, nil
	}

	e, err := jsonata.Compile(a.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile input expression '%s': %w", a.Input, err)
	}

	results, err := e.Eval(executionCtx.StepResults)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input expression '%s': %w", a.Input, err)
	}

	return results, nil
}

func (a *TransformAction) transform(data interface{}) (interface{}, error) {
	e, err := jsonata.Compile(a.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", a.Expression, err)
	}

	result, err := e.Eval(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", a.Expression, err)
	}

	return result, nil
}
