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

## Current Implementation Status

### Available Components
- **API Server** (`cmd/api/`) - Fiber-based REST API with workflows endpoint
- **CLI Worker** (`cmd/operion-worker/`) - Background workflow execution tool
- **Domain Models** (`pkg/models/`) - Core workflow, action, and trigger models
- **Workflow Engine** (`pkg/workflow/`) - Workflow execution, management, and repository
- **Event System** (`pkg/event_bus/`, `pkg/events/`) - Event-driven communication
- **Unified Registry** (`pkg/registry/`) - Schema-based action and trigger registration system
- **File Persistence** (`pkg/persistence/file/`) - JSON file storage

### Incomplete/Placeholder Components
- **Dashboard** (`cmd/dashboard/`) - Directory exists but not implemented
- **Kafka Trigger** (`pkg/triggers/kafka/`) - Interface defined but not fully implemented
- **Extended CLI Commands** - Only worker commands implemented

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
registry := registry.GetDefaultRegistry()

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
# Start workflow workers
./bin/operion-worker run

# Start workers with custom worker ID  
./bin/operion-worker run --worker-id my-worker
```

The CLI tool provides:
- Background workflow execution
- Worker lifecycle management  
- Signal handling for graceful shutdown
- Structured logging with logrus
```

## Claude Guidance

- Never use emoji at documentation