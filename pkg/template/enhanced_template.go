package template

import (
	"os"
	"strings"

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

	// Use JSONata for templating with enhanced context
	return Render(input, enhancedData)
}

// getEnvVars returns environment variables as a map
// NeedsTemplating checks if a string contains JSONata expressions that need templating
func NeedsTemplating(input string) bool {
	return strings.Contains(input, "vars.") ||
		strings.Contains(input, "env.") ||
		strings.Contains(input, "steps.") ||
		strings.Contains(input, "trigger.") ||
		strings.Contains(input, "execution.") ||
		strings.Contains(input, "metadata.") ||
		strings.Contains(input, "$") || // JSONata functions start with $
		strings.Contains(input, "&") // JSONata string concatenation
}

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
