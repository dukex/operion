# Operion

A cloud-native workflow automation platform built in Go that enables event-driven workflows with configurable triggers and actions. Designed for Kubernetes deployments and following cloud-native principles.

## Overview

Operion enables you to create automated workflows through:

- **Triggers**: Events that initiate workflows (scheduled execution)
- **Actions**: Operations executed in workflows (HTTP requests, file operations, logging, data transformation)
- **Conditionals**: Logic for flow control between steps
- **Context**: Data sharing between workflow steps
- **Workers**: Background processes that execute workflows

![image](https://github.com/user-attachments/assets/8dfd67d2-fe4b-4196-ab11-3d931ee2f90c)

## Features

- **Cloud-Native** - Stateless, container-first design optimized for Kubernetes
- **Event-Driven** - Decoupled architecture with pub/sub messaging for scalability
- **Extensible** - Plugin system with dynamic .so file loading for triggers and actions
- **REST API** - HTTP interface for managing workflows
- **CLI Tools** - Command-line interfaces for dispatcher and worker services
- **Multiple Storage Options** - File-based, database, and cloud storage support
- **Worker Management** - Background execution with proper lifecycle management
- **Horizontal Scaling** - Support for multiple instances and load balancing
- **Observability** - Built-in metrics, structured logging, and health checks

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
- **Redis Queue** (`pkg/triggers/queue/`) - Redis-based queue consumption for task processing
- **Webhook** (`pkg/triggers/webhook/`) - HTTP endpoint triggers for external integrations

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

See [TODO.md](./TODO.md) for a comprehensive list of planned features organized by priority.

### High Priority Cloud-Native Features

- **RabbitMQ Trigger**: AMQP message consumption with enterprise features
- **AWS SQS Trigger**: Native AWS queue integration with FIFO support
- **Google Pub/Sub Trigger**: Google Cloud messaging integration
- **Email Action**: SMTP-based notifications for cloud environments
- **Slack/Discord Actions**: Team communication via webhooks
- **Database Actions**: Cloud database operations (PostgreSQL, MySQL, MongoDB)
- **Conditional Actions**: Workflow branching and business rule implementation

### Infrastructure Enhancements
- **Kubernetes Integration**: Helm charts, HPA support, and service mesh compatibility
- **Enhanced Observability**: Prometheus metrics, Jaeger tracing, and health checks
- **Security Features**: OAuth2/OIDC, RBAC, secret management integration
- **Multi-tenancy**: Organization isolation and resource quotas

### Available Interfaces
- **Visual Workflow Editor**: React-based browser interface for visualizing and editing workflows
- **REST API**: Complete workflow management via HTTP endpoints
- **CLI Tools**: Command-line workflow and service management
