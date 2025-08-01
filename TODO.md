# Operion Development TODO

This file tracks planned features and improvements for Operion, a cloud-native workflow automation platform.

## High Priority TODOs (Essential Features)

### Cloud-Native Triggers

#### TODO-T001: RabbitMQ Trigger Implementation
**Priority:** High  
**Description:** Implement RabbitMQ-specific trigger for AMQP message consumption  
**Location:** `pkg/triggers/rabbitmq/`

```go
// pkg/triggers/rabbitmq/
type RabbitMQTrigger struct {
    ID           string            `json:"id"`
    ConnectionURL string           `json:"connection_url"`    // "amqp://user:pass@host:port/"
    Exchange     string            `json:"exchange"`
    RoutingKey   string            `json:"routing_key"`
    Queue        string            `json:"queue"`
    QueueDurable bool              `json:"queue_durable"`
    AutoAck      bool              `json:"auto_ack"`
    PrefetchCount int              `json:"prefetch_count"`
    TLS          *TLSConfig        `json:"tls,omitempty"`
}
```

**Use Cases:**
- Enterprise messaging patterns
- Microservice communication via AMQP
- Message routing with exchanges and queues
- Dead letter queue handling

**Dependencies:** `github.com/rabbitmq/amqp091-go`

---

#### TODO-T002: AWS SQS Trigger Implementation  
**Priority:** High  
**Description:** Implement AWS SQS trigger for cloud-native queue consumption  
**Location:** `pkg/triggers/sqs/`

```go
// pkg/triggers/sqs/
type SQSTrigger struct {
    ID                  string `json:"id"`
    QueueURL           string `json:"queue_url"`
    Region             string `json:"region"`
    MaxMessages        int32  `json:"max_messages"`        // 1-10
    WaitTimeSeconds    int32  `json:"wait_time_seconds"`   // Long polling
    VisibilityTimeout  int32  `json:"visibility_timeout"`
    MessageAttributeNames []string `json:"message_attribute_names"`
    AWSConfig          *AWSConfig `json:"aws_config,omitempty"`
}

type AWSConfig struct {
    AccessKeyID     string `json:"access_key_id,omitempty"`
    SecretAccessKey string `json:"secret_access_key,omitempty"`
    SessionToken    string `json:"session_token,omitempty"`
    Profile         string `json:"profile,omitempty"`
    RoleARN         string `json:"role_arn,omitempty"`
}
```

**Use Cases:**
- AWS-native serverless workflows
- Cross-service communication in AWS
- FIFO queue support
- Dead letter queue integration

**Dependencies:** `github.com/aws/aws-sdk-go-v2`

---

#### TODO-T003: Enhanced Kafka Trigger
**Priority:** Medium  
**Description:** Enhance existing Kafka trigger with cloud-native features  
**Location:** `pkg/triggers/kafka/` (enhance existing)

**Enhancements:**
- SASL authentication (SCRAM, OAuth)
- Schema Registry integration
- Avro/Protobuf message support
- Kafka Connect integration
- Multi-partition consumption strategies

---

#### TODO-T004: Google Pub/Sub Trigger
**Priority:** Medium  
**Description:** Implement Google Cloud Pub/Sub trigger  
**Location:** `pkg/triggers/pubsub/`

```go
// pkg/triggers/pubsub/
type PubSubTrigger struct {
    ID             string            `json:"id"`
    ProjectID      string            `json:"project_id"`
    Subscription   string            `json:"subscription"`
    MaxMessages    int32             `json:"max_messages"`
    Credentials    *GCPCredentials   `json:"credentials,omitempty"`
}
```

**Dependencies:** `cloud.google.com/go/pubsub`

---

### Cloud-Native Actions

#### TODO-A001: Email Action Implementation
**Priority:** High  
**Description:** SMTP-based email notifications for cloud environments  
**Location:** `pkg/actions/email/`

