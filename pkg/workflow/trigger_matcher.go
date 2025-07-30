package workflow

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
)

// TriggerMatcher handles matching trigger events against workflow configurations
type TriggerMatcher struct {
	logger *slog.Logger
}

// MatchResult represents the result of trigger matching
type MatchResult struct {
	Workflow      *models.Workflow
	MatchedTrigger *models.WorkflowTrigger
	MatchScore    int // Higher score indicates better match
}

// NewTriggerMatcher creates a new trigger matcher
func NewTriggerMatcher(logger *slog.Logger) *TriggerMatcher {
	return &TriggerMatcher{
		logger: logger.With("module", "trigger_matcher"),
	}
}

// MatchWorkflows finds workflows that match the given trigger event
func (tm *TriggerMatcher) MatchWorkflows(triggerEvent events.TriggerEvent, workflows []*models.Workflow) []MatchResult {
	var results []MatchResult

	tm.logger.Debug("Matching trigger event against workflows",
		"trigger_type", triggerEvent.TriggerType,
		"source", triggerEvent.Source,
		"workflows_count", len(workflows))

	for _, workflow := range workflows {
		// Skip inactive workflows
		if workflow.Status != models.WorkflowStatusActive {
			continue
		}

		// Check each trigger in the workflow
		for _, wt := range workflow.WorkflowTriggers {
			if match := tm.matchTrigger(triggerEvent, wt); match != nil {
				results = append(results, MatchResult{
					Workflow:       workflow,
					MatchedTrigger: wt,
					MatchScore:     match.Score,
				})
				tm.logger.Debug("Found matching workflow",
					"workflow_id", workflow.ID,
					"workflow_name", workflow.Name,
					"trigger_id", wt.TriggerID,
					"score", match.Score)
			}
		}
	}

	tm.logger.Info("Completed trigger matching",
		"trigger_type", triggerEvent.TriggerType,
		"source", triggerEvent.Source,
		"matches_found", len(results))

	return results
}

// TriggerMatch represents a single trigger match
type TriggerMatch struct {
	Score  int
	Reason string
}

// matchTrigger checks if a workflow trigger matches the trigger event
func (tm *TriggerMatcher) matchTrigger(triggerEvent events.TriggerEvent, workflowTrigger *models.WorkflowTrigger) *TriggerMatch {
	// Match by trigger type
	switch triggerEvent.TriggerType {
	case "kafka":
		return tm.matchKafkaTrigger(triggerEvent, workflowTrigger)
	case "webhook":
		return tm.matchWebhookTrigger(triggerEvent, workflowTrigger)
	case "schedule":
		return tm.matchScheduleTrigger(triggerEvent, workflowTrigger)
	default:
		tm.logger.Warn("Unknown trigger type", "type", triggerEvent.TriggerType)
		return nil
	}
}

// matchKafkaTrigger matches Kafka trigger events
func (tm *TriggerMatcher) matchKafkaTrigger(triggerEvent events.TriggerEvent, workflowTrigger *models.WorkflowTrigger) *TriggerMatch {
	// Check if workflow trigger is for Kafka
	if workflowTrigger.TriggerID != "kafka" {
		return nil
	}

	score := 0

	// Match topic
	configTopic, exists := workflowTrigger.Configuration["topic"]
	if !exists {
		return nil
	}

	triggerTopic, exists := triggerEvent.TriggerData["topic"]
	if !exists {
		return nil
	}

	if fmt.Sprintf("%v", configTopic) == fmt.Sprintf("%v", triggerTopic) {
		score += 100 // Exact topic match
	} else {
		return nil // Topic must match exactly
	}

	// Optional: Match message key pattern
	if configKeyPattern, exists := workflowTrigger.Configuration["key_pattern"]; exists {
		triggerKey, keyExists := triggerEvent.TriggerData["key"]
		if keyExists {
			keyStr := fmt.Sprintf("%v", triggerKey)
			patternStr := fmt.Sprintf("%v", configKeyPattern)
			if tm.matchPattern(keyStr, patternStr) {
				score += 50 // Key pattern match
			}
		}
	}

	// Optional: Match message content filters
	if contentFilters, exists := workflowTrigger.Configuration["content_filters"]; exists {
		if filtersMap, ok := contentFilters.(map[string]interface{}); ok {
			if tm.matchContentFilters(triggerEvent.TriggerData["message"], filtersMap) {
				score += 25 // Content filter match
			}
		}
	}

	return &TriggerMatch{
		Score:  score,
		Reason: fmt.Sprintf("Kafka topic match: %v", triggerTopic),
	}
}

