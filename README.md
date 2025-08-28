# Operion

A cloud-native workflow automation platform built in Go that enables event-driven workflows with node-based execution and connections. Designed for Kubernetes deployments and following cloud-native principles.

## Overview

Operion enables you to create automated workflows through:

- **Source Providers**: Self-contained modules that generate events from external sources (scheduler, webhook, kafka)
- **Nodes**: Individual executable units in workflows (triggers, actions, conditionals, transforms)
- **Connections**: Links between node ports that define data flow and execution order
- **Context**: Data sharing between workflow nodes through execution context
- **Workers**: Background processes that execute workflows node-by-node
- **Source Manager**: Orchestrates source providers and manages their lifecycle
- **Activator**: Bridges source events to workflow executions

![image](https://github.com/user-attachments/assets/8dfd67d2-fe4b-4196-ab11-3d931ee2f90c)

## Features

- **Cloud-Native** - Stateless, container-first design optimized for Kubernetes
- **Event-Driven** - Decoupled architecture with pub/sub messaging for scalability
- **Node-Based Architecture** - Visual workflow creation with nodes and connections
- **Extensible** - Plugin system with dynamic .so file loading for nodes
- **REST API** - HTTP interface for managing workflows
- **CLI Tools** - Command-line interfaces for activator, source manager, and worker services
- **Multiple Storage Options** - File-based, PostgreSQL, and cloud storage support
- **Worker Management** - Background execution with proper lifecycle management
- **Horizontal Scaling** - Support for multiple instances and load balancing
- **Observability** - Built-in metrics, structured logging, and health checks

## Architecture

The project follows a clean, layered architecture with clear separation of concerns:

- **Models** (`pkg/models/`) - Core domain models and interfaces
- **Business Logic** (`pkg/workflow/`) - Workflow execution and management
- **Providers** (`pkg/providers/`) - Self-contained event generation modules with isolated persistence
- **Infrastructure** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Plugin system for nodes with .so file loading
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

#### API Server

The API server supports the following environment variables:

```bash
PORT=9091                    # API server port (default: 9091)
DATABASE_URL=./data/workflows  # Database connection URL or file path (required)
EVENT_BUS_TYPE=gochannel     # Event bus type: gochannel, kafka (required)
PLUGINS_PATH=./plugins       # Path to plugins directory (default: ./plugins)
LOG_LEVEL=info              # Log level: debug, info, warn, error (default: info)
```

#### Database Options

Operion supports multiple persistence backends:

**File-based Storage** (default):
```bash
DATABASE_URL=./data/workflows
```

**PostgreSQL Database**:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/operion
```

The PostgreSQL persistence layer includes:
- Normalized schema with separate tables for workflows, workflow_nodes, and workflow_connections
- JSONB storage for configuration data
- Automated schema migrations with version tracking
- Soft delete functionality
- Comprehensive indexing for performance

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
   - Examples: scheduler provider (`pkg/providers/scheduler/`), webhook provider (future)
2. **Source Manager Service** - Orchestrates source providers:
   - Manages provider lifecycle (Initialize → Configure → Prepare → Start)
   - Passes workflow definitions to providers during configuration
   - Publishes source events to event bus
3. **Activator Service** - Bridges source events to workflow executions:
   - Listens to source events from event bus  
   - Matches events to workflow triggers
   - Publishes `WorkflowTriggered` events for matched workflows
4. **Worker Service** - Executes workflows node-by-node:
   - Processes `WorkflowTriggered` events and `NodeActivation` events
   - Publishes granular events: `NodeExecutionFinished`, `NodeExecutionFailed`, `WorkflowFinished`

**Legacy Architecture (deprecated):**
1. **Direct workflow triggering** - Legacy trigger support (to be removed)

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

- **Trigger Node**: Scheduler node that triggers every minute via cron schedule
- **HTTP Request Node**: Fetches Bitcoin price data from CoinPaprika API
- **Transform Node**: Processes the data using Go template transformation
- **HTTP Request Node**: Posts processed data to webhook endpoint
- **Log Node**: Logs errors if any step fails
- **Connections**: Define the flow between nodes using source and target ports

#### Node-Based Architecture

Nodes use a standardized interface with:

- **Factory Pattern**: Nodes created via `NodeFactory.Create(config)`
- **Execution Context**: Access to previous node results via `ExecutionContext`
- **Input/Output Ports**: Structured connection points for data flow
- **Template Support**: Go template system for dynamic configuration
- **Structured Logging**: Each node receives a structured logger
- **Result Mapping**: Node results stored by ID for cross-node references

## Current Implementation

### Available Source Providers

- **Scheduler** (`pkg/providers/scheduler/`) - Self-contained cron-based scheduler with isolated persistence
  - Supports file-based persistence (`file://./data/scheduler`) or database persistence (future)
  - Manages its own schedule models and lifecycle
  - Configurable via `SCHEDULER_PERSISTENCE_URL` environment variable

### Available Nodes

#### Trigger Nodes
- **Scheduler** (`pkg/nodes/trigger/scheduler`) - Cron-based scheduling with robfig/cron
- **Kafka** (`pkg/nodes/trigger/kafka`) - Message-based triggering from Kafka topics
- **Webhook** (`pkg/nodes/trigger/webhook`) - HTTP endpoint triggers for external integrations

#### Action Nodes
- **HTTP Request** (`pkg/nodes/httprequest/`) - Make HTTP calls with retry logic, templating, and JSON/string response handling
- **Transform** (`pkg/nodes/transform/`) - Process data using Go templates
- **Log** (`pkg/nodes/log/`) - Output structured log messages for debugging and monitoring
- **Conditional** (`pkg/nodes/conditional/`) - Conditional branching based on data evaluation
- **Switch** (`pkg/nodes/switch/`) - Multi-path routing based on expression evaluation
- **Merge** (`pkg/nodes/merge/`) - Combine multiple input streams into single output


### Plugin System

- Dynamic loading of `.so` plugin files from `./plugins` directory
- Factory pattern with `NodeFactory` interfaces
- Protocol-based interfaces in `pkg/protocol/` for type safety
- Example plugins available in `examples/plugins/`
- **Native vs Plugin Nodes**: Core nodes built-in for performance, plugins for extensibility

### Workflow Execution Model

Operion operates on an event-driven, node-by-node execution model:

- **Execution Context**: Maintains state across nodes with `ExecutionContext`
- **Node Isolation**: Each node processed as individual event for scalability
- **Event Publishing**: Granular events published for monitoring and debugging
- **State Management**: Node results stored by ID and accessible via Go templates
- **Port-Based Routing**: Success and error outputs route through different ports to connected nodes

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
# Create plugin.go implementing protocol.NodeFactory
# Export symbol: var Node protocol.NodeFactory = &MyNodeFactory{}
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
