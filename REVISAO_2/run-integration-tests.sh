#!/bin/bash
set -e

echo "Starting services with Docker Compose..."
docker-compose down -v
docker-compose up -d

echo "Waiting for services to be ready..."
sleep 15

echo "Running integration tests..."
cd integration-tests
go test -v

echo "Tests completed, stopping services..."
cd ..
docker-compose down

echo "Done!"