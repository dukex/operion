// Package template provides templating functionality for dynamic workflow configuration.
package template

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/dukex/operion/pkg/models"
)

func RenderWithContext(input string, executionCtx *models.ExecutionContext) (any, error) {
	enhancedData := map[string]any{
		"steps":    executionCtx.StepResults,
		"vars":     executionCtx.Variables,
		"trigger":  executionCtx.TriggerData,
		"metadata": executionCtx.Metadata,
		"env":      getEnvVars(),
		"execution": map[string]any{
			"id":          executionCtx.ID,
			"workflow_id": executionCtx.WorkflowID,
		},
	}

	return Render(input, enhancedData)
}

func Render(templateStr string, data any) (any, error) {
	tmpl, err := template.
		New("transform").
		Parse(templateStr)

	if err != nil {
		return nil, fmt.Errorf("failed to parse template '%s': %w", templateStr, err)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template '%s': %w", templateStr, err)
	}

	result := buf.String()

	// Try to parse as JSON if it looks like JSON
	result = strings.TrimSpace(result)
	if (strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}")) ||
		(strings.HasPrefix(result, "[") && strings.HasSuffix(result, "]")) {
		var jsonResult any

		err := json.Unmarshal([]byte(result), &jsonResult)

		if err == nil {
			return jsonResult, nil
		}

		return jsonResult, fmt.Errorf("failed to parse json '%s': %w", templateStr, err)
	}

	// Try to parse as number
	if num, err := strconv.ParseFloat(result, 64); err == nil {
		return num, nil
	}

	// Try to parse as boolean
	if b, err := strconv.ParseBool(result); err == nil {
		return b, nil
	}

	// // Return as string
	return result, nil
}

// getEnvVars returns environment variables as a map
func getEnvVars() map[string]any {
	envMap := make(map[string]any)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	return envMap
}
