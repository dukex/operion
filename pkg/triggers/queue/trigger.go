package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/redis/go-redis/v9"
)

type QueueTrigger struct {
	ID            string
	Provider      string
	Connection    map[string]string
	Queue         string
	ConsumerGroup string
	WorkflowId    string
	Enabled       bool
	
	client   redis.UniversalClient
	callback protocol.TriggerCallback
	logger   *slog.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewQueueTrigger(config map[string]interface{}, logger *slog.Logger) (*QueueTrigger, error) {
	id, _ := config["id"].(string)
	provider, _ := config["provider"].(string)
	if provider == "" {
		provider = "redis"
	}
	
	queue, _ := config["queue"].(string)
	consumerGroup, _ := config["consumer_group"].(string)
	workflowId, _ := config["workflow_id"].(string)
	
	connectionConfig, _ := config["connection"].(map[string]interface{})
	connection := make(map[string]string)
	for k, v := range connectionConfig {
		if str, ok := v.(string); ok {
			connection[k] = str
		}
	}

	trigger := &QueueTrigger{
		ID:            id,
		Provider:      provider,
		Connection:    connection,
		Queue:         queue,
		ConsumerGroup: consumerGroup,
		Enabled:       true,
		WorkflowId:    workflowId,
		stopCh:        make(chan struct{}),
		logger: logger.With(
			"module", "queue_trigger",
			"id", id,
			"provider", provider,
			"queue", queue,
			"workflow_id", workflowId,
		),
	}
	
	if err := trigger.Validate(); err != nil {
		return nil, err
	}
	
	return trigger, nil
}

func (t *QueueTrigger) Validate() error {
	if t.ID == "" {
		return errors.New("queue trigger ID is required")
	}
	if t.Queue == "" {
		return errors.New("queue trigger queue name is required")
	}
	if t.Provider != "redis" {
		return fmt.Errorf("unsupported queue provider: %s (only 'redis' is supported)", t.Provider)
	}
	return nil
}

func (t *QueueTrigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	if !t.Enabled {
		t.logger.Info("QueueTrigger is disabled.")
		return nil
	}
	
	t.logger.Info("Starting QueueTrigger")
	t.callback = callback
	
	if err := t.initializeClient(); err != nil {
		return fmt.Errorf("failed to initialize queue client: %w", err)
	}
	
	t.wg.Add(1)
	go t.consume(ctx)
	
	return nil
}

func (t *QueueTrigger) initializeClient() error {
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
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := t.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	t.logger.Info("Connected to Redis", "addr", addr, "db", db)
	return nil
}

func (t *QueueTrigger) parseDB(dbStr string) (int, error) {
	var db int
	_, err := fmt.Sscanf(dbStr, "%d", &db)
	return db, err
}

func (t *QueueTrigger) consume(ctx context.Context) {
	defer t.wg.Done()
	
	t.logger.Info("Starting queue consumer", "queue", t.Queue)
	
	for {
		select {
		case <-t.stopCh:
			t.logger.Info("Queue consumer stopped")
			return
		case <-ctx.Done():
			t.logger.Info("Context cancelled, stopping queue consumer")
			return
		default:
			if err := t.processMessage(ctx); err != nil {
				t.logger.Error("Error processing message", "error", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (t *QueueTrigger) processMessage(ctx context.Context) error {
	result, err := t.client.BLPop(ctx, 1*time.Second, t.Queue).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("failed to pop message from queue: %w", err)
	}
	
	if len(result) < 2 {
		return nil
	}
	
	message := result[1]
	t.logger.Info("Received message from queue", "message", message)
	
	var triggerData map[string]interface{}
	if err := json.Unmarshal([]byte(message), &triggerData); err != nil {
		triggerData = map[string]interface{}{
			"message":   message,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
	} else {
		if triggerData["timestamp"] == nil {
			triggerData["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		}
	}
	
	go func() {
		if err := t.callback(context.Background(), triggerData); err != nil {
			t.logger.Error("Error executing workflow for trigger", "error", err)
		}
	}()
	
	return nil
}

func (t *QueueTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping QueueTrigger", "id", t.ID)
	
	close(t.stopCh)
	t.wg.Wait()
	
	if t.client != nil {
		if err := t.client.Close(); err != nil {
			t.logger.Error("Error closing Redis client", "error", err)
		}
	}
	
	return nil
}