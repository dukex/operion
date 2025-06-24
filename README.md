# Operion

A workflow automation system built in Go that allows creating event-driven workflows with configurable triggers and actions.

## Overview

Operion enables you to create automated workflows through:

- **Triggers**: Events that initiate workflows (scheduled execution)
- **Actions**: Operations executed in workflows (HTTP requests, file operations, logging, data transformation)
- **Conditionals**: Logic for flow control between steps
- **Context**: Data sharing between workflow steps
- **Workers**: Background processes that execute workflows

![image](https://github.com/user-attachments/assets/8dfd67d2-fe4b-4196-ab11-3d931ee2f90c)


## Features

- **Extensible** - Plugin system with dynamic .so file loading for triggers and actions
- **REST API** - HTTP interface for managing workflows
- **CLI Tools** - Command-line interfaces for dispatcher and worker services
- **File-based Storage** - Simple JSON persistence
- **Event-Driven** - Decoupled architecture with pub/sub messaging
- **Worker Management** - Background execution with proper lifecycle management
- **Concurrent Execution** - Efficient resource usage

## Architecture

The project follows a clean, layered architecture with clear separation of concerns:

- **Models** (`pkg/models/`) - Core domain models and interfaces
- **Business Logic** (`pkg/workflow/`) - Workflow execution and management
- **Infrastructure** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Plugin system for actions and triggers with .so file loading
- **Interface Layer** (`cmd/`) - Entry points (API server, CLI tool)

## Installation

### Prerequisites

- Go 1.24 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/dukex/operion.git
cd operion

# Download dependencies
go mod download

# Build all components
make build
```

### Configuration

Set the port via environment variable (defaults to 3000):

```bash
PORT=3000
```

## Usage

### Start the API Server

```bash
# For development (with live reload)
air

# Or run the built binary
./bin/api
```

The API will be available at `http://localhost:3000`

### Start the Visual Editor

```bash
# Navigate to the UI directory
cd ui/operion-editor

# Install dependencies (first time only)
npm install

# Start the development server
npm run dev
```

The visual workflow editor will be available at `http://localhost:5173`

### Start Services

#### Dispatcher Service (Event Publishers)
```bash
# Start dispatcher service to listen for triggers and publish events
./bin/operion-dispatcher run --database-url ./data/workflows --event-bus gochannel

# Start with custom dispatcher ID and plugins
./bin/operion-dispatcher run --dispatcher-id my-dispatcher --database-url ./data/workflows --event-bus kafka --plugins-path ./plugins

# List all available triggers
./bin/operion-dispatcher list

# Validate trigger configurations
./bin/operion-dispatcher validate
```

#### Worker Service (Workflow Execution)
```bash
# Start workers to execute workflows
./bin/operion-worker run

# Start workers with custom worker ID
./bin/operion-worker run --worker-id my-worker
```

#### Event-Driven Architecture
The system uses an event-driven architecture where:
1. **Dispatcher Service** loads trigger plugins, listens for trigger conditions and publishes `WorkflowTriggered` events
2. **Worker Service** handles workflow events and executes steps individually:
   - Receives `WorkflowTriggered` events and starts workflow execution
   - Processes `WorkflowStepAvailable` events for step-by-step execution
   - Publishes granular events: `WorkflowStepFinished`, `WorkflowStepFailed`, `WorkflowFinished`
3. **Event Bus** decouples trigger detection from workflow execution (supports GoChannel and Kafka)
4. **Native Actions** - Core actions built into binary for better performance
5. **Plugin System** enables dynamic loading of .so files for extensible triggers and custom actions

### API Endpoints

```bash
# List all workflows
curl http://localhost:3000/workflows

# Health check
curl http://localhost:3000/
```

### Example Workflow

See `./examples/data/workflows/bitcoin-price.json` for a complete workflow example that:
- Triggers every minute via cron schedule (`schedule` trigger)
- Fetches Bitcoin price data from CoinPaprika API (`http_request` action)
- Processes the data using JSONata transformation (`transform` action)
- Posts processed data to webhook endpoint (`http_request` action)
- Logs errors if any step fails (`log` action)

#### New Action Contract
Actions now use a standardized contract with:
- **Factory Pattern**: Actions created via `ActionFactory.Create(config)`
- **Execution Context**: Access to previous step results via `ExecutionContext.StepResults`
- **Template Support**: JSONata templating for dynamic configuration
- **Structured Logging**: Each action receives a structured logger
- **Result Mapping**: Step results stored by `uid` for cross-step references

## Current Implementation

### Available Triggers
- **Schedule** (`pkg/triggers/schedule/`) - Cron-based execution using robfig/cron with native implementation
- **Kafka** (`pkg/triggers/kafka/`) - Message-based triggering from Kafka topics with native implementation

### Available Actions  
- **HTTP Request** (`pkg/actions/http_request/`) - Make HTTP calls with retry logic, templating, and JSON/string response handling
- **Transform** (`pkg/actions/transform/`) - Process data using JSONata expressions with input extraction and templating
- **Log** (`pkg/actions/log/`) - Output structured log messages for debugging and monitoring
- **Plugin Actions**: Custom actions via .so plugins (example in `examples/plugins/actions/log/`)

### Plugin System
- Dynamic loading of `.so` plugin files from `./plugins` directory
- Factory pattern with `ActionFactory` and `TriggerFactory` interfaces
- Protocol-based interfaces in `pkg/protocol/` for type safety
- Example plugins available in `examples/plugins/`
- **Native vs Plugin Actions**: Core actions built-in for performance, plugins for extensibility

### Workflow Execution Model
The executor now operates on an event-driven, step-by-step model:
- **Execution Context**: Maintains state across steps with `ExecutionContext.StepResults`
- **Step Isolation**: Each step processed as individual event for scalability
- **Event Publishing**: Granular events published for monitoring and debugging
- **State Management**: Step results stored by `uid` and accessible via JSONata templates
- **Error Handling**: Failed steps can route to different next steps via `on_failure`

## Development

### Build Commands

```bash
make build          # Build API server for current platform
make build-linux    # Cross-compile for Linux  
make clean          # Clean build artifacts
```

### Development Server

```bash
air                 # Start development server with live reload
./bin/api           # Run built API server directly
```

### Plugin Development

```bash
# Build action plugin example
cd examples/plugins/actions/log
make

# Build custom plugin
# Create plugin.go implementing protocol.ActionFactory or protocol.TriggerFactory
# Export symbol: var Action protocol.ActionFactory = &MyActionFactory{}
go build -buildmode=plugin -o plugin.so plugin.go
```

## Future Features

### Planned Triggers
- **Kafka**: Listen to Kafka topics for events (interface exists)
- **Webhook**: Receive HTTP requests to trigger workflows
- **File System**: Watch for file changes

### Planned Actions
- **Slack**: Send messages to Slack channels
- **Email**: Send email notifications
- **Database**: Execute database operations
- **Template**: Generate files from templates

### Available Interfaces
- **Visual Workflow Editor**: React-based browser interface for visualizing and editing workflows
- **Web Dashboard**: Browser-based workflow editor and monitor

### Planned Interfaces
- **YAML Configuration**: Define workflows in YAML format
- **Extended CLI**: Additional workflow management commands
