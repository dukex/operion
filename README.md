# Operion

A workflow automation system built in Go that allows creating event-driven workflows with configurable triggers and actions.

## Overview

Operion enables you to create automated workflows through:

- **Triggers**: Events that initiate workflows (scheduled execution)
- **Actions**: Operations executed in workflows (HTTP requests, file operations, logging, data transformation)
- **Conditionals**: Logic for flow control between steps
- **Context**: Data sharing between workflow steps
- **Workers**: Background processes that execute workflows

## Features

- **Extensible** - Plugin architecture for adding new triggers and actions
- **REST API** - HTTP interface for managing workflows
- **CLI Tool** - Command-line interface for running workflow workers
- **File-based Storage** - Simple JSON persistence
- **Worker Management** - Background execution with proper lifecycle management
- **Concurrent Execution** - Efficient resource usage

## Architecture

The project follows a clean, layered architecture with clear separation of concerns:

- **Models** (`pkg/models/`) - Core domain models and interfaces
- **Business Logic** (`pkg/workflow/`) - Workflow execution and management
- **Infrastructure** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Unified registry system for actions and triggers
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

### Start Services

#### Trigger Service (Event Publishers)
```bash
# Start trigger service to listen for triggers and publish events
./bin/operion-trigger run

# Start with custom trigger service ID
./bin/operion-trigger run --trigger-id my-trigger-service

# List all available triggers
./bin/operion-trigger list

# Validate trigger configurations
./bin/operion-trigger validate
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
1. **Trigger Service** listens for trigger conditions and publishes `WorkflowTriggered` events
2. **Worker Service** subscribes to events and executes the corresponding workflows
3. **Event Bus** decouples trigger detection from workflow execution (supports in-memory and Kafka)

### API Endpoints

```bash
# List all workflows
curl http://localhost:3000/workflows

# Health check
curl http://localhost:3000/
```

### Example Workflow

See `./data/workflows/bitcoin-price.json` for a complete workflow example that:
- Triggers every minute via cron schedule
- Fetches Bitcoin price data from CoinPaprika API
- Processes the data using JSONata transformation
- Saves the result to a file

## Current Implementation

### Available Triggers
- **Schedule**: Cron-based execution using robfig/cron (fully implemented and registered)
- **Kafka**: Message-based triggering from Kafka topics (fully implemented and registered)

### Available Actions  
- Actions are available but need to be migrated to the new registry system
- **HTTP Request**: Make HTTP calls to external APIs
- **Transform**: Process data using JSONata expressions  
- **File Write**: Save data to files
- **Log**: Output log messages with configurable levels

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

### Planned Interfaces
- **Web Dashboard**: Browser-based workflow editor and monitor
- **YAML Configuration**: Define workflows in YAML format
- **Extended CLI**: Additional workflow management commands
- 