package webhook_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/triggers/webhook"
)

func TestWebhookTrigger_Validation(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]any{
				"path": "/webhook",
			},
			wantErr: false,
		},
		{
			name: "minimal config",
			config: map[string]any{
				"path": "/webhook",
			},
			wantErr: false,
		},
		{
			name: "invalid path",
			config: map[string]any{
				"path": "webhook",
			},
			wantErr: true,
		},
		{
			name:    "default path when missing",
			config:  map[string]any{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := webhook.NewTrigger(t.Context(), tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWebhookTrigger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebhookTrigger_StartStop(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	// Reset global manager for this test
	webhook.ResetGlobalManager()

	manager := webhook.GetWebhookServerManager(8081, logger)

	config := map[string]any{
		"id":   "test-webhook",
		"path": "/test",
	}

	trigger, err := webhook.NewTrigger(t.Context(), config, logger)
	if err != nil {
		t.Fatalf("Failed to create webhook trigger: %v", err)
	}

	callback := func(ctx context.Context, data map[string]any) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Failed to start webhook server manager: %v", err)
	}

	// Start trigger in a goroutine since it now blocks
	startDone := make(chan error, 1)

	go func() {
		startDone <- trigger.Start(ctx, callback)
	}()

	// Give the trigger time to register
	time.Sleep(100 * time.Millisecond)

	// Stop the trigger by cancelling context
	cancel()

	// Wait for trigger to finish
	select {
	case err := <-startDone:
		if err != nil {
			t.Errorf("Trigger Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Trigger Start did not return within timeout")
	}

	err = trigger.Stop(context.Background())
	if err != nil {
		t.Errorf("Failed to stop webhook trigger: %v", err)
	}

	err = manager.Stop(context.Background())
	if err != nil {
		t.Errorf("Failed to stop webhook server manager: %v", err)
	}
}

func TestWebhookTrigger_ServerShutdown(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	// Reset global manager for this test
	webhook.ResetGlobalManager()

	manager := webhook.GetWebhookServerManager(8082, logger)

	config := map[string]any{
		"id":   "test-webhook-shutdown",
		"path": "/test-shutdown",
	}

	trigger, err := webhook.NewTrigger(t.Context(), config, logger)
	if err != nil {
		t.Fatalf("Failed to create webhook trigger: %v", err)
	}

	callback := func(ctx context.Context, data map[string]any) error {
		return nil
	}

	ctx := context.Background()

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Failed to start webhook server manager: %v", err)
	}

	// Start trigger in a goroutine since it blocks
	startDone := make(chan error, 1)

	go func() {
		startDone <- trigger.Start(ctx, callback)
	}()

	// Give the trigger time to register
	time.Sleep(100 * time.Millisecond)

	// Stop the server manager
	err = manager.Stop(context.Background())
	if err != nil {
		t.Errorf("Failed to stop webhook server manager: %v", err)
	}

	// Wait for trigger to finish due to server shutdown
	select {
	case err := <-startDone:
		if err != nil {
			t.Errorf("Trigger Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Trigger Start did not return within timeout after server shutdown")
	}
}

func TestWebhookTriggerFactory(t *testing.T) {
	t.Parallel()

	factory := webhook.NewTriggerFactory()

	if factory.ID() != "webhook" {
		t.Errorf("Expected factory ID 'webhook', got '%s'", factory.ID())
	}

	logger := slog.Default()
	config := map[string]any{
		"id":   "test-webhook",
		"path": "/webhook",
	}

	trigger, err := factory.Create(t.Context(), config, logger)
	if err != nil {
		t.Fatalf("Failed to create trigger from factory: %v", err)
	}

	// Verify trigger implements protocol.Trigger interface
	if trigger == nil {
		t.Error("Created trigger is nil")
	}
}
