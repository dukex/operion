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

type ScheduleTrigger struct {
	ID         string
	CronExpr   string
	WorkflowId string
	Enabled    bool
	cron       *cron.Cron
	callback   protocol.TriggerCallback
	logger     *slog.Logger
}

func NewScheduleTrigger(config map[string]interface{}, logger *slog.Logger) (*ScheduleTrigger, error) {
	id, _ := config["id"].(string)
	cronExpr, _ := config["cron"].(string)
	workflowId, _ := config["workflow_id"].(string)

	trigger := &ScheduleTrigger{
		ID:         id,
		CronExpr:   cronExpr,
		Enabled:    true,
		WorkflowId: workflowId,
		logger: logger.With(
			"module", "schedule_trigger",
			"id", id,
			"cron", cronExpr,
			"workflow_id", workflowId,
		),
	}
	if err := trigger.Validate(); err != nil {
		return nil, err
	}
	return trigger, nil
}

// func (t *ScheduleTrigger) GetID() string   { return t.ID }
// func (t *ScheduleTrigger) GetType() string { return "schedule" }
// func (t *ScheduleTrigger) GetConfig() map[string]interface{} {
// 	return map[string]interface{}{
// 		"id":          t.ID,
// 		"cron":        t.CronExpr,
// 		"enabled":     t.Enabled,
// 		"workflow_id": t.WorkflowId,
// 	}
// }

func (t *ScheduleTrigger) Validate() error {
	if t.ID == "" {
		return errors.New("schedule trigger ID is required")
	}
	if t.CronExpr == "" {
		return errors.New("schedule trigger cron expression is required")
	}
	if _, err := cron.ParseStandard(t.CronExpr); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

// // GetSchema returns the JSON Schema for Schedule Trigger configuration
// func GetScheduleTriggerSchema() *models.RegisteredComponent {
// 	return &models.RegisteredComponent{
// 		Type:        "schedule",
// 		Name:        "Schedule (Cron)",
// 		Description: "Trigger workflow on a schedule using cron expressions",
// 		Schema: &models.JSONSchema{
// 			Type:        "object",
// 			Title:       "Schedule Trigger Configuration",
// 			Description: "Configuration for cron-based scheduling",
// 			Properties: map[string]*models.Property{
// 				"cron": {
// 					Type:        "string",
// 					Description: "Cron expression (e.g., '0 */5 * * *' for every 5 minutes)",
// 					Pattern:     `^(\*|[0-5]?\d)(\s+(\*|[01]?\d|2[0-3]))(\s+(\*|[12]?\d|3[01]))(\s+(\*|[1-9]|1[0-2]))(\s+(\*|[0-6]))?$`,
// 				},
// 				"workflow_id": {
// 					Type:        "string",
// 					Description: "ID of the workflow to trigger",
// 				},
// 			},
// 			Required: []string{"cron"},
// 		},
// 	}
// }

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
		return fmt.Errorf("failed to add cron job for trigger %s: %w", t.ID, err)
	}
	t.cron.Start()
	return nil
}

func (t *ScheduleTrigger) run() {
	t.logger.Info("Cron job triggered")

	triggerData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	go func() {
		if err := t.callback(context.Background(), triggerData); err != nil {
			t.logger.Error("Error executing workflow for trigger", "error", err)
		}
	}()
}

func (t *ScheduleTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping ScheduleTrigger", "id", t.ID)

	if t.cron != nil {
		t.cron.Stop()
	}
	return nil
}
