#!/bin/bash

# Operion Services Launcher
# Based on docs/getting-started/operators/installation.md

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
KAFKA_BROKERS=${KAFKA_BROKERS:-localhost:9092}
DATABASE_URL=${DATABASE_URL:-file://./examples/data}
EVENT_BUS=${EVENT_BUS:-kafka}
API_PORT=${API_PORT:-8099}
WEBHOOK_PORT=${WEBHOOK_PORT:-8085}
RECEIVER_CONFIG=${RECEIVER_CONFIG:-./configs/receivers.yaml}

# Service selection flags
RUN_API=false
RUN_DISPATCHER=false
RUN_WORKER=false

# Function to display usage
usage() {
    echo -e "${BLUE}Operion Services Launcher${NC}"
    echo ""
    echo "Usage: $0 [OPTIONS] [SERVICES]"
    echo ""
    echo "Services:"
    echo "  api         Run API server"
    echo "  dispatcher  Run dispatcher service"
    echo "  worker      Run worker service"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -k, --kafka-brokers     Kafka brokers (default: localhost:9092)"
    echo "  -d, --database-url      Database URL (default: file://./examples/data)"
    echo "  -e, --event-bus         Event bus type (default: kafka)"
    echo "  -p, --api-port          API server port (default: 8099)"
    echo "  -w, --webhook-port      Webhook server port (default: 8085)"
    echo "  -r, --receiver-config   Receiver config file (default: ./configs/receivers.yaml)"
    echo ""
    echo "Examples:"
    echo "  $0                      # Run all services"
    echo "  $0 api                  # Run only API server"
    echo "  $0 api worker           # Run API and worker"
    echo "  $0 -p 8080 api          # Run API on port 8080"
    echo ""
    echo "Environment Variables:"
    echo "  KAFKA_BROKERS          Override Kafka brokers"
    echo "  DATABASE_URL           Override database URL"
    echo "  EVENT_BUS              Override event bus type"
    echo "  API_PORT               Override API port"
    echo "  WEBHOOK_PORT           Override webhook port"
    echo "  RECEIVER_CONFIG        Override receiver config file"
}

# Function to check if binary exists
check_binary() {
    local binary=$1
    if [[ ! -f "./bin/$binary" ]]; then
        echo -e "${RED}Error: Binary ./bin/$binary not found${NC}"
        echo -e "${YELLOW}Please run 'make build' or 'go build -o bin/$binary ./cmd/$binary/' first${NC}"
        exit 1
    fi
}

# Function to check if Kafka is running
check_kafka() {
    echo -e "${BLUE}Checking Kafka connectivity...${NC}"
    
    # Try Docker Compose first
    if docker-compose ps kafka 2>/dev/null | grep -q "Up"; then
        echo -e "${GREEN}✓ Kafka is running via docker-compose${NC}"
        return 0
    fi
    
    # Try standalone Docker
    if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "kafka.*Up"; then
        echo -e "${GREEN}✓ Kafka is running via docker${NC}"
        return 0
    fi
    
    # Test direct connection
    if timeout 5 bash -c "</dev/tcp/localhost/9092" 2>/dev/null; then
        echo -e "${GREEN}✓ Kafka is accessible on localhost:9092${NC}"
        return 0
    fi
    
    echo -e "${RED}✗ Kafka is not accessible${NC}"
    echo -e "${YELLOW}Please start Kafka using:${NC}"
    echo -e "${YELLOW}  docker-compose up -d${NC}"
    echo -e "${YELLOW}or follow the installation guide${NC}"
    exit 1
}

# Function to run API server
run_api() {
    echo -e "${GREEN}Starting API Server on port $API_PORT...${NC}"
    check_binary "operion-api"
    
    export KAFKA_BROKERS
    exec ./bin/operion-api run \
        --port "$API_PORT" \
        --database-url "$DATABASE_URL" \
        --event-bus "$EVENT_BUS"
}

# Function to run dispatcher service
run_dispatcher() {
    echo -e "${GREEN}Starting Dispatcher Service...${NC}"
    check_binary "operion-dispatcher"
    
    if [[ ! -f "$RECEIVER_CONFIG" ]]; then
        echo -e "${RED}Error: Receiver config file $RECEIVER_CONFIG not found${NC}"
        exit 1
    fi
    
    export KAFKA_BROKERS
    exec ./bin/operion-dispatcher run \
        --database-url "$DATABASE_URL" \
        --event-bus "$EVENT_BUS" \
        --webhook-port "$WEBHOOK_PORT" \
        --receiver-config "$RECEIVER_CONFIG"
}

# Function to run worker service
run_worker() {
    echo -e "${GREEN}Starting Worker Service...${NC}"
    check_binary "operion-worker"
    
    export KAFKA_BROKERS
    exec ./bin/operion-worker run \
        --database-url "$DATABASE_URL" \
        --event-bus "$EVENT_BUS"
}

# Function to run services in background with process management
run_services() {
    local pids=()
    local services=()
    
    # Trap to kill all background processes on exit
    trap 'echo -e "\n${YELLOW}Shutting down services...${NC}"; kill ${pids[@]} 2>/dev/null; wait; exit' INT TERM
    
    if [[ "$RUN_API" == true ]]; then
        echo -e "${GREEN}Starting API Server on port $API_PORT...${NC}"
        check_binary "operion-api"
        export KAFKA_BROKERS
        ./bin/operion-api run \
            --port "$API_PORT" \
            --database-url "$DATABASE_URL" \
            --event-bus "$EVENT_BUS" &
        pids+=($!)
        services+=("API Server (PID: $!)")
    fi
    
    if [[ "$RUN_DISPATCHER" == true ]]; then
        echo -e "${GREEN}Starting Dispatcher Service...${NC}"
        check_binary "operion-dispatcher"
        if [[ ! -f "$RECEIVER_CONFIG" ]]; then
            echo -e "${RED}Error: Receiver config file $RECEIVER_CONFIG not found${NC}"
            kill ${pids[@]} 2>/dev/null
            exit 1
        fi
        export KAFKA_BROKERS
        ./bin/operion-dispatcher run \
            --database-url "$DATABASE_URL" \
            --event-bus "$EVENT_BUS" \
            --webhook-port "$WEBHOOK_PORT" \
            --receiver-config "$RECEIVER_CONFIG" &
        pids+=($!)
        services+=("Dispatcher Service (PID: $!)")
    fi
    
    if [[ "$RUN_WORKER" == true ]]; then
        echo -e "${GREEN}Starting Worker Service...${NC}"
        check_binary "operion-worker"
        export KAFKA_BROKERS
        ./bin/operion-worker run \
            --database-url "$DATABASE_URL" \
            --event-bus "$EVENT_BUS" &
        pids+=($!)
        services+=("Worker Service (PID: $!)")
    fi
    
    if [[ ${#pids[@]} -eq 0 ]]; then
        echo -e "${RED}No services selected to run${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}Services started:${NC}"
    for service in "${services[@]}"; do
        echo -e "  ${GREEN}✓${NC} $service"
    done
    
    echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
    
    # Wait for all background processes
    wait
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -k|--kafka-brokers)
            KAFKA_BROKERS="$2"
            shift 2
            ;;
        -d|--database-url)
            DATABASE_URL="$2"
            shift 2
            ;;
        -e|--event-bus)
            EVENT_BUS="$2"
            shift 2
            ;;
        -p|--api-port)
            API_PORT="$2"
            shift 2
            ;;
        -w|--webhook-port)
            WEBHOOK_PORT="$2"
            shift 2
            ;;
        -r|--receiver-config)
            RECEIVER_CONFIG="$2"
            shift 2
            ;;
        api)
            RUN_API=true
            shift
            ;;
        dispatcher)
            RUN_DISPATCHER=true
            shift
            ;;
        worker)
            RUN_WORKER=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

# If no services specified, run all
if [[ "$RUN_API" == false && "$RUN_DISPATCHER" == false && "$RUN_WORKER" == false ]]; then
    RUN_API=true
    RUN_DISPATCHER=true
    RUN_WORKER=true
fi

# Check prerequisites
check_kafka

# If only one service is requested, run it in foreground
if [[ "$RUN_API" == true && "$RUN_DISPATCHER" == false && "$RUN_WORKER" == false ]]; then
    run_api
elif [[ "$RUN_DISPATCHER" == true && "$RUN_API" == false && "$RUN_WORKER" == false ]]; then
    run_dispatcher
elif [[ "$RUN_WORKER" == true && "$RUN_API" == false && "$RUN_DISPATCHER" == false ]]; then
    run_worker
else
    # Run multiple services in background
    run_services
fi