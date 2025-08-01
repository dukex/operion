package models

type Conditional interface {
	Evaluate(exp any) (bool, error)
}

func GetConditional(c ConditionalExpression) Conditional {
	if c.Language == "simple" {
		return &SimpleConditionalInterpreter{}
	}

	return nil
}
