# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Operion** is a workflow automation system built in Go that allows creating event-driven workflows with configurable triggers and actions. The system provides multiple interfaces: a REST API server, CLI tool, and planned web dashboard.

## Architecture

The project follows **Hexagonal Architecture** (Ports & Adapters) with clear separation of concerns:

- **Domain Layer** (`internal/domain/`) - Core business logic and interfaces
- **Application Layer** (`internal/application/`) - Use cases and orchestration  
- **Infrastructure Layer** (`internal/adapters/`) - External integrations
- **Interface Layer** (`cmd/`) - Entry points (API, CLI, Dashboard)

### Key Domain Models

- **Workflow** - Contains triggers, steps, variables, and metadata
- **WorkflowStep** - Individual workflow steps with actions and conditionals
- **Action Interface** - Contract for executable actions (pluggable architecture)
- **Trigger Interface** - Contract for workflow triggers (extensible system)
- **ExecutionContext** - Carries state between workflow steps

### Plugin Architecture

Uses registry pattern for extensibility:
- **TriggerRegistry** & **ActionRegistry** in `pkg/registry/`
- Factory pattern for creating triggers and actions
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
- **CLI Tool** (`cmd/operion/`) - Functional CLI with worker management
- **Worker Manager** (`internal/application/worker_manager.go`) - Background workflow execution
- **Workflow Executor** (`internal/application/workflow_executor.go`) - Executes workflow steps
- **Schedule Trigger** (`pkg/triggers/schedule/`) - Cron-based using robfig/cron
- **HTTP Request Action** (`pkg/actions/http_request/`) - External API calls
- **Transform Action** (`pkg/actions/transform/`) - Data processing using JSONata
- **File Write Action** (`pkg/actions/file_write/`) - Save data to files
- **Log Action** (`pkg/actions/log/`) - Logging with configurable levels
- **File Persistence** (`internal/adapters/persistence/file/`) - JSON file storage

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

To add new triggers: Implement `domain.Trigger` interface and register in `TriggerRegistry`
To add new actions: Implement `domain.Action` interface and register in `ActionRegistry`  
To add new persistence: Implement `domain.Persistence` interface

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
./bin/operion workers run

# Start workers with custom worker ID  
./bin/operion workers run --worker-id my-worker
```

The CLI tool provides:
- Background workflow execution
- Worker lifecycle management  
- Signal handling for graceful shutdown
- Structured logging with logrus
```

## Claude Guidance

- Never use emoji at documentation