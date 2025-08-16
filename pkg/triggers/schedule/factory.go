package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/operion-flow/interfaces"
)

var (
	// ErrConfigNil is returned when config parameter is nil.
	ErrConfigNil = errors.New("config cannot be nil")
)

// TriggerFactory implements interfaces.TriggerFactory for creating TriggerFactory instances.
type TriggerFactory struct{}

// NewTriggerFactory creates a new instance of TriggerFactory.
func NewTriggerFactory() interfaces.TriggerFactory {
	return &TriggerFactory{}
}

// ID returns the unique identifier of the trigger factory.
func (f *TriggerFactory) ID() string {
	return "schedule"
}

// Name returns the human-readable name of the trigger factory.
func (f *TriggerFactory) Name() string {
	return "Schedule"
}

// Description returns a brief description of the trigger factory.
func (f *TriggerFactory) Description() string {
	return "Trigger workflow execution based on cron schedule expressions"
}

// Schema returns the JSON schema for the configuration of this trigger.
// nolint: funlen
func (f *TriggerFactory) Schema() map[string]any {
	//nolint: lll
	return map[string]any{
		"type":        "object",
		"title":       "Schedule Trigger Configuration",
		"description": "Configuration for cron-based workflow triggering using standard 5-field cron expressions or predefined descriptors",
		"properties": map[string]any{
			"cron": map[string]any{
				"type":        "string",
				"description": "Cron expression defining the schedule. Supports 5-field format (minute hour day month weekday) and predefined descriptors (@daily, @hourly, @every)",
				"examples": []any{
					map[string]string{
						"value":       "0 9 * * *",
						"description": "Daily at 9:00 AM UTC",
					},
					map[string]string{
						"value":       "*/15 * * * *",
						"description": "Every 15 minutes",
					},
					map[string]string{
						"value":       "0 0 1 * *",
						"description": "First day of every month at midnight",
					},
					map[string]string{
						"value":       "0 18 * * 5",
						"description": "Every Friday at 6:00 PM UTC",
					},
					map[string]string{
						"value":       "0 9-17 * * 1-5",
						"description": "Every hour from 9 AM to 5 PM on weekdays",
					},
					map[string]string{
						"value":       "@daily",
						"description": "Once a day at midnight (equivalent to '0 0 * * *')",
					},
					map[string]string{
						"value":       "@hourly",
						"description": "Once an hour at the beginning of the hour (equivalent to '0 * * * *')",
					},
					map[string]string{
						"value":       "@every 30m",
						"description": "Every 30 minutes using duration format",
					},
				},
				"oneOf": []map[string]any{
					{
						"pattern":     `^(\*|[0-5]?\d)\s+(\*|[01]?\d|2[0-3])\s+(\*|[12]?\d|3[01])\s+(\*|[1-9]|1[0-2])\s+(\*|[0-6])$`,
						"description": "Standard 5-field cron expression (minute hour day month weekday)",
					},
					{
						"pattern":     `^@(yearly|annually|monthly|weekly|daily|midnight|hourly)$`,
						"description": "Predefined schedule descriptors",
					},
					{
						"pattern":     `^@every\s+\d+(\.\d+)?(ns|us|µs|ms|s|m|h)+$`,
						"description": "Interval-based scheduling with duration format",
					},
				},
				"errorMessage": map[string]string{
					"pattern": "Invalid cron expression. Use 5-field format (minute hour day month weekday), predefined descriptors (@daily, @hourly), or @every with duration (e.g., @every 30m)",
					"oneOf":   "Cron expression must be either a 5-field format, predefined descriptor, or @every duration",
				},
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "Whether this trigger is active. Set to false to temporarily disable without removing the trigger configuration",
				"default":     true,
				"examples": []any{
					map[string]any{
						"value":       true,
						"description": "Trigger is active and will execute on schedule",
					},
					map[string]any{
						"value":       false,
						"description": "Trigger is disabled and will not execute",
					},
				},
			},
		},
		"required":             []string{"cron"},
		"additionalProperties": false,
		"$comment":             "All times are in UTC. Consider timezone implications when scheduling workflows",
		"examples": []map[string]any{
			{
				"description": "Daily backup at 2 AM UTC",
				"value": map[string]any{
					"cron":    "0 2 * * *",
					"enabled": true,
				},
			},
			{
				"description": "Business hours monitoring (9 AM - 5 PM, weekdays)",
				"value": map[string]any{
					"cron": "0 9-17 * * 1-5",
				},
			},
			{
				"description": "High-frequency data processing every 5 minutes",
				"value": map[string]any{
					"cron": "*/5 * * * *",
				},
			},
			{
				"description": "Weekly maintenance on Sunday at midnight",
				"value": map[string]any{
					"cron": "@weekly",
				},
			},
			{
				"description": "Continuous monitoring every 30 seconds (testing only)",
				"value": map[string]any{
					"cron": "@every 30s",
				},
			},
			{
				"description": "Disabled trigger for maintenance",
				"value": map[string]any{
					"cron":    "0 9 * * *",
					"enabled": false,
				},
			},
		},
		"documentation": map[string]any{
			"quickStart": map[string]string{
				"basicDaily":    "For daily execution: use '@daily' or '0 0 * * *'",
				"businessHours": "For business hours: use '0 9-17 * * 1-5' (9 AM - 5 PM, weekdays)",
				"testing":       "For testing: use '@every 1m' for quick iterations",
				"timezoneNote":  "Remember: all schedules run in UTC timezone",
			},
			"commonPatterns": map[string]string{
				"everyHour":      "0 * * * *",
				"twiceDaily":     "0 6,18 * * *",
				"weekdays":       "0 9 * * 1-5",
				"monthlyFirst":   "0 0 1 * *",
				"quarterlyFirst": "0 0 1 1,4,7,10 *",
			},
		},
	}
}

// Create initializes a new TriggerFactory with the provided configuration and logger.
func (f *TriggerFactory) Create(
	ctx context.Context,
	config map[string]any,
	logger *slog.Logger,
) (interfaces.Trigger, error) {
	if config == nil {
		return nil, ErrConfigNil
	}

	trigger, err := NewTrigger(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule trigger: %w", err)
	}

	return trigger, nil
}
