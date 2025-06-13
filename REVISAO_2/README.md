# POC BDD - Microservices Job Processing Pipeline

This project implements a microservices architecture for job processing with BDD (Behavior-Driven Development) integration tests.

## Architecture

The system consists of the following services:

1. **Control-M** (Port 8081) - Job submission and management
2. **JMI** (Job Manager Integrator, Port 8082) - Integrates and validates jobs
3. **JMW** (Job Manager Worker, Port 8083) - Processes jobs
4. **JMR** (Job Manager Runner, Port 8084) - Executes jobs
5. **Scheduler Plugin** (Port 8085) - Creates schedules for jobs
6. **SPA** (Scheduler Plugin Adapter, Port 8086) - Configures adapters
7. **SPAQ** (Scheduler Plugin Adapter Queue, Port 8087) - Manages queue messages

## Data Flow

```
Control-M → JMI → JMW → JMR → Scheduler Plugin → SPA → SPAQ
```

Each service communicates through SQS queues and persists data in DynamoDB tables.

## Prerequisites

- Docker and Docker Compose
- Go 1.22+
- Make (optional, for using Makefile commands)

## Quick Start

1. **Build all services:**
   ```bash
   make build
   ```

2. **Start the system:**
   ```bash
   make up
   ```

3. **Check service health:**
   ```bash
   make health
   ```

4. **Run integration tests:**
   ```bash
   make test
   ```

5. **Submit a test job:**
   ```bash
   make test-job
   ```

6. **View logs:**
   ```bash
   make logs
   ```

7. **Stop the system:**
   ```bash
   make down
   ```

## Manual Commands

### Build and Start Services

```bash
# Build all services
for service in control-m jmi jmw jmr scheduler-plugin spa spaq; do
    cd $service && go mod tidy && cd ..
done

# Start services
docker-compose up -d

# Wait for services to be ready
sleep 30
```

### Run Tests

```bash
# Run BDD integration tests
cd integration-tests
go test -v

# Or run with godog directly
go run main_test.go features/
```

### Submit Jobs

```bash
# Submit a simple job
curl -X POST http://localhost:8081/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "job_name": "example-job",
    "job_type": "shell",
    "priority": 1,
    "parameters": {
      "command": "echo Hello World"
    }
  }'
```

### Check Service Status

```bash
# Check all service health endpoints
for port in 8081 8082 8083 8084 8085 8086 8087; do
    echo "Service on port $port:"
    curl -s http://localhost:$port/health | jq .
done
```

### View Data

```bash
# View jobs in JMI
curl -s http://localhost:8082/jobs | jq .

# View schedules
curl -s http://localhost:8085/schedules | jq .

# View adapters
curl -s http://localhost:8086/adapters | jq .

# View queue messages
curl -s http://localhost:8087/messages | jq .

# View queue stats
curl -s http://localhost:8087/stats | jq .
```

## BDD Tests

The integration tests are written using Gherkin syntax and implemented with Godog. The tests cover:

### Job Processing Pipeline
- Job submission through Control-M
- Processing through all services
- Data persistence verification
- End-to-end workflow validation

### Service Communication
- SQS message flow between services
- Service health endpoints
- Error handling and recovery

### Test Features

1. **features/job_processing.feature** - Tests the complete job processing pipeline
2. **features/service_communication.feature** - Tests inter-service communication

### Running Specific Tests

```bash
cd integration-tests

# Run all tests
go test -v

# Run specific feature
go run main_test.go features/job_processing.feature

# Run with pretty output
go run main_test.go -format=pretty features/
```

## Troubleshooting

### Services Not Starting
- Check if ports 8081-8087 and 4566 are available
- Ensure Docker has enough resources allocated
- Check logs: `make logs`

### Tests Failing
- Ensure all services are healthy: `make health`
- Check LocalStack is running: `curl http://localhost:4566/health`
- Verify SQS queues exist: `aws --endpoint-url=http://localhost:4566 sqs list-queues`

### LocalStack Issues
- Restart LocalStack: `docker-compose restart localstack`
- Check initialization: `docker-compose logs localstack`

## Development

### Adding New Services
1. Create service directory with Go application
2. Add Dockerfile
3. Update docker-compose.yml
4. Add service to Makefile
5. Create BDD tests for new functionality

### Modifying Tests
1. Update feature files in `integration-tests/features/`
2. Implement step definitions in `integration-tests/steps/`
3. Run tests to verify changes

## Architecture Decisions

- **Microservices**: Each component is a separate service for scalability
- **Event-Driven**: Services communicate through SQS queues
- **Data Persistence**: Each service manages its own data in DynamoDB
- **Health Checks**: All services expose health endpoints
- **BDD Testing**: Behavior-driven tests ensure system works as expected

## Monitoring

Each service exposes the following endpoints:

- `GET /health` - Health check
- `GET /stats` - Service-specific statistics (where applicable)
- Service-specific endpoints for data retrieval

## Scaling

The architecture supports horizontal scaling:

- Multiple instances of JMW and JMR can run simultaneously
- SQS provides natural load balancing
- DynamoDB handles concurrent access
- Each service is stateless (except for data persistence)
