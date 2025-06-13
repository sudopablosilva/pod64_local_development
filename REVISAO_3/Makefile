.PHONY: build up down test clean logs

# Build all services
build:
	@echo "Building all services..."
	@for service in control-m jmi jmw jmr scheduler-plugin spa spaq; do \
		echo "Building $$service..."; \
		cd $$service && go mod tidy && cd ..; \
	done
	@cd integration-tests && go mod tidy && cd ..

# Start all services
up:
	@echo "Starting all services..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 30

# Stop all services
down:
	@echo "Stopping all services..."
	docker-compose down -v

# Run integration tests
test:
	@echo "Running integration tests..."
	cd integration-tests && go test -v

# Run BDD tests with godog
bdd-test:
	@echo "Running BDD tests..."
	cd integration-tests && go run main_test.go features/

# Clean up everything
clean:
	@echo "Cleaning up..."
	docker-compose down -v --remove-orphans
	docker system prune -f

# Show logs
logs:
	docker-compose logs -f

# Show logs for specific service
logs-%:
	docker-compose logs -f $*

# Health check all services
health:
	@echo "Checking service health..."
	@for port in 8081 8082 8083 8084 8085 8086 8087; do \
		echo "Checking service on port $$port..."; \
		curl -s http://localhost:$$port/health || echo "Service on port $$port is not responding"; \
	done

# Submit a test job
test-job:
	@echo "Submitting test job..."
	curl -X POST http://localhost:8081/jobs \
		-H "Content-Type: application/json" \
		-d '{"job_name":"test-job","job_type":"shell","priority":1}'

# Full test cycle
full-test: clean build up test down

# Development cycle
dev: build up logs
