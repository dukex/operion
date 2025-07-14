# Queue Trigger

The Queue Trigger allows workflows to be triggered by messages from a message queue. Currently supports Redis as the queue provider.

## Configuration

### Basic Configuration

```json
{
  "trigger_id": "queue",
  "configuration": {
    "provider": "redis",
    "queue": "task_queue",
    "consumer_group": "operion_workers",
    "connection": {
      "addr": "localhost:6379",
      "password": "",
      "db": "0"
    }
  }
}
```

### Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `provider` | string | No | "redis" | Queue provider (currently only "redis" supported) |
| `queue` | string | Yes | - | Name of the queue to consume from |
| `consumer_group` | string | No | - | Consumer group identifier |
| `connection` | object | No | - | Connection configuration |
| `connection.addr` | string | No | "localhost:6379" | Redis server address |
| `connection.password` | string | No | "" | Redis password |
| `connection.db` | string | No | "0" | Redis database number |

## Usage

### 1. Redis Setup

Make sure you have Redis running:

```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:alpine

# Or install locally
brew install redis
redis-server
```

### 2. Push Messages to Queue

Messages can be pushed to the Redis queue using any Redis client:

```bash
# Using redis-cli
redis-cli LPUSH task_queue '{"task_id": "123", "post_id": "1", "action": "fetch"}'

# Using Redis client in other languages
# Python example:
# import redis
# r = redis.Redis()
# r.lpush('task_queue', '{"task_id": "123", "post_id": "1"}')
```

### 3. Message Format

Messages should be JSON-formatted strings. The queue trigger will:
- Parse JSON messages and pass them as trigger data
- For non-JSON messages, wrap them in a simple object with `message` and `timestamp` fields

### 4. Trigger Data

The trigger provides the following data to workflow steps:

- **JSON messages**: All fields from the parsed JSON, plus `timestamp` if not present
- **Plain text messages**: `{"message": "...", "timestamp": "..."}`

## Example Workflow

```json
{
  "id": "queue-task-processor",
  "name": "Redis Queue Task Processor",
  "workflow_triggers": [
    {
      "id": "redis-queue-trigger",
      "trigger_id": "queue",
      "configuration": {
        "provider": "redis",
        "queue": "task_queue",
        "connection": {
          "addr": "localhost:6379",
          "db": "0"
        }
      }
    }
  ],
  "steps": [
    {
      "id": "log-task",
      "action_id": "log",
      "configuration": {
        "message": "Processing task: {{ trigger.task_id }}",
        "level": "info"
      }
    },
    {
      "id": "process-data",
      "action_id": "http_request",
      "configuration": {
        "url": "https://api.example.com/process",
        "method": "POST",
        "body": {
          "task_id": "{{ trigger.task_id }}",
          "data": "{{ trigger.data }}"
        }
      }
    }
  ]
}
```

## Testing

### Unit Tests

```bash
go test ./pkg/triggers/queue/...
```

### Integration Testing

1. Start Redis:
   ```bash
   docker run -d --name redis-test -p 6379:6379 redis:alpine
   ```

2. Start the dispatcher:
   ```bash
   ./bin/operion-dispatcher run --database-url ./examples/data --event-bus gochannel
   ```

3. Start a worker:
   ```bash
   ./bin/operion-worker run
   ```

4. Push a test message:
   ```bash
   redis-cli LPUSH task_queue '{"task_id": "test-123", "post_id": "1"}'
   ```

## Use Cases

- **Task Queue Processing**: Process background jobs from a queue
- **Event-Driven Microservices**: React to events published by other services
- **Batch Processing**: Process items from a work queue
- **Integration Workflows**: Bridge between different systems using message queues

## Error Handling

- Connection failures are logged and the trigger will retry
- Invalid messages are wrapped in a simple format and processed
- Redis connection is automatically retried with exponential backoff
- The trigger gracefully handles shutdown signals

## Limitations

- Currently only supports Redis as a queue provider
- Uses simple BLPOP for message consumption (no advanced Redis Streams features)
- No built-in message acknowledgment or dead letter queue support
- Single consumer per trigger instance