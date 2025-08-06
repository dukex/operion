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
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/registry"
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
			spm.stop(ctx, cancel)
			os.Exit(0)
		default:
			spm.logger.Warn("Unhandled signal received", "signal", sig)
		}
	}()
}

func (spm *SourceProviderManager) restart(ctx context.Context, cancel context.CancelFunc) {
	spm.restartCount++
	spmCtx := context.WithoutCancel(ctx)
	spm.stop(spmCtx, cancel)

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
	// Start source providers with lifecycle management
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

	// Create default empty configuration - providers handle their own setup
	config := map[string]any{}
	sourceConfigs := []map[string]any{config}

	// Start a provider instance for each source configuration
	for _, config := range sourceConfigs {
		// Generate provider instance key using provider ID
		instanceKey := providerID

		provider, err := spm.registry.CreateSourceProvider(ctx, providerID, config)
		if err != nil {
			spm.logger.Error("Failed to create source provider",
				"provider_id", providerID,
				"instance_key", instanceKey,
				"error", err)

			continue
		}

		// Execute complete lifecycle if provider supports it
		if lifecycle, ok := provider.(protocol.ProviderLifecycle); ok {
			deps := protocol.Dependencies{
				Logger: spm.logger,
			}

			// Step 1: Initialize dependencies
			if err := lifecycle.Initialize(ctx, deps); err != nil {
				spm.logger.Error("Failed to initialize provider",
					"provider_id", providerID,
					"instance_key", instanceKey,
					"error", err)

				continue
			}

			// Step 2: Configure with current workflows
			workflows, err := spm.persistence.Workflows(ctx)
			if err != nil {
				spm.logger.Error("Failed to get workflows for configuration",
					"provider_id", providerID,
					"instance_key", instanceKey,
					"error", err)

				continue
			}

			if err := lifecycle.Configure(workflows); err != nil {
				spm.logger.Error("Failed to configure provider",
					"provider_id", providerID,
					"instance_key", instanceKey,
					"error", err)

				continue
			}

			// Step 3: Prepare for startup
			if err := lifecycle.Prepare(ctx); err != nil {
				spm.logger.Error("Failed to prepare provider",
					"provider_id", providerID,
					"instance_key", instanceKey,
					"error", err)

				continue
			}
		}

		spm.providerMutex.Lock()
		spm.runningProviders[instanceKey] = provider
		spm.providerMutex.Unlock()

		// Create callback for the provider
		callback := spm.createSourceEventCallback("")

		// Step 4: Start the provider
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

func (spm *SourceProviderManager) stop(ctx context.Context, cancel context.CancelFunc) {
	spm.logger.Info("Stopping source provider manager")

	if cancel != nil {
		cancel()
	}

	spm.providerMutex.Lock()
	defer spm.providerMutex.Unlock()

	for sourceID, provider := range spm.runningProviders {
		spm.logger.Info("Stopping source provider", "source_id", sourceID)

		if err := provider.Stop(ctx); err != nil {
			spm.logger.Error("Error stopping source provider",
				"source_id", sourceID,
				"error", err)
		}
	}

	spm.runningProviders = make(map[string]protocol.SourceProvider)
	spm.logger.Info("All source providers stopped")
}
