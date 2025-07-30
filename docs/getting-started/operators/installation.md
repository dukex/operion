# Installation Guide

This guide walks you through installing Operion from source and getting your first workflow automation platform running.

## Prerequisites

- **Go 1.19+** - Required for building the Go backend services
- **Node.js 16+** & **npm** - Required for the React-based visual editor
- **Git** - For cloning the repository
- **Make** - For using the provided build commands
- **Docker** - For running Kafka (recommended) or local Kafka installation

## Quick Start Installation

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/operion.git
cd operion
```

### 2. Install Dependencies

```bash
# Download Go dependencies
go mod download
```

### 3. Build the Services

```bash
# Build all services
make build
```

This creates the following binaries in the `bin/` directory:
- `operion-api` - Main REST API server
- `operion-worker` - Background workflow execution service
- `operion-dispatcher` - Trigger listener and event publisher

### 4. Basic Configuration

The project includes example data and workflows in the `examples/` directory which will be used as the working directory. No additional setup is required as the example data structure is already provided.

### 5. Set Up Kafka (Required for Event Bus)

Since the services use Kafka as the event bus, you can start Kafka using the provided Docker Compose configuration:

**Using Docker Compose (Recommended):**
```bash
# Start Kafka and create required topics automatically
docker-compose up -d

# Check that topics were created successfully
docker-compose logs kafka-setup
```

**Alternative - Using Docker directly:**
```bash
# Start Kafka in KRaft mode (no Zookeeper required)
docker run -d --name kafka -p 9092:9092 \
  -e KAFKA_NODE_ID=1 \
  -e KAFKA_PROCESS_ROLES=broker,controller \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
  -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
  -e KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0 \
  -e KAFKA_NUM_PARTITIONS=3 \
  -e CLUSTER_ID=operion-dev-cluster \
  confluentinc/cp-kafka:latest

# Wait for Kafka to start, then create the required topics
sleep 15
docker exec kafka kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic operion.events \
  --partitions 3 \
  --replication-factor 1

docker exec kafka kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic operion.trigger \
  --partitions 3 \
  --replication-factor 1
```

### 6. Start the Operion Services

Start the services in the following order:

**Terminal 1 - API Server:**
```bash
export KAFKA_BROKERS=localhost:9092
./bin/operion-api run --port 8099 --database-url file://./examples/data --event-bus kafka
```
The API server will start on port 8099 as specified.

**Terminal 2 - Dispatcher Service:**
```bash
export KAFKA_BROKERS=localhost:9092
./bin/operion-dispatcher run --database-url file://./examples/data --event-bus kafka --receiver-config ./configs/receivers.yaml
```
This service runs receivers that listen for external triggers and publishes events to the `operion.trigger` and `operion.events` Kafka topics.

**Terminal 3 - Worker Service:**
```bash
export KAFKA_BROKERS=localhost:9092
./bin/operion-worker run --database-url file://./examples/data --event-bus kafka
```
This service executes workflows when events are received from the `operion.events` Kafka topic.

## Verification

### Check API Health

```bash
curl http://localhost:8099/health
```

Expected response:
```json
{
  "checkers": {
    "registry": "Plugins loaded successfully",
    "repository": "Persistence layer is healthy"
  },
  "message": "Operion API is healthy",
  "status": "healthy",
  "timestamp": "2025-07-26T23:00:21.313196Z"
}
```

### List Available Actions

```bash
curl http://localhost:8099/registry/actions
```

You should see built-in actions like `http_request`, `transform`, and `log`.

## Directory Structure

After installation, your directory should look like:

```
operion/
├── bin/                    # Built binaries
│   ├── operion-api
│   ├── operion-worker
│   └── operion-dispatcher
├── configs/                # Configuration files
│   └── receivers.yaml     # Receiver configuration
├── data/
│   └── workflows/          # Workflow storage
├── plugins/                # Plugin .so files
├── ui/operion-editor/      # React frontend
├── examples/               # Example workflows
└── docker-compose.yml     # Kafka development setup
```

## Next Steps

- [Docker Deployment](docker-deployment.md) - Deploy using Docker containers
- [Kubernetes Deployment](kubernetes-deployment.md) - Deploy to Kubernetes
- [Production Setup](production-setup.md) - Configure for production use
- [First Workflow Tutorial](../developers/first-workflow.md) - Create your first workflow

## Troubleshooting

### Build Issues

**Error: `go: module not found`**
```bash
go mod tidy
go mod download
```

### Service Issues

**Error: `bind: address already in use`**
Check if another service is using port 8099:
```bash
lsof -i :8099
```

**Kafka connection issues:**
Verify Kafka is running and accessible:
```bash
# Check if Kafka is running
docker ps | grep kafka
# or for local installation
ps aux | grep kafka

# Test Kafka connectivity
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092

# If using Docker Compose
docker-compose ps
docker-compose logs kafka
```

**Error: `no such file or directory`**
Ensure you've built the binaries:
```bash
make build
ls -la bin/
```

For more troubleshooting help, see [Troubleshooting Guide](../../how-to/troubleshooting/common-issues.md).