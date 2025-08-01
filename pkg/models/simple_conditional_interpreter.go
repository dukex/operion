// Package models provides conditional expression evaluation for workflow steps
package models

import (
	"fmt"
	"strconv"
)

type SimpleConditionalInterpreter struct{}

func (s SimpleConditionalInterpreter) Evaluate(exp any) (bool, error) {
	if exp == nil {
		return true, nil
	}

	switch v := exp.(type) {
	case bool:
		return v, nil
	case string:
		if v == "" {
			return true, nil
		}

		result, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("cannot convert string %q to boolean: %w", v, err)
		}
		return result, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to boolean", exp)
	}
}
