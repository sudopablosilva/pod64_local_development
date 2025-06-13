#!/bin/bash

echo "Initializing LocalStack resources..."

# Create DynamoDB tables
awslocal dynamodb create-table \
    --table-name jobs \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal dynamodb create-table \
    --table-name executions \
    --attribute-definitions \
        AttributeName=executionName,AttributeType=S \
    --key-schema \
        AttributeName=executionName,KeyType=HASH \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal dynamodb create-table \
    --table-name schedules \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal dynamodb create-table \
    --table-name adapters \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal dynamodb create-table \
    --table-name queue_messages \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5

# Create SQS queues
awslocal sqs create-queue --queue-name job-requests
awslocal sqs create-queue --queue-name jmw-queue
awslocal sqs create-queue --queue-name jmr-queue
awslocal sqs create-queue --queue-name sp-queue
awslocal sqs create-queue --queue-name spa-queue
awslocal sqs create-queue --queue-name spaq-queue

echo "LocalStack initialization completed!"
