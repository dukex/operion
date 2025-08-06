# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Operion** is a cloud-native workflow automation platform built in Go that enables event-driven workflows with configurable triggers and actions. Designed following cloud-native principles, Operion is stateless, container-first, and optimized for Kubernetes deployments. The system provides multiple interfaces: a REST API server, CLI tools, and a React-based visual editor.

## Architecture

The project follows a clean, layered architecture with clear separation of concerns:

- **Models Layer** (`pkg/models/`) - Core domain models and interfaces
- **Business Logic** (`pkg/workflow/`) - Workflow execution and management
- **Infrastructure Layer** (`pkg/persistence/`, `pkg/event_bus/`) - External integrations and data access
- **Extensions** (`pkg/registry/`) - Plugin system for actions and triggers
- **Interface Layer** (`cmd/`) - Entry points (API server, CLI tool)

### Key Domain Models

- **Workflow** - Contains triggers, steps, variables, and metadata
- **WorkflowStep** - Individual workflow steps with actions
- **Action Interface** - Contract for executable actions (pluggable architecture)
- **Trigger Interface** - Contract for workflow triggers (extensible system)
- **ExecutionContext** - Carries state between workflow steps

### Plugin Architecture

Uses plugin-based system for extensibility:
- **Registry** - Plugin registry for both actions and triggers in `pkg/registry/`
- Dynamic loading of `.so` plugin files from filesystem
- Factory pattern with `ActionFactory` and `TriggerFactory` interfaces
- Protocol-based interfaces in `pkg/protocol/` for actions and triggers
- Runtime configuration from `map[string]any`
- **Schema Support** - All ActionFactory and TriggerFactory implementations include Schema() method returning JSON Schema for configuration validation
- **Templating Examples** - All action schemas include comprehensive examples showing how to use templating with step results, trigger data, and built-in functions
- **Trigger Factory Interface** - All TriggerFactory implementations include ID(), Name(), Description(), and Schema() methods for consistent trigger registration and documentation

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

### Environment Variables

#### API Server (operion-api)
```bash
PORT=9091              # API server port (default: 9091)
DATABASE_URL           # Database connection URL (required)
KAFKA_BROKERS          # Kafka broker addresses (required)
PLUGINS_PATH=./plugins # Path to action plugins directory (default: ./plugins)
LOG_LEVEL=info         # Log level: debug, info, warn, error (default: info)
```

#### Worker Service (operion-worker)
```bash
WORKER_ID              # Custom worker ID (auto-generated if not provided)
DATABASE_URL           # Database connection URL (required)
KAFKA_BROKERS          # Kafka broker addresses (required)
PLUGINS_PATH=./plugins # Path to action plugins directory (default: ./plugins)
LOG_LEVEL=info         # Log level: debug, info, warn, error (default: info)
```

#### Dispatcher Service (operion-dispatcher)
```bash
DISPATCHER_ID          # Custom dispatcher ID (auto-generated if not provided)
DATABASE_URL           # Database connection URL (required)
KAFKA_BROKERS          # Kafka broker addresses (required)
PLUGINS_PATH=./plugins # Path to action plugins directory (default: ./plugins)
WEBHOOK_PORT=8085      # Port for webhook HTTP server (default: 8085)
LOG_LEVEL=info         # Log level: debug, info, warn, error (default: info)
```

#### Source Manager Service (operion-source-manager)
```bash
SOURCE_MANAGER_ID      # Custom source manager ID (auto-generated if not provided)
DATABASE_URL           # Database connection URL (required)
KAFKA_BROKERS          # Kafka broker addresses (required)
PLUGINS_PATH=./plugins # Path to source provider plugins directory (default: ./plugins)
SOURCE_PROVIDERS       # Comma-separated list of providers to run (e.g., 'scheduler,webhook')
LOG_LEVEL=info         # Log level: debug, info, warn, error (default: info)
```

#### Activator Service (operion-activator)
```bash
ACTIVATOR_ID           # Custom activator ID (auto-generated if not provided)
DATABASE_URL           # Database connection URL (required)
KAFKA_BROKERS          # Kafka broker addresses (required)
LOG_LEVEL=info         # Log level: debug, info, warn, error (default: info)
```

### Visual Editor Development
```bash
cd ui/operion-editor    # Navigate to UI directory
npm install             # Install dependencies (first time)
npm run dev             # Start development server on port 5173
npm run build           # Build for production
npm run lint            # Run ESLint
```

### Testing and Quality
```bash
make test           # Run all tests
make test-coverage  # Generate coverage report (coverage.out and coverage.html)
make fmt            # Format Go code
make lint           # Run golangci-lint
```

### Dependencies
```bash
go mod download     # Download dependencies
go mod tidy         # Clean up dependencies
```

