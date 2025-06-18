# Operion API Documentation

## Overview

Operion is a workflow automation system that allows creating event-driven workflows with configurable triggers and actions. This API provides comprehensive endpoints for managing workflows, updating steps, and discovering available triggers and actions.

## Base URL

```
http://localhost:3000
```

## Authentication

Currently, the API does not require authentication. This may change in future versions.

## Content Type

All API endpoints accept and return JSON data. Include the following header in your requests:

```
Content-Type: application/json
```

## Error Handling

The API uses RFC 7807 Problem Details for HTTP APIs format for error responses:

```json
{
  "type": "validation_error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Workflow name is required",
  "instance": "/workflows"
}
```

Common error types:
- `validation_error` (400) - Invalid input data or unsupported action types
- `not_found` (404) - Resource not found
- `internal_error` (500) - Server error

### Validation Errors

#### Action Validation

When using invalid action types in workflows or steps, you'll receive detailed error messages:

```json
{
  "type": "validation_error",
  "title": "Bad Request", 
  "status": 400,
  "detail": "invalid action type 'invalid_action' in step 'my_step'. Available types: [http_request transform file_write log]",
  "instance": "/workflows/123e4567-e89b-12d3-a456-426614174000"
}
```

#### Step Name Validation

When using invalid step names (with spaces, uppercase, or special characters), you'll receive validation errors:

```json
{
  "type": "validation_error",
  "title": "Bad Request",
  "status": 400,
  "detail": "invalid step name 'My Step'. Step names must be lowercase alphanumeric with underscores only (e.g., 'fetch_data', 'log_result')",
  "instance": "/workflows/123e4567-e89b-12d3-a456-426614174000"
}
```

## API Endpoints

### Health Check

#### GET /

Check if the API is running.

**Response:**
```
Status: 200 OK
Content-Type: text/plain

Operion Workflow Automation API
```

---

## Workflows

### List All Workflows

#### GET /workflows

Retrieve all workflows.

**Response:**
```json
Status: 200 OK

[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "Bitcoin Price Monitor",
    "description": "Monitor Bitcoin price and log alerts",
    "triggers": [
      {
        "id": "trigger-1",
        "type": "schedule",
        "configuration": {
          "cron": "0 */5 * * * *"
        }
      }
    ],
    "steps": [
      {
        "id": "fetch_btc_price",
        "name": "fetch_btc_price",
        "action": {
          "id": "action-1",
          "type": "http_request",
          "name": "Get BTC Price",
          "description": "Fetch current Bitcoin price",
          "configuration": {
            "url": "https://api.coinbase.com/v2/exchange-rates?currency=BTC",
            "method": "GET"
          }
        },
        "conditional": {
          "language": "javascript",
          "expression": "true"
        },
        "enabled": true
      }
    ],
    "variables": {
      "threshold": 50000
    },
    "status": "active",
    "metadata": {
      "version": "1.0"
    },
    "owner": "system",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

### Get Single Workflow

#### GET /workflows/{id}

Retrieve a specific workflow by ID.

**Parameters:**
- `id` (path, required): Workflow UUID

**Response:**
```json
Status: 200 OK

