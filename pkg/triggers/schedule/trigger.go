// Package schedule provides cron-based scheduling trigger implementation.
package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/robfig/cron/v3"
)

var (
	ErrCronExpressionRequired = errors.New("schedule trigger cron expression is required")
)

type ScheduleTrigger struct {
	CronExpr string
	Enabled  bool
	cron     *cron.Cron
	callback protocol.TriggerCallback
	logger   *slog.Logger
}

func NewScheduleTrigger(config map[string]any, logger *slog.Logger) (*ScheduleTrigger, error) {
	cronExpr, _ := config["cron"].(string)

	enabled := true

	if enabledVal, exists := config["enabled"]; exists {
		if enabledBool, ok := enabledVal.(bool); ok {
			enabled = enabledBool
		}
	}

	trigger := &ScheduleTrigger{
		CronExpr: cronExpr,
		Enabled:  enabled,
		logger: logger.With(
			"module", "schedule_trigger",
			"cron", cronExpr,
			"enabled", enabled,
		),
	}

	err := trigger.Validate()
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *ScheduleTrigger) Validate() error {
	if t.CronExpr == "" {
		return ErrCronExpressionRequired
	}

	if _, err := cron.ParseStandard(t.CronExpr); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

func (t *ScheduleTrigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	if !t.Enabled {
		t.logger.Info("ScheduleTrigger is disabled.")

		return nil
	}

	t.logger.Info("Starting ScheduleTrigger")
	t.callback = callback

	t.cron = cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.DefaultLogger),
		cron.Recover(cron.DefaultLogger),
	))

	id, err := t.cron.AddFunc(t.CronExpr, t.run)

	t.logger.Info("Adding cron job for trigger", "id", id)

	if err != nil {
		return fmt.Errorf("failed to add cron job for trigger with cron %s: %w", t.CronExpr, err)
	}

	t.cron.Start()

	return nil
}

func (t *ScheduleTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping ScheduleTrigger", "cron", t.CronExpr)

	if t.cron != nil {
		t.cron.Stop()
	}

	return nil
}

func (t *ScheduleTrigger) run() {
	t.logger.Info("Cron job triggered")

	triggerData := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	go func() {
		err := t.callback(context.Background(), triggerData)
		if err != nil {
			t.logger.Error("Error executing workflow for trigger", "error", err)
		}
	}()
}
