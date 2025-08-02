# Environment Variables TODO

This file lists potentially useful environment variables that are not currently implemented but could be valuable additions to Operion in future versions.

## General Configuration

### TLS/SSL Configuration
- `OPERION_TLS_ENABLED` - Enable TLS/SSL encryption
- `OPERION_TLS_CERT_FILE` - Path to TLS certificate file
- `OPERION_TLS_KEY_FILE` - Path to TLS private key file
- `OPERION_TLS_CA_FILE` - Path to CA certificate file
- `OPERION_TLS_VERIFY_CLIENT` - Require client certificate verification

### CORS Configuration
- `OPERION_CORS_ENABLED` - Enable Cross-Origin Resource Sharing
- `OPERION_CORS_ORIGINS` - Allowed origins (comma-separated)
- `OPERION_CORS_METHODS` - Allowed HTTP methods
- `OPERION_CORS_HEADERS` - Allowed headers
- `OPERION_CORS_CREDENTIALS` - Allow credentials in CORS requests

### Authentication Configuration
- `OPERION_AUTH_ENABLED` - Enable authentication
- `OPERION_AUTH_TYPE` - Authentication type (jwt, basic, oauth)
- `OPERION_JWT_SECRET` - JWT signing secret
- `OPERION_JWT_EXPIRY` - JWT token expiration time
- `OPERION_AUTH_HEADER` - Header name for authentication token

## Database Configuration Enhancements

### Connection Pool Settings
- `DATABASE_MAX_CONNECTIONS` - Maximum database connections
- `DATABASE_CONNECTION_TIMEOUT` - Connection timeout
- `DATABASE_POOL_SIZE` - Connection pool size
- `DATABASE_SSL_MODE` - SSL mode (disable, require, prefer)

## Event Bus Configuration Enhancements

### Kafka Configuration
- `KAFKA_BROKERS` - Kafka broker addresses (comma-separated)
- `KAFKA_TOPIC` - Kafka topic for workflow events
- `KAFKA_CONSUMER_GROUP` - Kafka consumer group ID
- `KAFKA_CLIENT_ID` - Kafka client identifier
- `KAFKA_VERSION` - Kafka protocol version
- `KAFKA_COMPRESSION` - Message compression (none, gzip, snappy, lz4)
- `KAFKA_BATCH_SIZE` - Producer batch size
- `KAFKA_TIMEOUT` - Request timeout
- `KAFKA_RETRY_MAX` - Maximum retry attempts

#### Kafka Security
- `KAFKA_SECURITY_PROTOCOL` - Security protocol (PLAINTEXT, SSL, SASL_PLAINTEXT, SASL_SSL)
- `KAFKA_SSL_CA_FILE` - Path to CA certificate file
- `KAFKA_SSL_CERT_FILE` - Path to client certificate file
- `KAFKA_SSL_KEY_FILE` - Path to client private key file
- `KAFKA_SASL_MECHANISM` - SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
- `KAFKA_SASL_USERNAME` - SASL username
- `KAFKA_SASL_PASSWORD` - SASL password

### RabbitMQ Configuration
- `RABBITMQ_URL` - RabbitMQ connection URL
- `RABBITMQ_EXCHANGE` - Exchange name for workflow events
- `RABBITMQ_QUEUE` - Queue name for workflow events
- `RABBITMQ_ROUTING_KEY` - Routing key for messages
- `RABBITMQ_DURABLE` - Make queues and exchanges durable
- `RABBITMQ_AUTO_DELETE` - Auto-delete queues when unused

## Workflow Execution Configuration

### Performance Settings
- `OPERION_MAX_CONCURRENT_WORKFLOWS` - Maximum concurrent workflow executions
- `OPERION_WORKFLOW_TIMEOUT` - Default workflow execution timeout
- `OPERION_STEP_TIMEOUT` - Default step execution timeout
- `OPERION_WORKER_POOL_SIZE` - Number of worker goroutines
- `OPERION_QUEUE_SIZE` - Internal queue size for workflow events

### Retry Configuration
- `OPERION_RETRY_ENABLED` - Enable automatic retries for failed workflows
- `OPERION_MAX_RETRIES` - Maximum retry attempts
- `OPERION_RETRY_DELAY` - Initial delay between retries
- `OPERION_RETRY_BACKOFF` - Backoff strategy (fixed, exponential)
- `OPERION_RETRY_MAX_DELAY` - Maximum delay between retries

