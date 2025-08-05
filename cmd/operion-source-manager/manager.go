package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
)

// Constants for source provider types
const (
	SchedulerProviderType = "scheduler"
)

type SourceProviderManager struct {
	id               string
	sourceEventBus   eventbus.SourceEventBus
	runningProviders map[string]protocol.SourceProvider
	providerMutex    sync.RWMutex
	logger           *slog.Logger
	persistence      persistence.Persistence
	registry         *registry.Registry
	restartCount     int
	providerFilter   []string
}

func NewSourceProviderManager(
	id string,
	persistence persistence.Persistence,
	sourceEventBus eventbus.SourceEventBus,
	logger *slog.Logger,
	registry *registry.Registry,
	providerFilter []string,
) *SourceProviderManager {
	return &SourceProviderManager{
		id:               id,
		logger:           logger.With("module", "operion-source-manager", "manager_id", id),
		persistence:      persistence,
		registry:         registry,
		restartCount:     0,
		sourceEventBus:   sourceEventBus,
		runningProviders: make(map[string]protocol.SourceProvider),
		providerFilter:   providerFilter,
	}
}

func (spm *SourceProviderManager) Start(ctx context.Context) {
	spmCtx, cancel := context.WithCancel(ctx)
	spm.logger.Info("Starting source provider manager")

	spm.handleSignals(spmCtx, cancel)
	spm.run(ctx, cancel)
}

func (spm *SourceProviderManager) handleSignals(ctx context.Context, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		spm.logger.Info("Received signal", "signal", sig)

		switch sig {
		case syscall.SIGHUP:
			spm.logger.Info("Reloading configuration...")
			spm.restart(ctx, cancel)
		case syscall.SIGINT, syscall.SIGTERM:
			spm.logger.Info("Shutting down gracefully...")
			spm.stop(cancel)
			os.Exit(0)
		default:
			spm.logger.Warn("Unhandled signal received", "signal", sig)
		}
	}()
}

func (spm *SourceProviderManager) restart(ctx context.Context, cancel context.CancelFunc) {
	spm.restartCount++
	spmCtx := context.WithoutCancel(ctx)
	spm.stop(cancel)

	if spm.restartCount > 5 {
		spm.logger.Error("Restart limit reached, exiting...")
		os.Exit(1)
	}

	backoff := time.Duration(spm.restartCount) * time.Second
	spm.logger.Info("Restarting source provider manager...", "backoff", backoff)
	time.Sleep(backoff)

	spm.Start(spmCtx)
}

func (spm *SourceProviderManager) run(ctx context.Context, cancel context.CancelFunc) {
	// First, create schedules from workflows
	if err := spm.createSchedulesFromWorkflows(ctx); err != nil {
		spm.logger.Error("Failed to create schedules from workflows", "error", err)
		spm.restart(ctx, cancel)
		return
	}

	// Then start source providers
	if err := spm.startSourceProviders(ctx); err != nil {
		spm.logger.Error("Failed to start source providers", "error", err)
		spm.restart(ctx, cancel)
		return
	}

	spm.logger.Info("Source provider manager started successfully")

	// Keep running until context is cancelled
	<-ctx.Done()

	spm.logger.Info("Source provider manager stopped")
}

func (spm *SourceProviderManager) createSchedulesFromWorkflows(ctx context.Context) error {
	workflowRepo := workflow.NewRepository(spm.persistence)

	workflows, err := workflowRepo.FetchAll(ctx)
	if err != nil {
		return err
	}

	scheduleCount := 0

	for _, wf := range workflows {
		if wf.Status != models.WorkflowStatusActive {
			continue
		}

		for _, trigger := range wf.WorkflowTriggers {
			// Check if this trigger needs a schedule (scheduler source provider)
			if cronExpr, exists := trigger.Configuration["cron_expression"]; exists {
				sourceID := trigger.SourceID
				if sourceID == "" {
					spm.logger.Warn("Trigger has cron_expression but no source_id",
						"workflow_id", wf.ID,
						"trigger_id", trigger.ID)
					continue
				}

				// Check if schedule already exists
				existingSchedule, err := spm.persistence.ScheduleBySourceID(sourceID)
				if err != nil {
					spm.logger.Error("Failed to check existing schedule",
						"source_id", sourceID,
						"error", err)
					continue
				}

				if existingSchedule != nil {
					spm.logger.Debug("Schedule already exists", "source_id", sourceID)
					continue
				}

				// Create new schedule
				cronStr, ok := cronExpr.(string)
				if !ok {
					spm.logger.Warn("Invalid cron_expression type",
						"source_id", sourceID,
						"type", cronExpr)
					continue
				}

				schedule, err := models.NewSchedule(sourceID, sourceID, cronStr)
				if err != nil {
					spm.logger.Error("Failed to create schedule",
						"source_id", sourceID,
						"cron", cronStr,
						"error", err)
					continue
				}

				if err := spm.persistence.SaveSchedule(schedule); err != nil {
					spm.logger.Error("Failed to save schedule",
						"source_id", sourceID,
						"error", err)
					continue
				}

				scheduleCount++

				spm.logger.Info("Created schedule",
					"source_id", sourceID,
					"cron", cronStr,
					"next_due_at", schedule.NextDueAt)
			}
		}
	}

	spm.logger.Info("Schedule creation completed", "created_count", scheduleCount)
	return nil
}

