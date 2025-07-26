# Kafka Trigger

The Kafka trigger allows workflows to be triggered when messages are received on specified Kafka topics.

## Configuration

### Required Parameters

- `topic` - The Kafka topic name to subscribe to

### Optional Parameters

- `consumer_group` - Kafka consumer group ID (auto-generated if not provided)
- `brokers` - Comma-separated list of Kafka broker addresses (uses `KAFKA_BROKERS` env var if not provided)

### Environment Variables

- `KAFKA_BROKERS` - Comma-separated list of Kafka broker addresses (e.g., `kafka1:9092,kafka2:9092`)

## Example Configuration

### Basic Configuration

```json
{
  "id": "user-events-trigger",
  "type": "kafka",
  "configuration": {
    "topic": "user-events"
  }
}
```

### Advanced Configuration

```json
{
  "id": "order-processing-trigger",
  "type": "kafka", 
  "configuration": {
    "topic": "orders",
    "consumer_group": "operion-order-processor",
    "brokers": "kafka1.example.com:9092,kafka2.example.com:9092"
  }
}
```

## Trigger Data

When a Kafka message is received, the following data is passed to the workflow:

```json
{
  "topic": "user-events",
  "partition": 0,
  "offset": 12345,
  "timestamp": "2024-06-18T05:09:00Z",  
  "key": "user-123",
  "message": {
    "user_id": "123",
    "event_type": "login",
    "timestamp": "2024-06-18T05:09:00Z"
  },
  "headers": {
    "content-type": "application/json",
    "source": "auth-service"
  }
}
```

### Data Fields

- `topic` - Kafka topic name
- `partition` - Partition number the message came from
- `offset` - Message offset within the partition
- `timestamp` - When the trigger processed the message
- `key` - Kafka message key (if present)
- `message` - Parsed message payload (JSON if parseable, otherwise raw string)
- `headers` - Kafka message headers as key-value pairs

## Consumer Groups

### Auto-Generated Consumer Groups

If no `consumer_group` is specified, one will be automatically generated with the format:
```
operion-triggers-{trigger_id}
```

### Custom Consumer Groups

Use custom consumer groups for:
- **Load Balancing**: Multiple trigger services can share the same consumer group
- **Offset Management**: Resume from last processed message after restarts
- **Parallel Processing**: Different workflows can use different consumer groups for the same topic

## Environment Setup

### Using Environment Variables

```bash
export KAFKA_BROKERS="kafka1:9092,kafka2:9092,kafka3:9092"
./bin/operion-trigger run
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  trigger-service:
    image: operion:latest
    command: ["./bin/operion-trigger", "run"]
    environment:
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - kafka

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
```

## Message Processing

### JSON Messages

Messages that can be parsed as JSON will have their data available in `message`:

```json
{
  "message": {
    "user_id": "123",
    "action": "purchase",
    "amount": 99.99
  }
}
```

### Non-JSON Messages

Messages that cannot be parsed as JSON will be available as a raw string:

```json
{
  "message": {
    "raw_message": "Plain text message content"
  }
}
```

## Error Handling

### Connection Failures

- Automatic retry with exponential backoff
- Detailed logging of connection issues
- Graceful handling of broker unavailability

### Message Processing Errors

- Individual message processing errors are logged but don't stop the consumer
- Messages are marked as processed even if workflow execution fails
- Consider implementing dead letter queues for failed message handling

### Consumer Group Failures

- Automatic rebalancing when consumer group membership changes
- Session timeout and heartbeat configuration for reliability
- Offset management ensures no message loss during restarts

## Performance Considerations

### Consumer Configuration

The Kafka trigger uses optimized consumer settings:

- **Session Timeout**: 10 seconds
- **Heartbeat Interval**: 3 seconds  
- **Offset Strategy**: Start from newest messages
- **Error Handling**: Return errors for monitoring

### Scaling

- **Horizontal Scaling**: Run multiple trigger services with the same consumer group
- **Topic Partitions**: Ensure adequate partitions for parallel processing
- **Consumer Groups**: Use different consumer groups for different processing requirements

### Monitoring

Key metrics to monitor:

- Consumer lag (difference between latest offset and consumed offset)
- Message processing rate
- Error rates and types
- Consumer group rebalancing frequency

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   ERROR: failed to create Kafka consumer group: kafka: client has run out of available brokers
   ```
   - Check `KAFKA_BROKERS` environment variable
   - Verify Kafka broker addresses and ports
   - Ensure network connectivity

2. **Topic Does Not Exist**
   ```
   ERROR: Kafka consumer error: topic 'missing-topic' does not exist
   ```
   - Create the topic manually or enable auto-creation
   - Verify topic name spelling

3. **Consumer Group Issues**
   ```
   ERROR: Consumer group rebalance failed
   ```
   - Check for duplicate consumer group IDs
   - Verify consumer group configuration
   - Monitor for network issues

### Debugging

Enable detailed logging by setting log level to debug:

```bash
export LOG_LEVEL=debug
./bin/operion-trigger run
```

### Testing

Test Kafka trigger functionality:

```bash
# Validate Kafka trigger configuration
./bin/operion-trigger validate

# List all Kafka triggers
./bin/operion-trigger list | grep kafka

# Test with local Kafka
docker run -d --name kafka-test -p 9092:9092 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  confluentinc/cp-kafka:latest
```