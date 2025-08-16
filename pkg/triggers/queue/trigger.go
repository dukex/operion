// Package queue provides message queue trigger implementation.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/operion-flow/interfaces"
	redis "github.com/redis/go-redis/v9"
)

type Provider int

const (
	RedisProvider Provider = iota
)

var providerName = map[Provider]string{
	RedisProvider: "redis",
}

func getProviderByName(name string) (Provider, error) {
	for p, n := range providerName {
		if n == name {
			return p, nil
		}
	}

	return 0, fmt.Errorf("unsupported queue provider: %s", name)
}

type Trigger struct {
	Provider      Provider
	Connection    map[string]string
	Queue         string
	ConsumerGroup string
	Enabled       bool

	client   redis.UniversalClient
	callback interfaces.TriggerCallback
	logger   *slog.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewTrigger(ctx context.Context, config map[string]any, logger *slog.Logger) (*Trigger, error) {
	provider, _ := config["provider"].(string)
	if provider == "" {
		provider = providerName[RedisProvider]
	}

	queue, _ := config["queue"].(string)
	consumerGroup, _ := config["consumer_group"].(string)

	connectionConfig, _ := config["connection"].(map[string]any)

	connection := make(map[string]string)
	for k, v := range connectionConfig {
		if str, ok := v.(string); ok {
			connection[k] = str
		}
	}

	providerEnum, err := getProviderByName(provider)
	if err != nil {
		return nil, err
	}

	trigger := &Trigger{
		Provider:      providerEnum,
		Connection:    connection,
		Queue:         queue,
		ConsumerGroup: consumerGroup,
		Enabled:       true,
		stopCh:        make(chan struct{}),
		logger: logger.With(
			"module", "queue_trigger",
			"provider", provider,
			"queue", queue,
		),
	}

	err = trigger.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *Trigger) Validate(_ context.Context) error {
	if t.Queue == "" {
		return errors.New("queue trigger queue name is required")
	}

	return nil
}

func (t *Trigger) Start(ctx context.Context, callback interfaces.TriggerCallback) error {
	if !t.Enabled {
		t.logger.InfoContext(ctx, "QueueTrigger is disabled.")

		return nil
	}

	t.logger.InfoContext(ctx, "Starting QueueTrigger")
	t.callback = callback

	err := t.initializeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize queue client: %w", err)
	}

	t.wg.Add(1)

	go t.consume(ctx)

	return nil
}

func (t *Trigger) initializeClient(ctx context.Context) error {
	addr := t.Connection["addr"]
	if addr == "" {
		addr = "localhost:6379"
	}

	password := t.Connection["password"]
	db := 0

	if dbStr := t.Connection["db"]; dbStr != "" {
		var err error
		if db, err = t.parseDB(dbStr); err != nil {
			return fmt.Errorf("invalid db value: %w", err)
		}
	}

	t.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := t.client.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	t.logger.InfoContext(ctx, "Connected to Redis", "addr", addr, "db", db)

	return nil
}

func (t *Trigger) parseDB(dbStr string) (int, error) {
	var db int

	_, err := fmt.Sscanf(dbStr, "%d", &db)

	return db, err
}

func (t *Trigger) consume(ctx context.Context) {
	defer t.wg.Done()

	t.logger.InfoContext(ctx, "Starting queue consumer", "queue", t.Queue)

	for {
		select {
		case <-t.stopCh:
			t.logger.InfoContext(ctx, "Queue consumer stopped")

			return
		case <-ctx.Done():
			t.logger.InfoContext(ctx, "Context cancelled, stopping queue consumer")

			return
		default:
			err := t.processMessage(ctx)
			if err != nil {
				t.logger.ErrorContext(ctx, "Error processing message", "error", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (t *Trigger) processMessage(ctx context.Context) error {
	result, err := t.client.BLPop(ctx, 1*time.Second, t.Queue).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}

		return fmt.Errorf("failed to pop message from queue: %w", err)
	}

	if len(result) < 2 {
		return nil
	}

	message := result[1]
	t.logger.InfoContext(ctx, "Received message from queue", "message", message)

	var triggerData map[string]any
	if err := json.Unmarshal([]byte(message), &triggerData); err != nil {
		triggerData = map[string]any{
			"message":   message,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
	} else {
		if triggerData["timestamp"] == nil {
			triggerData["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		}
	}

	go func() {
		err := t.callback(ctx, triggerData)
		if err != nil {
			t.logger.ErrorContext(ctx, "Error executing workflow for trigger", "error", err)
		}
	}()

	return nil
}

func (t *Trigger) Stop(ctx context.Context) error {
	t.logger.InfoContext(ctx, "Stopping QueueTrigger")

	close(t.stopCh)
	t.wg.Wait()

	if t.client != nil {
		err := t.client.Close()
		if err != nil {
			t.logger.ErrorContext(ctx, "Error closing Redis client", "error", err)
		}
	}

	return nil
}