func (spm *SourceProviderManager) startSourceProviders(ctx context.Context) error {
	// Get available source providers from registry
	availableProviders := spm.registry.GetAvailableSourceProviders()

	var providersToStart []protocol.SourceProviderFactory

	if len(spm.providerFilter) == 0 {
		// No filter - start all available providers
		providersToStart = availableProviders
	} else {
		// Filter providers based on configuration
		for _, factory := range availableProviders {
			for _, filterName := range spm.providerFilter {
				if factory.ID() == filterName {
					providersToStart = append(providersToStart, factory)

					break
				}
			}
		}
	}

	spm.logger.Info("Starting source providers",
		"available_count", len(availableProviders),
		"filtered_count", len(providersToStart),
		"filter", spm.providerFilter)

	var wg sync.WaitGroup
	for _, factory := range providersToStart {
		wg.Add(1)

		go func(factory protocol.SourceProviderFactory) {
			defer wg.Done()

			if err := spm.startSourceProvider(ctx, factory); err != nil {
				spm.logger.Error("Failed to start source provider",
					"provider_id", factory.ID(),
					"error", err)
			}
		}(factory)
	}

	wg.Wait()
	return nil
}

func (spm *SourceProviderManager) startSourceProvider(ctx context.Context, factory protocol.SourceProviderFactory) error {
	providerID := factory.ID()

	// Create provider configuration
	var sourceConfigs []map[string]any

	if providerID == SchedulerProviderType {
		// Create single orchestrator instance (no source-specific configuration)
		config := map[string]any{
			"persistence": spm.persistence,
		}
		sourceConfigs = []map[string]any{config}
	}
	// Add other provider types as needed

	if len(sourceConfigs) == 0 {
		spm.logger.Info("No configuration found for provider", "provider_id", providerID)
		return nil
	}

	// Start a provider instance for each source configuration
	for _, config := range sourceConfigs {
		// Generate provider instance key (scheduler uses orchestrator key, others use source_id)
		var instanceKey string
		if providerID == SchedulerProviderType {
			instanceKey = SchedulerProviderType + "-orchestrator"
		} else {
			sourceID, ok := config["source_id"].(string)
			if !ok {
				spm.logger.Error("Missing source_id in config", "provider_id", providerID)
				continue
			}

			instanceKey = sourceID
		}

		provider, err := spm.registry.CreateSourceProvider(ctx, providerID, config)
		if err != nil {
			spm.logger.Error("Failed to create source provider",
				"provider_id", providerID,
				"instance_key", instanceKey,
				"error", err)
			continue
		}

		spm.providerMutex.Lock()
		spm.runningProviders[instanceKey] = provider
		spm.providerMutex.Unlock()

		// Create callback - scheduler orchestrator handles all sources, others handle specific source
		var callback protocol.SourceEventCallback
		if providerID == SchedulerProviderType {
			callback = spm.createSourceEventCallback("") // Empty source_id for orchestrator
		} else {
			callback = spm.createSourceEventCallback(instanceKey)
		}

		if err := provider.Start(ctx, callback); err != nil {
			spm.logger.Error("Failed to start source provider",
				"provider_id", providerID,
				"instance_key", instanceKey,
				"error", err)

			spm.providerMutex.Lock()
			delete(spm.runningProviders, instanceKey)
			spm.providerMutex.Unlock()
			continue
		}

		spm.logger.Info("Started source provider",
			"provider_id", providerID,
			"instance_key", instanceKey)
	}

	return nil
}

func (spm *SourceProviderManager) createSourceEventCallback(sourceID string) protocol.SourceEventCallback {
	return func(ctx context.Context, sourceID, providerType, eventType string, data map[string]any) error {
		logger := spm.logger.With(
			"source_id", sourceID,
			"provider_type", providerType,
			"event_type", eventType)

		logger.Info("Source event received, publishing to source event bus")

		// Create SourceEvent
		sourceEvent := events.NewSourceEvent(sourceID, providerType, eventType, data)

		// Validate the event
		if err := sourceEvent.Validate(); err != nil {
			logger.Error("Invalid source event", "error", err)
			return err
		}

		// Publish to source event bus
		logger.Info("Publishing source event",
			"source_id", sourceEvent.SourceID,
			"provider_id", sourceEvent.ProviderID,
			"event_type", sourceEvent.EventType)

		if err := spm.sourceEventBus.PublishSourceEvent(ctx, sourceEvent); err != nil {
			logger.Error("Failed to publish source event", "error", err)
			return err
		}

		logger.Info("Successfully published source event to event bus")
		return nil
	}
}

func (spm *SourceProviderManager) stop(cancel context.CancelFunc) {
	spm.logger.Info("Stopping source provider manager")

	if cancel != nil {
		cancel()
	}

	spm.providerMutex.Lock()
	defer spm.providerMutex.Unlock()

	for sourceID, provider := range spm.runningProviders {
		spm.logger.Info("Stopping source provider", "source_id", sourceID)

		if err := provider.Stop(context.Background()); err != nil {
			spm.logger.Error("Error stopping source provider",
				"source_id", sourceID,
				"error", err)
		}
	}

	spm.runningProviders = make(map[string]protocol.SourceProvider)
	spm.logger.Info("All source providers stopped")
}
