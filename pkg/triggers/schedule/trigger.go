// Package schedule provides cron-based scheduling trigger implementation.
package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alaingilbert/cron"
	"github.com/dukex/operion/pkg/protocol"
)

var (
	// ErrCronRequired is returned when cron field is missing or not a string.
	ErrCronRequired = errors.New("cron field is required and must be a string")
	// ErrCronExpressionRequired is returned when cron expression is empty.
	ErrCronExpressionRequired = errors.New("cron expression is required")
	// ErrEnabledFieldType is returned when enabled field is not a boolean.
	ErrEnabledFieldType = errors.New("enabled field must be a boolean (true or false)")
)

// Trigger represents a cron-based trigger that executes workflows based on a schedule.
type Trigger struct {
	CronExpr string
	Enabled  bool
	cron     *cron.Cron
	callback protocol.TriggerCallback
	logger   *slog.Logger
}

// NewTrigger creates a new ScheduleTrigger with the provided configuration and logger.
func NewTrigger(ctx context.Context, config map[string]any, logger *slog.Logger) (*Trigger, error) {
	cronExpr, ok := config["cron"].(string)
	if !ok {
		return nil, ErrCronRequired
	}

	enabled := true

	enabledVal, exists := config["enabled"]
	if exists {
		if enabledBool, ok := enabledVal.(bool); ok {
			enabled = enabledBool
		} else {
			return nil, fmt.Errorf("%w, got %T", ErrEnabledFieldType, enabledVal)
		}
	}

	trigger := &Trigger{
		CronExpr: cronExpr,
		Enabled:  enabled,
		logger: logger.With(
			"module", "schedule_trigger",
			"cron", cronExpr,
			"enabled", enabled,
		),
	}

	err := trigger.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

// Validate checks the cron expression and provides helpful suggestions if invalid.
func (t *Trigger) Validate(ctx context.Context) error {
	if t.CronExpr == "" {
		return ErrCronExpressionRequired
	}

	_, err := cron.ParseStandard(t.CronExpr)
	if err != nil {
		suggestions := buildCronSuggestions(t.CronExpr)

		return fmt.Errorf("invalid cron expression '%s': %w\n\n%s", t.CronExpr, err, suggestions)
	}

	// Warn about timezone implications
	t.logger.InfoContext(ctx, "Schedule trigger configured successfully",
		"cron", t.CronExpr,
		"timezone", "UTC",
		"next_execution_hint", "All schedules run in UTC timezone. Consider your local timezone when setting schedules.")

	return nil
}

// buildCronSuggestions provides helpful error messages and suggestions for common cron expression mistakes.
func buildCronSuggestions(cronExpr string) string {
	suggestions := "Common fixes:\n"

	// Check for common patterns and suggest fixes
	switch {
	case len(cronExpr) == 0:
		suggestions += "• Cron expression cannot be empty\n"
		suggestions += "• Try: '0 9 * * *' for daily at 9 AM UTC\n"
		suggestions += "• Try: '@daily' for daily at midnight UTC\n"
	case cronExpr == "* * * * * *":
		suggestions += "• This appears to be 6-field format, but Operion uses 5-field format\n"
		suggestions += "• Remove the seconds field: '* * * * *' for every minute\n"
	case cronExpr == "0 0 0 * *":
		suggestions += "• Day of month cannot be 0, it should be 1-31\n"
		suggestions += "• Try: '0 0 1 * *' for first day of every month\n"
	default:
		suggestions += "• Ensure 5 fields: minute (0-59), hour (0-23), day (1-31), month (1-12), weekday (0-6)\n"
		suggestions += "• Use spaces to separate fields: '0 9 * * *'\n"
		suggestions += "• For ranges use '-': '0 9-17 * * *'\n"
		suggestions += "• For lists use ',': '0 9,17 * * *'\n"
		suggestions += "• For intervals use '/': '*/15 * * * *'\n"
	}

	suggestions += "\nExamples:\n"
	suggestions += "• '0 9 * * *'     - Every day at 9:00 AM UTC\n"
	suggestions += "• '*/15 * * * *'  - Every 15 minutes\n"
	suggestions += "• '0 9-17 * * 1-5' - Every hour 9 AM-5 PM on weekdays\n"
	suggestions += "• '@daily'        - Once a day at midnight\n"
	suggestions += "• '@every 30m'    - Every 30 minutes\n"

	return suggestions
}

// Start starts the schedule trigger.
func (t *Trigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	if !t.Enabled {
		t.logger.InfoContext(ctx, "ScheduleTrigger is disabled.")

		return nil
	}

	t.logger.InfoContext(ctx, "Starting ScheduleTrigger")

	t.callback = callback

	t.cron = cron.New().
		WithLogger(t.logger).
		WithContext(ctx).
		Build()

	id, err := t.cron.AddJob(t.CronExpr, t.run)

	t.logger.InfoContext(ctx, "Adding cron job for trigger", "id", id)

	if err != nil {
		return fmt.Errorf("failed to add cron job for trigger with cron %s: %w", t.CronExpr, err)
	}

	t.cron.Start()

	return nil
}

// Stop stops the schedule trigger.
func (t *Trigger) Stop(ctx context.Context) error {
	t.logger.InfoContext(ctx, "Stopping ScheduleTrigger", "cron", t.CronExpr)

	if t.cron != nil {
		t.cron.Stop()
	}

	return nil
}

func (t *Trigger) run(ctx context.Context) {
	t.logger.InfoContext(ctx, "Cron job triggered")

	triggerData := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	go func() {
		err := t.callback(ctx, triggerData)
		if err != nil {
			t.logger.ErrorContext(ctx, "Error executing workflow for trigger", "error", err)
		}
	}()
}