{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Bitcoin Price Monitor",
  "description": "Monitor Bitcoin price and log alerts",
  "triggers": [...],
  "steps": [...],
  "variables": {...},
  "status": "active",
  "metadata": {...},
  "owner": "system",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Workflow not found

### Create Workflow

#### POST /workflows

Create a new workflow.

**Request Body:**
```json
{
  "name": "New Workflow",
  "description": "Description of the workflow",
  "triggers": [
    {
      "id": "trigger-1",
      "type": "schedule",
      "configuration": {
        "cron": "0 */10 * * * *"
      }
    }
  ],
  "steps": [
    {
      "name": "log_hello",
      "action": {
        "type": "log",
        "name": "Log Hello",
        "description": "Log a hello message",
        "configuration": {
          "level": "info",
          "message": "Hello from workflow!"
        }
      },
      "enabled": true
    }
  ],
  "variables": {},
  "status": "inactive",
  "owner": "user123"
}
```

**Required Fields:**
- `name` (string, min 3 chars): Workflow name
- `description` (string): Workflow description

**Optional Fields:**
- `triggers` (array): List of workflow triggers
- `steps` (array): List of workflow steps
- `variables` (object): Workflow variables
- `status` (string): Workflow status (defaults to "inactive")
- `metadata` (object): Additional metadata
- `owner` (string): Workflow owner

**Response:**
```json
Status: 201 Created

{
  "id": "456e7890-e89b-12d3-a456-426614174001",
  "name": "New Workflow",
  "description": "Description of the workflow",
  "triggers": [...],
  "steps": [...],
  "variables": {},
  "status": "inactive",
  "metadata": {},
  "owner": "user123",
  "created_at": "2024-01-15T11:00:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input data

### Update Workflow (Partial)

#### PATCH /workflows/{id}

Partially update an existing workflow using JSON Merge Patch (RFC 7396).

**Parameters:**
- `id` (path, required): Workflow UUID

**Request Body:**
JSON Merge Patch format - only include fields you want to update:
```json
{
  "name": "Updated Workflow Name",
  "status": "active"
}
```

Or update specific nested fields:
```json
{
  "variables": {
    "new_threshold": 100
  },
  "metadata": {
    "version": "2.0"
  }
}
```

**Action Validation:**
All workflow steps are validated to ensure action types are valid. Supported action types:
- `http_request` - Make HTTP requests
- `transform` - Transform data using JSONata
- `file_write` - Write data to files  
- `log` - Log messages

**Response:**
```json
Status: 200 OK

{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Updated Workflow Name",
  "description": "Original description",
  "status": "active",
  // ... other fields (unchanged fields remain as they were)
  "updated_at": "2024-01-15T12:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input data or invalid action types
- `404 Not Found` - Workflow not found

### Delete Workflow

#### DELETE /workflows/{id}

Delete a workflow.

**Parameters:**
- `id` (path, required): Workflow UUID

**Response:**
```
Status: 204 No Content
```

**Error Responses:**
- `404 Not Found` - Workflow not found

### Update Workflow Steps

#### PATCH /workflows/{id}/steps

Replace the entire steps array of a specific workflow.

**Parameters:**
- `id` (path, required): Workflow UUID

**Request Body:**
Array of workflow steps that will replace the existing steps. Step and action IDs will be auto-generated:
```json
[
  {
    "name": "log_message",
    "action": {
      "type": "log",
      "name": "Log Action",
      "description": "Log a message",
      "configuration": {
        "level": "info",
        "message": "Hello World"
      }
    },
    "enabled": true
  }
]
```

**Action Validation:**
All action types in steps are validated against available actions:
- `http_request` - Make HTTP requests to external APIs
- `transform` - Transform data using JSONata expressions  
- `file_write` - Write data to files
- `log` - Log messages with configurable levels

Invalid action types will return a 400 error with details about available types.

**Response:**
```json
Status: 200 OK

[
  {
    "id": "auto-generated-uuid",
    "name": "log_message",
    "action": {
      "id": "auto-generated-uuid",
      "type": "log",
      "name": "Log Action",
      "description": "Log a message",
      "configuration": {
        "level": "info",
        "message": "Hello World"
      }
    },
    "conditional": {
      "language": "",
      "expression": ""
    },
    "on_success": null,
    "on_failure": null,
    "enabled": true
  }
]
```

**Error Responses:**
- `400 Bad Request` - Invalid step data or unsupported action types
- `404 Not Found` - Workflow not found

---

## Registry

### Get Available Actions

#### GET /registry/actions

Get list of all available action types with their configuration schemas.

**Response:**
```json
Status: 200 OK

[
  {
    "type": "http_request",
    "name": "HTTP Request",
    "description": "Make HTTP requests to external APIs",
    "config_schema": {
      "url": {
        "type": "string",
        "required": true,
        "description": "The URL to make the request to"
      },
      "method": {
        "type": "string",
        "required": true,
        "enum": ["GET", "POST", "PUT", "DELETE", "PATCH"],
        "description": "HTTP method"
      },
      "headers": {
        "type": "object",
        "required": false,
        "description": "HTTP headers to include"
      },
      "body": {
        "type": "object",
        "required": false,
        "description": "Request body (for POST/PUT/PATCH)"
      }
    }
  },
  {
    "type": "transform",
    "name": "Transform Data",
    "description": "Transform data using JSONata expressions",
    "config_schema": {
      "expression": {
        "type": "string",
        "required": true,
        "description": "JSONata expression for data transformation"
      }
    }
  },
  {
    "type": "file_write",
    "name": "Write File",
    "description": "Write data to a file",
    "config_schema": {
      "path": {
        "type": "string",
        "required": true,
        "description": "File path to write to"
      },
      "content": {
        "type": "string",
        "required": true,
        "description": "Content to write to the file"
      }
    }
  },
  {
    "type": "log",
    "name": "Log Message",
    "description": "Log a message with configurable level",
    "config_schema": {
      "level": {
        "type": "string",
        "required": false,
        "enum": ["debug", "info", "warn", "error"],
        "default": "info",
        "description": "Log level"
      },
      "message": {
        "type": "string",
        "required": true,
        "description": "Message to log"
      }
    }
  }
]
```

### Get Available Triggers

#### GET /registry/triggers

Get list of all available trigger types with their configuration schemas.

**Response:**
```json
Status: 200 OK

[
  {
    "type": "schedule",
    "name": "Schedule (Cron)",
    "description": "Trigger workflow on a schedule using cron expressions",
    "config_schema": {
      "cron": {
        "type": "string",
        "required": true,
        "description": "Cron expression (e.g., '0 */5 * * * *' for every 5 minutes)"
      }
    }
  },
  {
    "type": "kafka",
    "name": "Kafka Message",
    "description": "Trigger workflow when Kafka message is received",
    "config_schema": {
      "topic": {
        "type": "string",
        "required": true,
        "description": "Kafka topic to subscribe to"
      },
      "brokers": {
        "type": "array",
        "required": true,
        "description": "List of Kafka broker addresses"
      },
      "group_id": {
        "type": "string",
        "required": false,
        "description": "Consumer group ID"
      }
    }
  }
]
```

---

## Data Models

### Workflow

```json
{
  "id": "string (UUID)",
  "name": "string (required, min 3 chars)",
  "description": "string (required)",
  "triggers": "TriggerItem[]",
  "steps": "WorkflowStep[]",
  "variables": "object",
  "status": "string (WorkflowStatus)",
  "metadata": "object",
  "owner": "string",
  "created_at": "string (ISO 8601 date)",
  "updated_at": "string (ISO 8601 date)",
  "deleted_at": "string (ISO 8601 date, optional)"
}
```

### WorkflowStep

```json
{
  "id": "string (auto-generated UUID when creating)",
  "name": "string (required, lowercase alphanumeric with underscores only)",
  "action": "ActionItem (required)",
  "conditional": "ConditionalExpression (optional)",
  "on_success": "string (optional, next step ID)",
  "on_failure": "string (optional, error step ID)",
  "enabled": "boolean (required)"
}
```

**Step Name Requirements:**
- Must be lowercase
- Only alphanumeric characters (a-z, 0-9) and underscores (_) allowed
- No spaces or special characters
- Examples: `fetch_data`, `log_result`, `transform_json`, `send_notification`

**ID Generation:**
- Step IDs are automatically generated as UUIDs when creating or updating workflows
- Action IDs are also auto-generated
- You should not provide IDs in your requests - they will be created automatically

### ActionItem

```json
{
  "id": "string (auto-generated UUID when creating)",
  "type": "string (action type, must be one of: http_request, transform, file_write, log)",
  "name": "string",
  "description": "string",
  "configuration": "object (action-specific config)"
}
```

### TriggerItem

```json
{
  "id": "string",
  "type": "string (trigger type)",
  "configuration": "object (trigger-specific config)"
}
```

### ConditionalExpression

```json
{
  "language": "string (required: 'javascript', 'cel', 'simple')",
  "expression": "string (required)"
}
```

---

## Workflow Status Values

The workflow status must be one of the following enum values:

- `active` - Workflow is running and triggers are enabled
- `inactive` - Workflow is stopped, triggers are disabled (default for new workflows)
- `paused` - Workflow is temporarily paused
- `error` - Workflow encountered an error

**Validation:**
Any other status value will result in a validation error. The status field is required.

---

## Usage Examples

### Creating a Simple Log Workflow

```bash
curl -X POST http://localhost:3000/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Hello World Workflow",
    "description": "A simple workflow that logs hello world every minute",
    "triggers": [
      {
        "id": "schedule-trigger",
        "type": "schedule",
        "configuration": {
          "cron": "0 * * * * *"
        }
      }
    ],
    "steps": [
      {
        "name": "log_hello",
        "action": {
          "type": "log",
          "name": "Hello Log",
          "description": "Log hello world message",
          "configuration": {
            "level": "info",
            "message": "Hello, World!"
          }
        },
        "enabled": true
      }
    ],
    "owner": "demo-user"
  }'
```

### Creating a Bitcoin Price Monitor

```bash
curl -X POST http://localhost:3000/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bitcoin Price Monitor",
    "description": "Monitor Bitcoin price and log when it changes significantly",
    "triggers": [
      {
        "id": "price-check-trigger",
        "type": "schedule",
        "configuration": {
          "cron": "0 */5 * * * *"
        }
      }
    ],
    "steps": [
      {
        "name": "fetch_btc_price",
        "action": {
          "type": "http_request",
          "name": "Get Bitcoin Price",
          "description": "Fetch current Bitcoin price from Coinbase API",
          "configuration": {
            "url": "https://api.coinbase.com/v2/exchange-rates?currency=BTC",
            "method": "GET",
            "headers": {
              "Accept": "application/json"
            }
          }
        },
        "enabled": true
      },
      {
        "name": "extract_usd_price",
        "action": {
          "type": "transform",
          "name": "Extract Price",
          "description": "Extract USD price from API response",
          "configuration": {
            "expression": "data.rates.USD"
          }
        },
        "enabled": true
      },
      {
        "name": "log_price",
        "action": {
          "type": "log",
          "name": "Log BTC Price",
          "description": "Log the current Bitcoin price",
          "configuration": {
            "level": "info",
            "message": "Current Bitcoin price: ${{data}}"
          }
        },
        "enabled": true
      }
    ],
    "variables": {
      "alert_threshold": 50000,
      "last_price": 0
    },
    "owner": "crypto-trader"
  }'
```

### Updating Workflow (Partial Updates)

```bash
# Update just the workflow name and status
curl -X PATCH http://localhost:3000/workflows/123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Bitcoin Monitor",
    "status": "active"
  }'

# Update workflow variables
curl -X PATCH http://localhost:3000/workflows/123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "alert_threshold": 60000,
      "check_interval": 300
    }
  }'
```

### Updating Workflow Steps

```bash
# Replace entire steps array
curl -X PATCH http://localhost:3000/workflows/123e4567-e89b-12d3-a456-426614174000/steps \
  -H "Content-Type: application/json" \
  -d '[
    {
      "name": "log_message",
      "action": {
        "type": "log",
        "name": "Updated Log Action",
        "description": "Updated logging action",
        "configuration": {
          "level": "warn",
          "message": "This is an updated message"
        }
      },
      "enabled": true
    }
  ]'

# Update with multiple steps
curl -X PATCH http://localhost:3000/workflows/123e4567-e89b-12d3-a456-426614174000/steps \
  -H "Content-Type: application/json" \
  -d '[
    {
      "name": "fetch_data",
      "action": {
        "type": "http_request",
        "name": "API Call",
        "description": "Make API request",
        "configuration": {
          "url": "https://api.example.com/data",
          "method": "GET"
        }
      },
      "enabled": true
    },
    {
      "name": "log_response",
      "action": {
        "type": "log",
        "name": "Log Result",
        "description": "Log API response",
        "configuration": {
          "level": "info",
          "message": "API Response: {{data}}"
        }
      },
      "enabled": true
    }
  ]'
```

---

## Frontend Integration Guide

This API is designed to support a visual workflow builder frontend. Here are key considerations:

### Building a Workflow Designer

1. **Get Available Components**: Use `/registry/actions` and `/registry/triggers` to populate component palettes
2. **Schema Validation**: Use the `config_schema` from registry endpoints to build dynamic forms
3. **Visual Flow**: The `on_success` and `on_failure` fields in steps support building visual flow diagrams
4. **Real-time Updates**: Use the step update endpoint for real-time workflow editing

### Recommended Frontend Flow

1. Load available actions and triggers from registry endpoints
2. Create workflow with basic info (name, description)
3. Add/modify steps using the steps update endpoint
4. Use conditional expressions for complex logic
5. Test workflows before activating them

### Configuration Schema Format

The `config_schema` follows this pattern:
```json
{
  "field_name": {
    "type": "string|number|boolean|object|array",
    "required": true|false,
    "description": "Human readable description",
    "enum": ["option1", "option2"], // for select fields
    "default": "default_value" // optional default
  }
}
```

This schema can be used to dynamically generate forms in your frontend application.

---

## Development and Testing

### Running the API

```bash
# Development with live reload
air

# Or build and run
make build
./bin/api
```

The API will be available at `http://localhost:3000` by default. Set the `PORT` environment variable to use a different port.

### Data Storage

Workflows are stored as JSON files in the `./data/workflows/` directory. Each workflow is saved as `{workflow-id}.json`.

### Testing

The API includes comprehensive integration tests to ensure reliability:

```bash
# Run all tests
make test-all

# Run integration tests only
make test-integration

# Run with coverage
make test-coverage
```

#### Integration Test Coverage

The test suite covers:
- **CRUD Operations**: Create, read, update, delete workflows
- **PATCH Endpoints**: JSON merge patch validation and functionality
- **Action Validation**: Ensures only valid action types are accepted
- **Registry Endpoints**: Available actions and triggers discovery
- **Error Handling**: Proper HTTP status codes and error messages
- **Edge Cases**: Missing resources, validation failures, malformed data

Tests use isolated temporary storage and clean up automatically, making them safe to run in any environment.