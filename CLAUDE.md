# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Operion** is a workflow automation system built in Go that allows creating event-driven workflows with configurable triggers and actions. The system provides multiple interfaces: a REST API server, CLI tool, and planned web dashboard.

## Architecture

The project follows a clean, layered architecture with clear separation of concerns:

- **Models Layer** (`pkg/models/`) - Core domain models and interfaces
- **Business Logic** (`pkg/workflow/`) - Workflow execution and management
- **Infrastructure Layer** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Plugin system for actions and triggers
- **Interface Layer** (`cmd/`) - Entry points (API server, CLI tool)

### Key Domain Models

- **Workflow** - Contains triggers, steps, variables, and metadata
- **WorkflowStep** - Individual workflow steps with actions and conditionals
- **Action Interface** - Contract for executable actions (pluggable architecture)
- **Trigger Interface** - Contract for workflow triggers (extensible system)
- **ExecutionContext** - Carries state between workflow steps

### Plugin Architecture

Uses unified registry pattern for extensibility:
- **Registry** - Single registry for both actions and triggers in `pkg/registry/`
- Schema-based component registration with `RegisteredComponent`
- Type-safe factory pattern with generics
- JSON Schema validation for configurations
- Runtime configuration from `map[string]interface{}`

## Development Commands

### Build Commands
```bash
make build          # Build API server for current platform
make build-linux    # Cross-compile for Linux
make clean          # Clean build artifacts
```

### Development Server
```bash
air                 # Start development server with live reload (proxy on port 3001, app on port 3000)
./bin/api           # Run built API server directly
```

### Dependencies
```bash
go mod download     # Download dependencies
go mod tidy         # Clean up dependencies
```

## OpenTelemetry Observability

The system includes comprehensive OpenTelemetry tracing for excellent observability across all components.

### Tracing Configuration

Set the following environment variable to configure where traces are sent:
```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"  # For OTLP HTTP
# or
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"  # For OTLP gRPC
```

### Traced Operations

The following operations are fully instrumented with spans:

#### Trigger Service (`operion-trigger`)
- **Service Start**: `trigger_service.start` - Service initialization and workflow loading
- **Workflow Trigger Setup**: `trigger_service.start_workflow_triggers` - Setting up triggers for each workflow
- **Individual Trigger Start**: `trigger_service.start_trigger` - Starting each trigger instance
- **Trigger Fired**: `trigger_service.trigger_fired` - When a trigger fires and publishes an event

#### Worker Service (`operion-worker`)
- **Worker Start**: `worker.start` - Worker initialization and event subscription
- **Event Handling**: `worker.handle_workflow_triggered` - Processing workflow triggered events

#### Workflow Execution (`workflow-executor`)
- **Workflow Execution**: `workflow.execute` - Complete workflow execution
- **Step Execution**: `workflow.execute_step` - Individual workflow step execution
- **Action Execution**: `workflow.execute_action` - Individual action execution within steps

#### Event Bus (`event-bus`)
- **Event Publishing**: `event_bus.publish` - Publishing events to the message queue
- **Event Subscription**: `event_bus.subscribe` - Subscribing to events
- **Message Handling**: `event_bus.handle_message` - Processing individual messages

#### Registry (`registry`)
- **Action Creation**: `registry.create_action` - Creating action instances
- **Trigger Creation**: `registry.create_trigger` - Creating trigger instances

### Span Attributes

All spans include relevant attributes for filtering and analysis:

- `operion.workflow.id` - Workflow ID
- `operion.workflow.name` - Workflow name
- `operion.trigger.id` - Trigger ID
- `operion.trigger.type` - Trigger type (e.g., "schedule", "kafka")
- `operion.action.id` - Action ID
- `operion.action.type` - Action type
- `operion.step.id` - Step ID
- `operion.step.name` - Step name
- `operion.execution.id` - Execution ID
- `operion.event.id` - Event ID
- `operion.service.id` - Service ID (trigger service ID)
- `operion.worker.id` - Worker ID

### Example Trace Flow

A complete workflow execution will show the following trace hierarchy:

```
trigger_service.trigger_fired
└── event_bus.publish
    └── event_bus.handle_message
        └── worker.handle_workflow_triggered
            └── workflow.execute
                ├── workflow.execute_step (step 1)
                │   ├── registry.create_action
                │   └── workflow.execute_action
                ├── workflow.execute_step (step 2)
                │   ├── registry.create_action
                │   └── workflow.execute_action
                └── event_bus.publish (workflow finished event)
```

### Observability Benefits

With this tracing implementation, you can:

1. **End-to-End Visibility**: Track a workflow execution from trigger to completion
2. **Performance Analysis**: Identify bottlenecks in workflow execution
3. **Error Debugging**: See exactly where and why workflows fail
4. **Dependency Mapping**: Understand the flow between services
5. **Resource Monitoring**: Track execution times and resource usage
6. **Business Metrics**: Monitor workflow success rates and processing times

