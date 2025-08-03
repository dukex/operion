package schedule

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

var (
	ErrConfigNil = errors.New("config cannot be nil")
)

func NewScheduleTriggerFactory() protocol.TriggerFactory {
	return &ScheduleTriggerFactory{}
}

type ScheduleTriggerFactory struct{}

func (f *ScheduleTriggerFactory) ID() string {
	return "schedule"
}

func (f *ScheduleTriggerFactory) Name() string {
	return "Schedule"
}

func (f *ScheduleTriggerFactory) Description() string {
	return "Trigger workflow execution based on cron schedule expressions"
}

func (f *ScheduleTriggerFactory) Schema() map[string]any {
	return map[string]any{
		"type":        "object",
		"title":       "Schedule Trigger Configuration",
		"description": "Configuration for cron-based workflow triggering",
		"properties": map[string]any{
			"cron": map[string]any{
				"type":        "string",
				"description": "Cron expression defining the schedule (standard 5-field format)",
				"pattern":     `^(\*|[0-5]?\d)(\s+(\*|[01]?\d|2[0-3]))(\s+(\*|[12]?\d|3[01]))(\s+(\*|[1-9]|1[0-2]))(\s+(\*|[0-6]))?$`,
				"examples": []string{
					"0 9 * * *",    // Daily at 9 AM
					"*/15 * * * *", // Every 15 minutes
					"0 0 1 * *",    // First day of every month
					"0 18 * * 5",   // Every Friday at 6 PM
					"30 2 * * 0",   // Every Sunday at 2:30 AM
				},
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "Whether this trigger is active",
				"default":     true,
				"examples":    []bool{true, false},
			},
		},
		"required": []string{"cron"},
		"examples": []map[string]any{
			{
				"cron":    "0 2 * * *",
				"enabled": true,
			},
			{
				"cron": "0 9-17 * * 1-5",
			},
			{
				"cron": "*/15 * * * *",
			},
		},
	}
}

func (f *ScheduleTriggerFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, ErrConfigNil
	}

	trigger, err := NewScheduleTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule trigger: %w", err)
	}

	return trigger, nil
}
