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

Uses plugin-based system for extensibility:
- **Registry** - Plugin registry for both actions and triggers in `pkg/registry/`
- Dynamic loading of `.so` plugin files from filesystem
- Factory pattern with `ActionFactory` and `TriggerFactory` interfaces
- Protocol-based interfaces in `pkg/protocol/` for actions and triggers
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

### Visual Editor Development
```bash
cd ui/operion-editor    # Navigate to UI directory
npm install             # Install dependencies (first time)
npm run dev             # Start development server on port 5173
npm run build           # Build for production
npm run lint            # Run ESLint
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
- **CLI Dispatcher Service** (`cmd/operion-dispatcher/`) - Trigger listener and event publisher (replaces operion-trigger)
- **Visual Workflow Editor** (`ui/operion-editor/`) - React-based browser interface for workflow visualization
- **Domain Models** (`pkg/models/`) - Core workflow, action, and trigger models
- **Workflow Engine** (`pkg/workflow/`) - Workflow execution, management, and repository
- **Event System** (`pkg/event_bus/`, `pkg/events/`) - Event-driven communication
- **Plugin Registry** (`pkg/registry/`) - Plugin-based system for actions and triggers with .so file loading
- **File Persistence** (`pkg/persistence/file/`) - JSON file storage

### Available Triggers
- **Schedule Trigger** (`pkg/triggers/schedule/`) - Cron-based scheduling with robfig/cron
- **Kafka Trigger** (`pkg/triggers/kafka/`) - Kafka topic-based triggering with IBM/sarama

### Available Actions
- Actions will be registered as they are migrated to the new architecture

### Incomplete/Placeholder Components
- **Dashboard** (`cmd/dashboard/`) - Directory exists but not implemented (replaced by React-based UI)
- **Actions** - Need to be migrated to new pkg structure and registered
- **Visual Editor Editing** - Current UI is read-only, workflow creation/editing features need implementation

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

To add new triggers: Implement `protocol.Trigger` and `protocol.TriggerFactory` interfaces, compile as `.so` plugin
To add new actions: Implement `protocol.Action` and `protocol.ActionFactory` interfaces, compile as `.so` plugin
To add new persistence: Implement `persistence.Persistence` interface

### Plugin Development

```go
// Action plugin example
type MyActionFactory struct{}

func (f *MyActionFactory) Type() string {
    return "my-action"
}

func (f *MyActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
    return NewMyAction(config)
}

// Export symbol for plugin loading
var Action protocol.ActionFactory = &MyActionFactory{}

// Trigger plugin example  
type MyTriggerFactory struct{}

func (f *MyTriggerFactory) ID() string {
    return "my-trigger"
}

func (f *MyTriggerFactory) Create(config map[string]interface{}, logger *slog.Logger) (protocol.Trigger, error) {
    return NewMyTrigger(config, logger)
}

// Export symbol for plugin loading
var Trigger protocol.TriggerFactory = &MyTriggerFactory{}
```

### Plugin Registry Usage

```go
// Create registry and load plugins
registry := NewRegistry(logger)
actions, _ := registry.LoadActionPlugins("./plugins")
triggers, _ := registry.LoadTriggerPlugins("./plugins")

// Register loaded plugins
for _, action := range actions {
    registry.RegisterAction(action)
}
for _, trigger := range triggers {
    registry.RegisterTrigger(trigger)
}

// Create instances
action, err := registry.CreateAction("my-action", map[string]interface{}{
    "param": "value",
})
trigger, err := registry.CreateTrigger("my-trigger", config)
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

### Dispatcher Management
```bash
# Start dispatcher service (listens for triggers and publishes events)
./bin/operion-dispatcher run --database-url ./data/workflows --event-bus gochannel

# Start dispatcher with custom ID and plugins
./bin/operion-dispatcher run --dispatcher-id my-dispatcher --database-url ./data/workflows --event-bus kafka --plugins-path ./plugins

# List all triggers in workflows
./bin/operion-dispatcher list

# Validate trigger configurations
./bin/operion-dispatcher validate
```

## Architecture Overview

The system is designed with clear separation of concerns:

- **Dispatcher Service** (`operion-dispatcher`) - Loads trigger plugins, listens to external triggers and publishes `WorkflowTriggered` events
- **Worker Service** (`operion-worker`) - Subscribes to `WorkflowTriggered` events and executes workflows with action plugins
- **Event Bus** - Decouples trigger detection from workflow execution (supports GoChannel and Kafka)
- **Plugin System** - Dynamic loading of `.so` files for extensible actions and triggers

## Event Flow

1. **Dispatcher Service** loads trigger plugins and detects trigger conditions (cron, webhook, etc.)
2. **Dispatcher Service** publishes `WorkflowTriggered` event to event bus
3. **Worker Service** receives event and executes corresponding workflow using action plugins
4. **Worker Service** publishes workflow lifecycle events (`WorkflowStarted`, `WorkflowFinished`, etc.)

The CLI tools provide:
- Background workflow execution and trigger management
- Event-driven architecture with pub/sub messaging
- Plugin-based extensibility with .so file loading
- Signal handling for graceful shutdown (SIGHUP for restart, SIGINT/SIGTERM for shutdown)
- Structured logging with slog

## Claude Guidance

- Never use emoji at documentation