```go
// pkg/actions/email/
type EmailAction struct {
    SMTPConfig EmailSMTPConfig      `json:"smtp"`
    To         []string            `json:"to"`
    CC         []string            `json:"cc,omitempty"`
    BCC        []string            `json:"bcc,omitempty"`
    Subject    string              `json:"subject"`
    Body       string              `json:"body"`
    BodyType   string              `json:"body_type"`    // "text", "html"
    Attachments []EmailAttachment  `json:"attachments,omitempty"`
}

type EmailSMTPConfig struct {
    Host       string `json:"host"`
    Port       int    `json:"port"`
    Username   string `json:"username"`
    Password   string `json:"password"`
    TLS        bool   `json:"tls"`
    StartTLS   bool   `json:"start_tls"`
}
```

**Use Cases:**
- Alert notifications
- Report delivery
- Workflow completion notifications
- Error notifications

---

#### TODO-A002: Slack/Discord Webhook Action
**Priority:** High  
**Description:** Team communication via webhooks  
**Location:** `pkg/actions/slack/`

```go
// pkg/actions/slack/
type SlackAction struct {
    WebhookURL string                `json:"webhook_url"`
    Channel    string                `json:"channel,omitempty"`
    Username   string                `json:"username,omitempty"`
    IconEmoji  string                `json:"icon_emoji,omitempty"`
    Message    string                `json:"message"`
    Blocks     []SlackBlock          `json:"blocks,omitempty"`
    Metadata   map[string]string     `json:"metadata,omitempty"`
}
```

**Use Cases:**
- Incident alerting
- Deployment notifications
- Team status updates
- Workflow monitoring

---

#### TODO-A003: Database Action Implementation
**Priority:** High  
**Description:** Cloud database operations (PostgreSQL, MySQL, MongoDB)  
**Location:** `pkg/actions/database/`

```go
// pkg/actions/database/
type DatabaseAction struct {
    ConnectionString string                 `json:"connection_string"`
    Driver          string                 `json:"driver"`        // "postgres", "mysql", "mongodb"
    Operation       string                 `json:"operation"`     // "insert", "update", "delete", "select"
    Table           string                 `json:"table"`
    Data            map[string]any `json:"data,omitempty"`
    Conditions      map[string]any `json:"conditions,omitempty"`
    Query           string                 `json:"query,omitempty"`     // Raw SQL/Query
    Timeout         string                 `json:"timeout"`
    TLS             *TLSConfig            `json:"tls,omitempty"`
}
```

**Use Cases:**
- Data persistence in workflows
- State tracking
- Analytics data collection
- Configuration management

---

#### TODO-A004: Conditional/Branch Action
**Priority:** High  
**Description:** Workflow branching based on conditions  
**Location:** `pkg/actions/conditional/`

```go
// pkg/actions/conditional/
type ConditionalAction struct {
    Condition    string                 `json:"condition"`      // JSONata expression
    TrueAction   *WorkflowStep         `json:"true_action,omitempty"`
    FalseAction  *WorkflowStep         `json:"false_action,omitempty"`
    Cases        []ConditionalCase     `json:"cases,omitempty"`    // Switch-like behavior
}

type ConditionalCase struct {
    Condition string        `json:"condition"`
    Action    *WorkflowStep `json:"action"`
}
```

**Use Cases:**
- Dynamic workflow routing
- Business rule implementation
- Error handling flows
- Multi-path workflows

---

#### TODO-A005: Template/Report Generation Action
**Priority:** Medium  
**Description:** Generate documents from templates  
**Location:** `pkg/actions/template/`

```go
// pkg/actions/template/
type TemplateAction struct {
    TemplateType   string                 `json:"template_type"`   // "html", "json", "yaml", "text"
    Template       string                 `json:"template"`        // Template content or URL
    Data           map[string]any `json:"data"`
    OutputFormat   string                 `json:"output_format"`   // "json", "html", "pdf"
    OutputTarget   OutputTarget           `json:"output_target"`
}

type OutputTarget struct {
    Type   string `json:"type"`     // "http", "s3", "gcs", "response"
    Config map[string]any `json:"config"`
}
```

**Use Cases:**
- Report generation
- Configuration file creation
- API response formatting
- Document transformation

---

## Medium Priority TODOs (Enhanced Features)

#### TODO-A006: Delay/Wait Action
**Priority:** Medium  
**Description:** Time-based workflow control  
**Location:** `pkg/actions/delay/`

