package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func TestLocalStackConnection(t *testing.T) {
	// Test LocalStack health
	resp, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		t.Fatalf("Failed to connect to LocalStack: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("LocalStack is not healthy: status %d", resp.StatusCode)
	}

	fmt.Println("✓ LocalStack is running and healthy")
}

func TestAWSServices(t *testing.T) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:4566"}, nil
			})),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Test DynamoDB
	dynamoClient := dynamodb.NewFromConfig(cfg)
	tables, err := dynamoClient.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		t.Fatalf("Failed to list DynamoDB tables: %v", err)
	}

	expectedTables := []string{"jobs", "schedules", "adapters", "queue_messages"}
	for _, expectedTable := range expectedTables {
		found := false
		for _, table := range tables.TableNames {
			if table == expectedTable {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected table %s not found", expectedTable)
		}
	}
	fmt.Println("✓ All DynamoDB tables are available")

	// Test SQS
	sqsClient := sqs.NewFromConfig(cfg)
	queues, err := sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		t.Fatalf("Failed to list SQS queues: %v", err)
	}

	expectedQueues := []string{"job-requests", "jmw-queue", "jmr-queue", "sp-queue", "spa-queue", "spaq-queue"}
	for _, expectedQueue := range expectedQueues {
		found := false
		for _, queueURL := range queues.QueueUrls {
			if contains(queueURL, expectedQueue) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected queue %s not found", expectedQueue)
		}
	}
	fmt.Println("✓ All SQS queues are available")
}

func TestBasicBDDScenario(t *testing.T) {
	// This simulates a basic BDD scenario
	fmt.Println("Running basic BDD scenario...")

	// Given: LocalStack is running
	resp, err := http.Get("http://localhost:4566/_localstack/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatal("Given: LocalStack should be running")
	}
	fmt.Println("✓ Given: LocalStack is running")

	// When: I check the test service
	testResp, err := http.Get("http://localhost:8090")
	if err != nil {
		t.Fatalf("When: Failed to connect to test service: %v", err)
	}
	defer testResp.Body.Close()
	fmt.Println("✓ When: Test service is accessible")

	// Then: The service should respond successfully
	if testResp.StatusCode != http.StatusOK {
		t.Fatalf("Then: Expected status 200, got %d", testResp.StatusCode)
	}
	fmt.Println("✓ Then: Service responds successfully")

	fmt.Println("✓ Basic BDD scenario completed successfully")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}
