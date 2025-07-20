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
- **API Server** (`cmd/api/`) - Fiber-based REST API with workflows and registry endpoints
  - `/workflows` - CRUD operations for workflows
  - `/registry/actions` - Sorted list of available actions with complete JSON schemas
  - `/registry/triggers` - Sorted list of available triggers with complete JSON schemas
- **CLI Worker** (`cmd/operion-worker/`) - Background workflow execution tool
- **CLI Dispatcher Service** (`cmd/operion-dispatcher/`) - Trigger listener and event publisher (replaces operion-trigger)
- **Visual Workflow Editor** (`ui/operion-editor/`) - React-based browser interface for workflow visualization
- **Domain Models** (`pkg/models/`) - Core workflow, action, and trigger models
- **Workflow Engine** (`pkg/workflow/`) - Workflow execution, management, and repository
- **Event System** (`pkg/event_bus/`, `pkg/events/`) - Event-driven communication
- **Plugin Registry** (`pkg/registry/`) - Plugin-based system for actions and triggers with .so file loading
- **File Persistence** (`pkg/persistence/file/`) - JSON file storage

### Available Triggers
- **Schedule Trigger** (`pkg/triggers/schedule/`) - Cron-based scheduling with robfig/cron with complete JSON schema
- **Webhook Trigger** (`pkg/triggers/webhook/`) - HTTP webhook endpoints with centralized server management and complete JSON schema
- **Queue Trigger** (`pkg/triggers/queue/`) - Message queue-based triggering with complete JSON schema

### Available Actions
- **HTTP Request** (`pkg/actions/http_request/`) - Make HTTP calls with retry logic and templating support
  - Schema includes: url (required), method, headers, body, retries (object with attempts/delay)
  - Templating examples: `{{steps.get_user_id.user_id}}`, `{{trigger.webhook.url}}/callback`
  - Retry config: `{"attempts": 3, "delay": 1000}` (attempts: 0-5, delay: 100-30000ms)
- **Transform** (`pkg/actions/transform/`) - Process data using JSONata expressions with input extraction
  - Schema includes: expression (required), input, id
  - JSONata examples: `$.name`, `{ "fullName": $.firstName & " " & $.lastName }`, `$count($.items)`
- **Log** (`pkg/actions/log/`) - Output log messages for debugging and monitoring
  - Schema includes: message (required), level
  - Templating examples: `Processing user: {{trigger.webhook.user_name}}`, `{{steps.api_call.status}}`

### Incomplete/Placeholder Components
- **Dashboard** (`cmd/dashboard/`) - Directory exists but not implemented (replaced by React-based UI)
- **Visual Editor Editing** - Current UI is read-only, workflow creation/editing features need implementation

## Development Memories

- **TODO Tracking**: Check the TODO.md file to see if the implementation was described

(rest of the file content remains the same)