```go
// pkg/actions/delay/
type DelayAction struct {
    Duration     string `json:"duration"`      // "5s", "2m", "1h"
    UntilTime    string `json:"until_time"`    // "2024-12-01T10:00:00Z"
    Condition    string `json:"condition"`     // Wait until condition is true
    MaxWait      string `json:"max_wait"`      // Maximum wait time
    PollInterval string `json:"poll_interval"` // For condition-based waits
}
```

**Use Cases:**
- Rate limiting
- Scheduled delays
- External condition waiting
- Batch processing intervals

---

#### TODO-A007: Aggregation/Analytics Action
**Priority:** Medium  
**Description:** Data aggregation and analysis  
**Location:** `pkg/actions/aggregate/`

```go
// pkg/actions/aggregate/
type AggregateAction struct {
    DataSource   string                 `json:"data_source"`   // Step result reference
    Operations   []AggregateOperation   `json:"operations"`
    GroupBy      []string               `json:"group_by"`
    Filters      []string               `json:"filters"`       // JSONata expressions
    OutputFormat string                 `json:"output_format"` // "json", "csv"
}

type AggregateOperation struct {
    Type   string `json:"type"`   // "sum", "avg", "count", "max", "min"
    Field  string `json:"field"`
    Alias  string `json:"alias"`
}
```

**Use Cases:**
- Data summarization
- Metrics calculation
- Report aggregation
- Analytics pipelines

---

## Cloud-Native Infrastructure TODOs

#### TODO-I001: Kubernetes Integration
**Priority:** High  
**Description:** Native Kubernetes deployment and scaling

**Features:**
- Helm charts for deployment
- Horizontal Pod Autoscaler (HPA) support
- Service mesh integration (Istio)
- ConfigMap/Secret integration
- Health checks and readiness probes

---

#### TODO-I002: Observability Enhancement
**Priority:** High  
**Description:** Cloud-native monitoring and tracing

**Features:**
- Prometheus metrics export
- Jaeger/Zipkin distributed tracing
- Structured logging with correlation IDs
- Health check endpoints
- Performance metrics dashboard

---

#### TODO-I003: Security Enhancements
**Priority:** High  
**Description:** Cloud-native security features

**Features:**
- OAuth2/OIDC authentication
- RBAC for workflow management
- Secret management integration (Vault, AWS Secrets Manager)
- mTLS support
- Audit logging

---

#### TODO-I004: Multi-tenancy Support
**Priority:** Medium  
**Description:** Support for multiple tenants/organizations

**Features:**
- Tenant isolation
- Resource quotas per tenant
- Tenant-specific configurations
- Billing and usage tracking

---

## Lower Priority TODOs (Specialized Features)

#### TODO-T005: Azure Service Bus Trigger
**Priority:** Low  
**Description:** Microsoft Azure Service Bus integration  
**Location:** `pkg/triggers/servicebus/`

#### TODO-T006: Redis Streams Trigger  
**Priority:** Low  
**Description:** Enhanced Redis trigger using Streams API  
**Location:** `pkg/triggers/redis_streams/`

#### TODO-A008: Object Storage Action
**Priority:** Low  
**Description:** S3/GCS/Azure Blob operations  
**Location:** `pkg/actions/object_storage/`

#### TODO-A009: Webhook Action
**Priority:** Low  
**Description:** Make HTTP calls to external webhooks  
**Location:** `pkg/actions/webhook/`

## Implementation Notes

### Cloud-Native Principles
- **12-Factor App Compliance**: Configuration via environment variables
- **Stateless Design**: No local file dependencies for core operations
- **Container-First**: Docker/OCI container optimized
- **Observability**: Metrics, logs, and traces built-in
- **Scalability**: Horizontal scaling support
- **Security**: Zero-trust security model

### Development Guidelines
- All new features must include comprehensive tests
- Cloud provider integrations should support multiple authentication methods
- Configuration should be externalized (env vars, config maps)
- Error handling must be robust with proper retry logic
- All actions should support template/JSONata expressions
- Implement graceful shutdown for all components

### Removed Non-Cloud-Native Items
- ❌ **File Watcher Trigger**: Not suitable for containerized environments
- ❌ **File Operations Action**: Local filesystem operations conflict with cloud-native principles
- ❌ **Local File Storage**: Replaced with object storage solutions

These items were removed as they don't align with cloud-native, stateless application design principles.