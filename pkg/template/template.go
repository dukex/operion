package template

import (
	"fmt"

	jsonata "github.com/xiatechs/jsonata-go"
)

func Render(input string, data interface{}) (interface{}, error) {
	e, err := jsonata.Compile(input)

	if err != nil {
		return nil, fmt.Errorf("failed to compile input expression '%s': %w", input, err)
	}

	results, err := e.Eval(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input expression '%s': %w", input, err)
	}

	return results, err
}
