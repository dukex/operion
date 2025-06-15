package domain

import "context"

type Conditional interface {
	Evaluate(ctx context.Context, input ExecutionContext) (bool, error)
	GetExpression() string
}