// matchWebhookTrigger matches webhook trigger events
func (tm *TriggerMatcher) matchWebhookTrigger(triggerEvent events.TriggerEvent, workflowTrigger *models.WorkflowTrigger) *TriggerMatch {
	// Check if workflow trigger is for webhook
	if workflowTrigger.TriggerID != "webhook" {
		return nil
	}

	score := 0

	// Match path
	configPath, exists := workflowTrigger.Configuration["path"]
	if !exists {
		return nil
	}

	triggerPath, exists := triggerEvent.TriggerData["path"]
	if !exists {
		return nil
	}

	if fmt.Sprintf("%v", configPath) == fmt.Sprintf("%v", triggerPath) {
		score += 100 // Exact path match
	} else {
		return nil // Path must match exactly
	}

	// Optional: Match method
	if configMethod, exists := workflowTrigger.Configuration["method"]; exists {
		triggerMethod, methodExists := triggerEvent.TriggerData["method"]
		if methodExists {
			if fmt.Sprintf("%v", configMethod) == fmt.Sprintf("%v", triggerMethod) {
				score += 50 // Method match
			} else {
				return nil // Method must match if specified
			}
		}
	}

	// Optional: Match headers
	if configHeaders, exists := workflowTrigger.Configuration["required_headers"]; exists {
		if headersMap, ok := configHeaders.(map[string]interface{}); ok {
			originalData := triggerEvent.OriginalData
			if requestHeaders, exists := originalData["headers"]; exists {
				if tm.matchHeaders(requestHeaders, headersMap) {
					score += 25 // Header match
				}
			}
		}
	}

	return &TriggerMatch{
		Score:  score,
		Reason: fmt.Sprintf("Webhook path match: %v", triggerPath),
	}
}

// matchScheduleTrigger matches schedule trigger events
func (tm *TriggerMatcher) matchScheduleTrigger(triggerEvent events.TriggerEvent, workflowTrigger *models.WorkflowTrigger) *TriggerMatch {
	// Check if workflow trigger is for schedule
	if workflowTrigger.TriggerID != "schedule" {
		return nil
	}

	// For schedule triggers, we typically match by schedule ID or cron expression
	// This is a simplified implementation - in practice, you might want more sophisticated matching
	
	score := 100 // Schedule triggers typically match exactly

	return &TriggerMatch{
		Score:  score,
		Reason: "Schedule trigger match",
	}
}

// Helper methods for pattern matching

// matchPattern performs simple pattern matching (supports wildcards)
func (tm *TriggerMatcher) matchPattern(value, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			return strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix)
		}
	}
	
	return value == pattern
}

// matchContentFilters matches content against filters
func (tm *TriggerMatcher) matchContentFilters(messageData interface{}, filters map[string]interface{}) bool {
	if messageData == nil {
		return false
	}

	// Convert message data to map for easier filtering
	messageMap, ok := messageData.(map[string]interface{})
	if !ok {
		return false
	}

	// Check each filter condition
	for field, expectedValue := range filters {
		actualValue, exists := messageMap[field]
		if !exists {
			return false
		}

		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}

	return true
}

// matchHeaders matches request headers against required headers
func (tm *TriggerMatcher) matchHeaders(requestHeaders interface{}, requiredHeaders map[string]interface{}) bool {
	headersMap, ok := requestHeaders.(map[string]interface{})
	if !ok {
		return false
	}

	for headerName, expectedValue := range requiredHeaders {
		actualValue, exists := headersMap[headerName]
		if !exists {
			return false
		}

		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}

	return true
}