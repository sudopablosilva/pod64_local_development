#!/bin/bash
set -e

echo "Starting services with Docker Compose..."
finch compose -f docker-compose-test.yml down -v
finch compose -f docker-compose-test.yml build
finch compose -f docker-compose-test.yml up --abort-on-container-exit

echo "Tests completed, cleaning up..."
finch compose -f docker-compose-test.yml down

echo "Done!"