package triggers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dukex/operion/internal/domain"
	"github.com/robfig/cron"
)

type ScheduleTrigger struct {
    ID       string
    CronExpr string
    Timezone string
    Enabled  bool
    cron     *cron.Cron
    callback domain.TriggerCallback
}

// NewScheduleTrigger creates and validates a ScheduleTrigger
func NewScheduleTrigger(config map[string]interface{}) (*ScheduleTrigger, error) {
    // In a real app, you'd use a library like mapstructure to decode this
    id, _ := config["id"].(string)
    cronExpr, _ := config["cron"].(string)
    timezone, _ := config["timezone"].(string)

    trigger := &ScheduleTrigger{
        ID:       id,
        CronExpr: cronExpr,
        Timezone: timezone,
        Enabled:  true, // Default or from config
    }
    if err := trigger.Validate(); err != nil {
        return nil, err
    }
    return trigger, nil
}

func (t *ScheduleTrigger) GetID() string   { return t.ID }
func (t *ScheduleTrigger) GetType() string { return "schedule" }
func (t *ScheduleTrigger) GetConfig() map[string]interface{} {
    return map[string]interface{}{
        "id": t.ID, "cron": t.CronExpr, "timezone": t.Timezone,
    }
}

func (t *ScheduleTrigger) Validate() error {
    if t.ID == "" {
        return errors.New("schedule trigger ID is required")
    }
    if t.CronExpr == "" {
        return errors.New("schedule trigger cron expression is required")
    }
    // Use a cron parsing library to validate the expression
    if _, err := cron.ParseStandard(t.CronExpr); err != nil {
        return fmt.Errorf("invalid cron expression: %w", err)
    }
    return nil
}

func (t *ScheduleTrigger) Start(ctx context.Context, callback domain.TriggerCallback) error {
    if !t.Enabled {
        log.Printf("ScheduleTrigger '%s' is disabled.", t.ID)
        return nil
    }
    log.Printf("Starting ScheduleTrigger '%s' with schedule '%s'", t.ID, t.CronExpr)
    t.callback = callback
    
    // Initialize cron scheduler
    if t.Timezone != "" {
        loc, err := time.LoadLocation(t.Timezone)
        if err != nil {
            log.Printf("Warning: Could not load timezone '%s'. Defaulting to local. Error: %v", t.Timezone, err)
            t.cron = cron.New()
        } else {
            t.cron = cron.NewWithLocation(loc)
        }
    } else {
         t.cron = cron.New()
    }

    err := t.cron.AddFunc(t.CronExpr, t.run)
    if err != nil {
        return fmt.Errorf("failed to add cron job for trigger %s: %w", t.ID, err)
    }
    t.cron.Start()
    return nil
}

func (t *ScheduleTrigger) run() {
    log.Printf("Cron job triggered for '%s'", t.ID)
    triggerData := map[string]interface{}{
        "trigger_id":   t.ID,
        "trigger_type": "schedule",
        "timestamp":    time.Now().UTC().Format(time.RFC3339),
    }
    // Run the callback in a separate goroutine to not block the scheduler
    go func() {
        if err := t.callback(context.Background(), triggerData); err != nil {
            log.Printf("Error executing workflow for trigger %s: %v", t.ID, err)
        }
    }()
}

func (t *ScheduleTrigger) Stop(ctx context.Context) error {
    log.Printf("Stopping ScheduleTrigger '%s'", t.ID)
    if t.cron != nil {
        t.cron.Stop()
    }
    return nil
}