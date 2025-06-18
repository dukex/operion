# Operion Trigger Service

The `operion-trigger` CLI provides trigger management and event publishing for the Operion workflow automation system.

## Overview

The trigger service implements an event-driven architecture that separates trigger detection from workflow execution:

- **Trigger Service** (`operion-trigger`) - Monitors trigger conditions and publishes events
- **Worker Service** (`operion-worker`) - Executes workflows when triggered
- **Event Bus** - Facilitates communication between services

## Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│  Trigger        │    │   Event      │    │  Worker         │
│  Service        │───▶│   Bus        │───▶│  Service        │
│                 │    │              │    │                 │
│ • Cron triggers │    │ • GoChannel  │    │ • Executes      │
│ • Webhooks      │    │ • Kafka      │    │   workflows     │
│ • File watches  │    │ • RabbitMQ   │    │ • Actions       │
└─────────────────┘    └──────────────┘    └─────────────────┘
```

## Commands

### run
Starts the trigger service to monitor workflows and publish trigger events.

```bash
# Basic usage
./bin/operion-trigger run

# With custom service ID
./bin/operion-trigger run --trigger-id my-trigger-service

# Using Kafka as event bus
./bin/operion-trigger run --kafka

# Custom data path
./bin/operion-trigger run --data-path /path/to/workflows
```

### list
Lists all triggers found in workflows.

```bash
./bin/operion-trigger list
```

Example output:
```
Available Triggers:
==================

Workflow: Bitcoin Price Monitor (bitcoin-001)
Status: active
Triggers:
  - ID: trigger-001
    Type: schedule
    Config: map[cron:* * * * *]

Total triggers: 1
```

### validate
Validates trigger configurations to ensure they can be created successfully.

```bash
./bin/operion-trigger validate
```

Example output:
```
Trigger Validation Results:
===========================

Workflow: Bitcoin Price Monitor (bitcoin-001)
  Trigger: trigger-001 (schedule)
    ✅ VALID

Validation Summary:
  Total triggers: 1
  Valid triggers: 1
  Invalid triggers: 0
All triggers are valid! ✅
```

## Event Flow

1. **Trigger Detection**: Service monitors configured triggers (cron schedules, webhooks, etc.)
2. **Event Publishing**: When triggered, publishes `WorkflowTriggered` event to event bus
3. **Event Processing**: Worker services receive events and execute corresponding workflows
4. **Lifecycle Events**: Workers publish additional events (`WorkflowStarted`, `WorkflowFinished`, etc.)

## Event Types

### WorkflowTriggered
Published when a trigger condition is met:

```json
{
  "id": "event-12345678",
  "type": "workflow.triggered",
  "timestamp": "2024-01-15T10:30:00Z",
  "workflow_id": "bitcoin-001",
  "trigger_id": "trigger-001", 
  "trigger_type": "schedule",
  "trigger_data": {
    "triggered_at": "2024-01-15T10:30:00Z",
    "cron": "* * * * *"
  }
}
```

## Configuration

### Environment Variables

- `DATA_PATH` - Path to workflow data directory (default: `./data/workflows`)
- `KAFKA_BROKERS` - Kafka broker addresses for event bus (default: `kafka:9092`)

### Event Bus Options

- **GoChannel** (default) - In-memory event bus for single-node setups
- **Kafka** - Distributed event bus for multi-node deployments
- **RabbitMQ** - Alternative message broker (planned)

## Usage Examples

### Running with Docker Compose

```yaml
version: '3.8'
services:
  trigger-service:
    image: operion:latest
    command: ["./bin/operion-trigger", "run", "--kafka"]
    environment:
      - KAFKA_BROKERS=kafka:9092
      - DATA_PATH=/data/workflows
    volumes:
      - ./data:/data
    depends_on:
      - kafka

  worker-service:
    image: operion:latest
    command: ["./bin/operion-worker", "run", "--kafka"]
    environment:
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - kafka
      - trigger-service
```

### Development Setup

```bash
# Terminal 1: Start trigger service
./bin/operion-trigger run

# Terminal 2: Start worker service  
./bin/operion-worker run

# Terminal 3: Monitor logs and validate
./bin/operion-trigger validate
./bin/operion-trigger list
```

## Monitoring

The trigger service provides structured logging with configurable levels:

```
INFO[2024-01-15T10:30:00Z] Starting trigger service
INFO[2024-01-15T10:30:00Z] Found 3 workflows
INFO[2024-01-15T10:30:00Z] Starting triggers for workflow workflow_id=bitcoin-001
INFO[2024-01-15T10:30:00Z] Started trigger successfully trigger_id=trigger-001
INFO[2024-01-15T10:31:00Z] Trigger fired, publishing event trigger_id=trigger-001
INFO[2024-01-15T10:31:00Z] Successfully published trigger event event_id=event-12345678
```

## Error Handling

The service handles various error conditions gracefully:

- **Invalid trigger configurations** - Logged and skipped, service continues
- **Event bus connection failures** - Automatic retry with exponential backoff
- **Workflow parsing errors** - Individual workflows skipped, others continue
- **Signal handling** - Graceful shutdown on SIGINT/SIGTERM

## Performance Considerations

- **Trigger polling** - Configurable intervals to balance responsiveness and resource usage
- **Event batching** - Multiple triggers can fire simultaneously without blocking
- **Memory usage** - Stateless design with minimal memory footprint
- **Horizontal scaling** - Multiple trigger services can run in parallel with proper coordination