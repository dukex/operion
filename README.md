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
- Docker and Docker Compose (for Kafka setup)

### Quick Start with Docker

```bash
# Clone the repository
git clone https://github.com/dukex/operion.git
cd operion

# Start Kafka and AKHQ UI
docker-compose up -d

# Build all components
make build

# Start all services
./run-services.sh
```

### Build from Source

```bash
# Download dependencies
go mod download

# Build all components
make build
```

### Configuration

Set the API port via environment variable (defaults to 8099):

```bash
PORT=8099
```

## Usage

### Start the API Server

```bash
# For development (with live reload)
air

# Or run the built binary
./bin/operion-api run --port 8099 --database-url file://./examples/data --event-bus kafka
```

The API will be available at `http://localhost:8099`

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

#### All Services (Recommended)

```bash
# Start all services with proper configuration
./run-services.sh
```

#### Individual Services

**Dispatcher Service (Trigger Processing)**
```bash
./bin/operion-dispatcher run --database-url file://./examples/data --event-bus kafka --webhook-port 8085 --receiver-config ./configs/receivers.yaml
```

**Worker Service (Workflow Execution)**
```bash
./bin/operion-worker run --database-url file://./examples/data --event-bus kafka
```

#### Event-Driven Architecture

The system uses a receiver pattern architecture with multi-topic event routing:

1. **Receivers** consume from external sources (Kafka topics, webhooks, schedules) → publish `TriggerEvent` to `operion.trigger`
2. **Dispatcher Service** subscribes to `operion.trigger` → matches triggers to workflows → publishes `WorkflowTriggered` to `operion.events`  
3. **Worker Service** subscribes to `operion.events` → executes workflow steps → publishes step events back to `operion.events`
4. **Event Bus** supports multiple Kafka topics for proper event routing and isolation
5. **Monitoring** via AKHQ UI at `http://localhost:8080` to view Kafka topics and messages

### API Endpoints

```bash
# List all workflows
curl http://localhost:8099/workflows

# List available actions with schemas
curl http://localhost:8099/registry/actions

# List available triggers with schemas  
curl http://localhost:8099/registry/triggers

# Health check
curl http://localhost:8099/
```

### Example Workflow

See `./examples/data/workflows/employee-logger-workflow.json` for a working example that:

- Triggers on Kafka messages from `tp_employee` topic
- Extracts employee data using JSONata (`transform` action)  
- Logs employee information (`log` action)
- Demonstrates proper step linking with `on_success` fields
- Shows correct JSONata syntax for accessing trigger and step data

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
