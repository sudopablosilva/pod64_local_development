#!/bin/bash
set -e

echo "Waiting for services to be ready..."
sleep 15

echo "Checking LocalStack health..."
curl -s http://localstack:4566/_localstack/health | jq .

echo "Running integration tests..."
cd integration-tests
go test -v

echo "Tests completed!"