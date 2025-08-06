# Operion

A cloud-native workflow automation platform built in Go that enables event-driven workflows with configurable triggers and actions. Designed for Kubernetes deployments and following cloud-native principles.

## Overview

Operion enables you to create automated workflows through:

- **Source Providers**: Self-contained modules that generate events from external sources (scheduler, webhook, kafka)
- **Triggers**: Workflow trigger definitions that specify conditions for workflow execution
- **Actions**: Operations executed in workflows (HTTP requests, file operations, logging, data transformation)
- **Context**: Data sharing between workflow steps
- **Workers**: Background processes that execute workflows
- **Source Manager**: Orchestrates source providers and manages their lifecycle
- **Activator**: Bridges source events to workflow executions

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
- **Source Providers** (`pkg/sources/`) - Self-contained event generation modules with isolated persistence
- **Infrastructure** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Plugin system for actions and triggers with .so file loading
- **Interface Layer** (`cmd/`) - Entry points (API server, CLI tools, service managers)

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

#### Source Manager Service (Event Generation)

```bash
# Start source manager to run source providers (scheduler, webhook, etc.)
./bin/operion-source-manager --database-url file://./data --providers scheduler

# Start with custom configuration
SOURCE_MANAGER_ID=my-manager \
SCHEDULER_PERSISTENCE_URL=file://./data/scheduler \
./bin/operion-source-manager --database-url postgres://user:pass@localhost/db --providers scheduler,webhook

# Validate source provider configurations
./bin/operion-source-manager validate --database-url file://./data
```

#### Activator Service (Event Bridge)

```bash  
# Start activator to bridge source events to workflow executions
./bin/operion-activator --database-url file://./data

# Start with custom activator ID
./bin/operion-activator --activator-id my-activator --database-url postgres://user:pass@localhost/db
```

#### Dispatcher Service (Legacy Trigger Support)

```bash
# Start dispatcher service for legacy trigger support
./bin/operion-dispatcher --database-url ./data/workflows --event-bus gochannel

# Start with custom dispatcher ID and plugins
./bin/operion-dispatcher --dispatcher-id my-dispatcher --database-url ./data/workflows --event-bus kafka --plugins-path ./plugins

# Validate trigger configurations
./bin/operion-dispatcher validate
```

#### Worker Service (Workflow Execution)

```bash
# Start workers to execute workflows
./bin/operion-worker --database-url file://./data

# Start workers with custom worker ID  
./bin/operion-worker --worker-id my-worker --database-url postgres://user:pass@localhost/db
```

#### Event-Driven Architecture

The system uses a modern event-driven architecture with complete provider isolation:

**New Source-Based Architecture:**
1. **Source Providers** - Self-contained modules that generate events from external sources:
   - Each provider manages its own persistence and configuration
   - Completely isolated from core system (only receives workflow definitions)
   - Examples: scheduler provider (`pkg/sources/scheduler/`), webhook provider (future)
2. **Source Manager Service** - Orchestrates source providers:
   - Manages provider lifecycle (Initialize → Configure → Prepare → Start)
   - Passes workflow definitions to providers during configuration
   - Publishes source events to event bus
3. **Activator Service** - Bridges source events to workflow executions:
   - Listens to source events from event bus  
   - Matches events to workflow triggers
   - Publishes `WorkflowTriggered` events for matched workflows
4. **Worker Service** - Executes workflows step-by-step:
   - Processes `WorkflowTriggered` events and `WorkflowStepAvailable` events
   - Publishes granular events: `WorkflowStepFinished`, `WorkflowStepFailed`, `WorkflowFinished`

**Legacy Architecture (still supported):**
1. **Dispatcher Service** - Legacy trigger support with plugin loading
2. **Direct workflow triggering** - For backwards compatibility

**Benefits:**
- **Complete Isolation**: Source providers are self-contained modules
- **Pluggable Architecture**: Easy to add new event sources without core changes
- **Flexible Persistence**: Each provider can use different storage (file, database)
- **Scalable**: Source generation decoupled from workflow execution

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
- Processes the data using Go template transformation (`transform` action)
- Posts processed data to webhook endpoint (`http_request` action)
- Logs errors if any step fails (`log` action)

#### New Action Contract

Actions now use a standardized contract with:

- **Factory Pattern**: Actions created via `ActionFactory.Create(config)`
- **Execution Context**: Access to previous step results via `ExecutionContext.StepResults`
- **Template Support**: Go template system for dynamic configuration
- **Structured Logging**: Each action receives a structured logger
- **Result Mapping**: Step results stored by `uid` for cross-step references

## Current Implementation

### Available Source Providers

- **Scheduler** (`pkg/sources/scheduler/`) - Self-contained cron-based scheduler with isolated persistence
  - Supports file-based persistence (`file://./data/scheduler`) or database persistence (future)
  - Manages its own schedule models and lifecycle
  - Configurable via `SCHEDULER_PERSISTENCE_URL` environment variable

### Available Triggers (Legacy)

- **Schedule** (`pkg/triggers/schedule/`) - Cron-based execution using robfig/cron with native implementation
- **Kafka** (`pkg/triggers/kafka/`) - Message-based triggering from Kafka topics with native implementation  
- **Redis Queue** (`pkg/triggers/queue/`) - Redis-based queue consumption for task processing
- **Webhook** (`pkg/triggers/webhook/`) - HTTP endpoint triggers for external integrations

### Available Actions

- **HTTP Request** (`pkg/actions/http_request/`) - Make HTTP calls with retry logic, templating, and JSON/string response handling
- **Transform** (`pkg/actions/transform/`) - Process data using Go templates with templating
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
- **State Management**: Step results stored by `uid` and accessible via Go templates
- **Error Handling**: Failed steps can route to different next steps via `on_failure`

## Development

### Build Commands

```bash
make build          # Build API server for current platform
make build-linux    # Cross-compile for Linux
make clean          # Clean build artifacts
```

### Testing

```bash
make test           # Run all tests
make test-coverage  # Generate coverage report (coverage.out and coverage.html)
make fmt            # Format Go code
make lint           # Run golangci-lint
```

### CI/CD

The project uses GitHub Actions for continuous integration:

- **Test and Coverage**: Runs on every PR and push to main
  - Tests with Go 1.24
  - Generates coverage reports
  - Uploads coverage to Codecov and Coveralls
  - Runs static analysis (vet, staticcheck, golangci-lint)
  - Format checking
  - Builds all binaries

See [`.github/workflows/test-and-coverage.yml`](.github/workflows/test-and-coverage.yml) for complete workflow configuration.

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

### Infrastructure Enhancements
- **Kubernetes Integration**: Helm charts, HPA support, and service mesh compatibility
- **Enhanced Observability**: Prometheus metrics, Jaeger tracing, and health checks
- **Security Features**: OAuth2/OIDC, RBAC, secret management integration
- **Multi-tenancy**: Organization isolation and resource quotas

### Available Interfaces
- **Visual Workflow Editor**: React-based browser interface for visualizing and editing workflows
- **REST API**: Complete workflow management via HTTP endpoints
- **CLI Tools**: Command-line workflow and service management