## Current Implementation Status

### Available Components
- **API Server** (`cmd/api/`) - Fiber-based REST API with workflows endpoint
- **CLI Worker** (`cmd/operion-worker/`) - Background workflow execution tool
- **CLI Trigger Service** (`cmd/operion-trigger/`) - Trigger listener and event publisher
- **Domain Models** (`pkg/models/`) - Core workflow, action, and trigger models
- **Workflow Engine** (`pkg/workflow/`) - Workflow execution, management, and repository
- **Event System** (`pkg/event_bus/`, `pkg/events/`) - Event-driven communication
- **Unified Registry** (`pkg/registry/`) - Schema-based action and trigger registration system
- **File Persistence** (`pkg/persistence/file/`) - JSON file storage

### Available Triggers
- **Schedule Trigger** (`pkg/triggers/schedule/`) - Cron-based scheduling with robfig/cron
- **Kafka Trigger** (`pkg/triggers/kafka/`) - Kafka topic-based triggering with IBM/sarama

### Available Actions
- Actions will be registered as they are migrated to the new architecture

### Incomplete/Placeholder Components
- **Dashboard** (`cmd/dashboard/`) - Directory exists but not implemented
- **Actions** - Need to be migrated to new pkg structure and registered

## Key Configuration

- **Port**: 3000 (configurable via `PORT` environment variable)
- **Data Storage**: `./data/workflows/` directory (file-based persistence with individual JSON files)
- **Go Version**: 1.24 with toolchain 1.24.4
- **Main Dependencies**: Fiber v2, validator/v10, robfig/cron, problems (RFC7807), urfave/cli/v3, sirupsen/logrus

## Development Patterns

- **Interface Segregation** - Small, focused interfaces
- **Dependency Injection** - Constructor-based injection throughout
- **Context Propagation** - Proper context.Context usage for cancellation
- **Structured Errors** - RFC7807 problem format for API responses
- **Validation** - Struct validation using validator/v10

## Extension Points

To add new triggers: Implement `models.Trigger` interface and register with `Registry.RegisterTrigger()`
To add new actions: Implement `models.Action` interface and register with `Registry.RegisterAction()`  
To add new persistence: Implement `persistence.Persistence` interface

### Registry Usage

```go
// Create registry
registry.RegisterAllComponent()

// Register action with schema
component := &models.RegisteredComponent{
    Type: "my-action",
    Name: "My Custom Action",
    Description: "Description of what this action does",
    Schema: &models.JSONSchema{
        Type: "object",
        Properties: map[string]*models.Property{
            "param": {Type: "string", Description: "Parameter description"},
        },
        Required: []string{"param"},
    },
}

registry.RegisterAction(component, func(config map[string]interface{}) (models.Action, error) {
    return NewMyAction(config)
})

// Create instance
action, err := registry.CreateAction("my-action", map[string]interface{}{
    "param": "value",
})
```

## API Endpoints

- `GET /` - Health check
- `GET /workflows` - List all workflows
- API runs on port 3000, development proxy on port 3001

## Sample Data

Sample workflows in `./data/workflows/` directory:
- `bitcoin-price.json` - Demonstrates Bitcoin price fetching with HTTP actions, data transformation, and cron scheduling
- `weather-pocrane.json` - Additional workflow example

## CLI Usage

### Worker Management
```bash
# Start workflow workers (execution)
./bin/operion-worker run

# Start workers with custom worker ID  
./bin/operion-worker run --worker-id my-worker
```

### Trigger Management
```bash
# Start trigger service (listens for triggers and publishes events)
./bin/operion-trigger run

# Start trigger service with custom ID
./bin/operion-trigger run --trigger-id my-trigger-service

# List all triggers in workflows
./bin/operion-trigger list

# Validate trigger configurations
./bin/operion-trigger validate

# Use Kafka as event bus
./bin/operion-trigger run --kafka
```

## Architecture Overview

The system is designed with clear separation of concerns:

- **Trigger Service** (`operion-trigger`) - Listens to external triggers and publishes `WorkflowTriggered` events
- **Worker Service** (`operion-worker`) - Subscribes to `WorkflowTriggered` events and executes workflows
- **Event Bus** - Decouples trigger detection from workflow execution (supports GoChannel and Kafka)

## Event Flow

1. **Trigger Service** detects trigger conditions (cron, webhook, etc.)
2. **Trigger Service** publishes `WorkflowTriggered` event to event bus
3. **Worker Service** receives event and executes corresponding workflow
4. **Worker Service** publishes workflow lifecycle events (`WorkflowStarted`, `WorkflowFinished`, etc.)

The CLI tools provide:
- Background workflow execution and trigger management
- Event-driven architecture with pub/sub messaging
- Signal handling for graceful shutdown  
- Structured logging with logrus
```

## Claude Guidance

- Never use emoji at documentation