### Circuit Breaker Configuration
- `OPERION_CIRCUIT_BREAKER_ENABLED` - Enable circuit breaker for external calls
- `OPERION_CIRCUIT_BREAKER_THRESHOLD` - Failure threshold to open circuit
- `OPERION_CIRCUIT_BREAKER_TIMEOUT` - Timeout before attempting to close circuit
- `OPERION_CIRCUIT_BREAKER_RESET_TIMEOUT` - Reset timeout for successful calls

## Caching Configuration

### Cache Settings
- `OPERION_CACHE_ENABLED` - Enable caching
- `OPERION_CACHE_TYPE` - Cache backend (memory, redis)
- `OPERION_CACHE_DEFAULT_TTL` - Default cache TTL
- `OPERION_CACHE_MAX_SIZE` - Maximum cache entries (memory cache)

### Redis Cache Configuration
- `REDIS_URL` - Redis connection URL
- `REDIS_POOL_SIZE` - Redis connection pool size
- `REDIS_TIMEOUT` - Redis operation timeout
- `REDIS_DB` - Redis database number
- `REDIS_PREFIX` - Key prefix for Redis entries

## Monitoring and Observability

### Metrics Configuration
- `OPERION_METRICS_ENABLED` - Enable Prometheus metrics
- `OPERION_METRICS_PORT` - Port for metrics endpoint
- `OPERION_METRICS_PATH` - Path for metrics endpoint
- `OPERION_METRICS_NAMESPACE` - Metrics namespace

### Health Check Configuration
- `OPERION_HEALTH_CHECK_ENABLED` - Enable health check endpoint
- `OPERION_HEALTH_CHECK_PATH` - Health check endpoint path
- `OPERION_HEALTH_CHECK_INTERVAL` - Interval between health checks
- `OPERION_HEALTH_CHECK_TIMEOUT` - Health check timeout

### Tracing Configuration
- `OPERION_TRACING_ENABLED` - Enable distributed tracing
- `OPERION_TRACING_TYPE` - Tracing backend (jaeger, zipkin)
- `JAEGER_ENDPOINT` - Jaeger collector endpoint
- `JAEGER_SAMPLE_RATE` - Tracing sample rate (0.0-1.0)

## Development and Testing

### Development Settings
- `OPERION_DEV_MODE` - Enable development mode features
- `OPERION_HOT_RELOAD` - Enable hot reload (dev mode only)
- `OPERION_DEBUG_ENABLED` - Enable debug logging and features
- `OPERION_PROFILER_ENABLED` - Enable Go profiler endpoints

### Testing Configuration
- `OPERION_TEST_MODE` - Enable test mode
- `OPERION_TEST_DATA_PATH` - Path to test data files
- `OPERION_MOCK_EXTERNAL_CALLS` - Mock external HTTP calls

## Plugin Configuration Enhancements

### Plugin System
- `OPERION_PLUGINS_ENABLED` - Enable plugin system
- `OPERION_PLUGINS_AUTO_LOAD` - Automatically load plugins on startup
- `OPERION_PLUGINS_WATCH` - Watch plugin directory for changes
- `OPERION_PLUGIN_TIMEOUT` - Plugin execution timeout

## Priority Recommendations

### High Priority
1. **Metrics and Monitoring** - Essential for production deployments
2. **Health Checks** - Required for container orchestration
3. **TLS/SSL Configuration** - Security requirement for production
4. **Database Connection Pooling** - Performance optimization

### Medium Priority
1. **Authentication/Authorization** - Security enhancement
2. **Caching** - Performance optimization
3. **Circuit Breaker** - Reliability improvement
4. **Retry Configuration** - Error handling enhancement

### Low Priority
1. **Development Mode Features** - Development convenience
2. **Advanced Plugin Configuration** - Extended functionality
3. **Distributed Tracing** - Advanced observability

## Implementation Notes

- Most configuration should support both environment variables and command-line flags
- Consider using a configuration library like Viper for more sophisticated config management
- Environment variables should follow consistent naming conventions (OPERION_ prefix)
- Required vs optional variables should be clearly documented
- Default values should be production-ready where possible