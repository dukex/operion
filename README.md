# OperiON

A workflow automation system built in Go, allowing you to create event-driven workflows with configurable triggers and actions.

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Available Triggers](#available-triggers)
- [Available Actions](#available-actions)
- [Development](#development)
- [API Reference](#api-reference)
- [Contributing](#contributing)

## 🎯 Overview

The OpeniON is a system that allows creating automated workflows through:

- **Triggers**: Events that initiate a workflow (Schedule, Kafka, Webhook, etc.)
- **Actions**: Operations executed in the workflow (HTTP Request, Slack, Email, etc.)
- **Conditionals**: Conditional logic for flow control
- **Context**: Data sharing between workflow steps

## ✨ Features

- 🔧 **Extensible** - Easy addition of new triggers and actions
- ⚙️ **Configurable** - Configuration via database and YAML
- 🔄 **Reusable** - Composed actions for reusability
- 🛡️ **Resilient** - Built-in retry mechanisms and error handling
- 🚀 **High Performance** - Concurrent execution and efficient resource usage

## 🏗️ Architecture

This project follows **Hexagonal Architecture** (Ports and Adapters) principles:

### Layers

- **Domain Layer** (`internal/domain/`) - Core business logic and interfaces
- **Application Layer** (`internal/application/`) - Use cases and orchestration
- **Infrastructure Layer** (`internal/adapters/`) - External integrations
- **Interface Layer** (`cmd/`) - Entry points (API, CLI, Dashboard)

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    Triggers     │────│    Workflow     │────│     Actions     │
│                 │    │                 │    │                 │
│ • Schedule      │    │ • Conditionals  │    │ • HTTP Request  │
│ • Kafka         │    │ • Variables     │    │ • Slack         │
│ • Webhook       │    │ • Context       │    │ • Email         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 📁 Project Structure

```
.
├── cmd/                       # Application entry points
│   ├── api/                   # REST API server
│   │   └── dto/               # Data Transfer Objects
│   ├── dashboard/             # Web dashboard
│   └── operion/               # CLI tool
└── internal/                  # Internal application code
    ├── actions/               # Action implementations
    │   └── http_request/      # HTTP request action
    ├── adapters/              # Infrastructure adapters
    │   └── persistence/       # Persistence layer
    │       └── file/          # File-based persistence
    ├── application/           # Application services
    ├── domain/                # Domain models and interfaces
    └── triggers/              # Trigger implementations
        ├── kafka/             # Kafka event trigger
        └── schedule/          # Cron schedule trigger
```

## 🚀 Installation

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (optional)
- Kafka (for event triggers)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/dukex/operion.git
cd operion

# Download dependencies
go mod download

# Build all components
make build

# Or build specific components
go build -o bin/api ./cmd/api
go build -o bin/dashboard ./cmd/dashboard
go build -o bin/operion ./cmd/operion
```

## ⚙️ Configuration

### Environment Variables

```bash
PORT=8080
```

## 🎮 Usage

### Starting the Services

```bash
# Start API server
./bin/api

# Start Dashboard (in another terminal)
./bin/dashboard

# Use CLI tool
./bin/operion --help
```

### Creating a Workflow

#### Via API

```bash
curl -X POST http://localhost:8080/workflows \
  -H "Content-Type: application/json" \
  -d '[
  {
    "id": "sample-1",
    "name": "Sample Workflow 1",
    "description": "This is a sample workflow to demonstrate the structure and content of a YAML file for workflows.",
    "version": "1.0",
    "triggers": [
      {
        "id": "cron_trigger",
        "type": "schedule",
        "configuration": {
          "cron": "0 0 * * *",
          "timezone": "UTC"
        }
      }
    ],
    "variables": {},
    "metadata": {},
    "created_at": "2023-10-01T00:00:00Z",
    "updated_at": "2023-10-01T00:00:00Z",
    "status": "active",
    "steps": [
      {
        "id": "bitcoin_data",
        "conditional": {},
        "enabled": true,
        "on_success": "get_price",
        "action": {
          "uid": "bitcoin_data",
          "type": "http_request",
          "name": "Fetch Bitcoin Data",
          "description": "Fetch data from the CoinPaprika API for Bitcoin",
          "configuration": {
            "method": "GET",
            "url": "https://api.coinpaprika.com/v1/coins/btc-bitcoin/ohlcv/today",
            "headers": {
              "Content": "application/json"
            },
            "retry": {
              "attempts": 3,
              "delay": 5
            }
          }
        }
      },
      {
        "id": "get_price",
        "conditional": {},
        "enabled": true,
        "on_success": "save_price",
        "action": {
          "id": "get_price",
          "type": "transform",
          "name": "Process Data using JSONata",
          "description": "Process the data fetched from the API",
          "configuration": {
            "input": "$.bitcoin_data",
            "exp": "{\n  \"price\": $.close ? $.close : $.open\n}"
          }
        }
      },
      {
        "id": "save_price",
        "conditional": {},
        "enabled": true,
        "action": {
          "id": "save_price",
          "type": "file_write",
          "name": "Save Price to File",
          "description": "Save the processed price data to a file",
          "configuration": {
            "file_name": "bitcoin_price.json",
            "directory": "/tmp",
            "overwrite": true
          }
        }
      }
    ]
  }
]
'
```

#### Via CLI

```bash
# Create workflow from file
./bin/operion create workflow -f data/workflows/daily-report.yaml

# List workflows
./bin/operion list workflows

# Execute workflow manually
./bin/operion execute workflow daily-report

# Check workflow status
./bin/operion status workflow daily-report
```

#### Via YAML File

TODO: soon

## 🔧 Available Triggers

### Schedule Trigger

Executes workflows based on cron expressions.

```yaml
trigger:
  type: "schedule"
  config:
    cron: "0 */6 * * *"        # Every 6 hours
    timezone: "America/New_York"
    enabled: true
```

### Kafka Trigger

Listens to Kafka topics for events.

```yaml
trigger:
  type: "kafka"
  config:
    topic: "user.events"
    consumer_group: "workflow-engine"
    filters:
      event_type: "user.signup"
      source: "web"
```

### Webhook Trigger

Receives HTTP requests to trigger workflows.

```yaml
trigger:
  type: "webhook"
  config:
    path: "/webhooks/github"
    method: "POST"
    headers:
      X-GitHub-Event: "push"
```

## ⚡ Available Actions

### HTTP Request Action

Makes HTTP requests to external APIs.

```yaml
action:
  type: "http_request"
  config:
    method: "POST"
    url: "https://api.service.com/endpoint"
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer ${API_TOKEN}"
    body: |
      {
        "data": "{{trigger.payload.data}}",
        "timestamp": "{{now | date "RFC3339"}}"
      }
    timeout: "30s"
```

### Slack Action

Sends messages to Slack channels.

```yaml
action:
  type: "slack"
  config:
    channel: "#alerts"
    username: "Workflow Bot"
    icon_emoji: ":robot_face:"
    message: |
      🚨 Alert: {{trigger.event_type}}
      
      Details: {{previous_step.response.message}}
      Time: {{now | date "15:04:05"}}
```

### Email Action

Sends email notifications.

```yaml
action:
  type: "email"
  config:
    to: ["admin@company.com", "{{trigger.user.email}}"]
    cc: ["team@company.com"]
    subject: "Workflow Alert: {{workflow.name}}"
    body: |
      Hi {{trigger.user.name}},
      
      Your workflow "{{workflow.name}}" has completed.
      
      Status: {{workflow.status}}
      Duration: {{workflow.duration}}
```

## 🛠️ Development

### Adding New Triggers

1. **Create trigger implementation:**

```go
// internal/triggers/webhook/webhook.go
package webhook

import (
    "context"
    "github.com/your-org/workflow-engine/internal/domain"
)

type WebhookTrigger struct {
    id     string
    config WebhookConfig
    server *http.Server
}

func (t *WebhookTrigger) Start(ctx context.Context, callback domain.TriggerCallback) error {
    // Implementation
}

func (t *WebhookTrigger) Stop(ctx context.Context) error {
    // Implementation
}

func (t *WebhookTrigger) GetType() string {
    return "webhook"
}
```

2. **Register the trigger:**

```go
// internal/application/registry.go
func (r *TriggerRegistry) RegisterDefaults() {
    r.Register("schedule", schedule.NewFactory())
    r.Register("kafka", kafka.NewFactory())
    r.Register("webhook", webhook.NewFactory()) // Add new trigger
}
```

3. **Add configuration schema:**

```go
// internal/triggers/webhook/config.go
type WebhookConfig struct {
    Path    string            `json:"path" yaml:"path"`
    Method  string            `json:"method" yaml:"method"`
    Headers map[string]string `json:"headers" yaml:"headers"`
    Port    int               `json:"port" yaml:"port"`
}
```

### Adding New Actions

1. **Create action implementation:**

```go
// internal/actions/database/database.go
package database

type DatabaseAction struct {
    id     string
    config DatabaseConfig
    client *sql.DB
}

func (a *DatabaseAction) Execute(ctx context.Context, input domain.ExecutionContext) (domain.ExecutionContext, error) {
    // Implementation
}
```

2. **Register the action:**

```go
// internal/application/registry.go
func (r *ActionRegistry) RegisterDefaults() {
    r.Register("http_request", http_request.NewFactory())
    r.Register("slack", slack.NewFactory())
    r.Register("database", database.NewFactory()) // Add new action
}
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific test
go test ./internal/triggers/schedule/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Generate documentation
make docs

# Check dependencies
make mod-check
```

## 📚 API Reference

### Workflows API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | `/workflows` | List workflows |
| POST   | `/workflows` | Create workflow |
| GET    | `/workflows/{id}` | Get workflow |
| PUT    | `/workflows/{id}` | Update workflow |
| DELETE | `/workflows/{id}` | Delete workflow |
| POST   | `/workflows/{id}/execute` | Execute workflow |

### Executions API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | `/executions` | List executions |
| GET    | `/executions/{id}` | Get execution |
| POST   | `/executions/{id}/cancel` | Cancel execution |

### Health API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | `/health` | Health check |
| GET    | `/metrics` | Prometheus metrics |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Write comprehensive tests for new features
- Update documentation for any changes
- Use conventional commit messages
- Ensure all CI checks pass

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 [Documentation](https://docs.workflow-engine.com)
- 🐛 [Issue Tracker](https://github.com/your-org/workflow-engine/issues)
- 💬 [Discussions](https://github.com/your-org/workflow-engine/discussions)
- 📧 [Email Support](mailto:support@workflow-engine.com)
- 