### Coding Standards

#### Struct Tag Alignment
All struct tags within a struct must be vertically aligned for better readability:

```go
// ✅ Correct - tags are aligned
type Example struct {
    ID          string `json:"id"          validate:"required"`
    Name        string `json:"name"        validate:"required,min=3"`
    Description string `json:"description"`
}

// ❌ Incorrect - tags are not aligned  
type Example struct {
    ID string `json:"id" validate:"required"`
    Name string `json:"name" validate:"required,min=3"`
    Description string `json:"description"`
}
```

The `tagalign` linter is configured to enforce this rule automatically. Run `make fmt` after making struct changes to ensure proper formatting.

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and quality assurance:

#### Test and Coverage Workflow (`.github/workflows/test-and-coverage.yml`)
- **Triggers**: Every pull request and push to main branch
- **Go Version**: 1.24
- **Quality Checks**:
  - Format verification with `gofmt`
  - Vet analysis with `go vet`
  - Static analysis with `staticcheck`
  - Linting with `golangci-lint`
- **Testing**:
  - Race condition detection with `-race` flag
  - Coverage generation with `-coverprofile=coverage.out`
  - Atomic coverage mode for accuracy
- **Coverage Reporting**:
  - Uploads to Codecov with project token
  - Uploads to Coveralls with GitHub token
  - HTML reports generated for artifacts
- **Build Verification**:
  - Builds all three binaries (operion-api, operion-worker, operion-dispatcher)
  - Uploads build artifacts for download
- **Caching**: Go modules cached for faster builds

## Current Implementation Status

### Available Components
- **API Server** (`cmd/api/`) - Fiber-based REST API with workflows and registry endpoints
  - `/workflows` - CRUD operations for workflows
  - `/registry/actions` - Sorted list of available actions with complete JSON schemas
  - `/registry/triggers` - Sorted list of available triggers with complete JSON schemas
- **CLI Worker** (`cmd/operion-worker/`) - Background workflow execution tool
- **CLI Dispatcher Service** (`cmd/operion-dispatcher/`) - Trigger listener and event publisher (replaces operion-trigger)
- **CLI Source Manager** (`cmd/operion-source-manager/`) - Centralized scheduler orchestrator for managing source providers
- **CLI Activator** (`cmd/operion-activator/`) - Bridge between source events and workflow events
- **Visual Workflow Editor** (`ui/operion-editor/`) - React-based browser interface for workflow visualization
- **Domain Models** (`pkg/models/`) - Core workflow, action, and trigger models
- **Workflow Engine** (`pkg/workflow/`) - Workflow execution, management, and repository
- **Event System** (`pkg/event_bus/`, `pkg/events/`) - Kafka-based event-driven communication with dual topics
- **Plugin Registry** (`pkg/registry/`) - Plugin-based system for actions and triggers with .so file loading
- **File Persistence** (`pkg/persistence/file/`) - JSON file storage

### Available Triggers
- **Schedule Trigger** (`pkg/triggers/schedule/`) - Cron-based scheduling with robfig/cron with complete JSON schema
- **Webhook Trigger** (`pkg/triggers/webhook/`) - HTTP webhook endpoints with centralized server management and complete JSON schema
- **Queue Trigger** (`pkg/triggers/queue/`) - Message queue-based triggering with complete JSON schema
- **Kafka Trigger** (`pkg/triggers/kafka/`) - Kafka topic message consumption with consumer group support and complete JSON schema
  - Trigger data includes: topic, partition, offset, timestamp, key, message, headers

### Available Actions
- **HTTP Request** (`pkg/actions/http_request/`) - Make HTTP calls with retry logic and templating support
  - Schema includes: url (required), method, headers, body, retries (object with attempts/delay)
  - Templating examples: `{{.step_results.get_user_id.user_id}}`, `{{.trigger_data.webhook.url}}/callback`
  - Retry config: `{"attempts": 3, "delay": 1000}` (attempts: 0-5, delay: 100-30000ms)
- **Transform** (`pkg/actions/transform/`) - Process data using Go templates
  - Schema includes: expression (required), id
  - Go template examples: `{{.name}}`, `{ "fullName": "{{.firstName}} {{.lastName}}" }`, `{{len .items}}`
- **Log** (`pkg/actions/log/`) - Output log messages for debugging and monitoring
  - Schema includes: message (required), level
  - Templating examples: `Processing user: {{.trigger_data.webhook.user_name}}`, `{{step_results.api_call.status}}`

## Development Memories

- **TODO Tracking**: Check the TODO.md file to see if the implementation was described
- Always run `golangci-lint run --fix <folder or file>`, passing the folder or files that have been edited.
- Always run `go fmt ./...` after finishing editing files and before building