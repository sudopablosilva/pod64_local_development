package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Execution represents our execution data structure
type Execution struct {
	ExecutionName string `dynamodbav:"executionName"`
	OriginalName  string `dynamodbav:"originalName"`
	ExecutionUuid string `dynamodbav:"executionUuid"`
	Status        string `dynamodbav:"status"`
	CreatedAt     string `dynamodbav:"createdAt"`
	UpdatedAt     string `dynamodbav:"updatedAt"`
	Version       int    `dynamodbav:"version"`
	Stage         string `dynamodbav:"stage"`
	ProcessedBy   string `dynamodbav:"processedBy"`
	Timestamp     int64  `dynamodbav:"timestamp"`
}

func main() {
	// Configure AWS SDK exactly like the AWS example
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: "http://localhost:4566",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"test",
			"test",
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	// Create test execution following AWS example pattern with unique key
	timestamp := time.Now().Unix()
	execution := Execution{
		ExecutionName: fmt.Sprintf("AWS_EXAMPLE_TEST_%d#v1#test-stage", timestamp),
		OriginalName:  fmt.Sprintf("AWS_EXAMPLE_TEST_%d", timestamp),
		ExecutionUuid: fmt.Sprintf("test-uuid-%d", timestamp),
		Status:        "started",
		CreatedAt:     "2025-06-13T13:30:00Z",
		UpdatedAt:     "2025-06-13T13:30:00Z",
		Version:       1,
		Stage:         "test-stage",
		ProcessedBy:   "TEST",
		Timestamp:     timestamp,
	}

	fmt.Printf("🧪 Testing AWS SDK v2 with LocalStack\n")
	fmt.Printf("Execution to insert: %+v\n", execution)

	// Use attributevalue.MarshalMap exactly like AWS example
	item, err := attributevalue.MarshalMap(execution)
	if err != nil {
		log.Fatalf("Failed to marshal execution: %v", err)
	}

	fmt.Printf("Marshaled item: %+v\n", item)

	// Use PutItem exactly like AWS example
	_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("executions"),
		Item:      item,
	})
	if err != nil {
		log.Fatalf("Failed to put item: %v", err)
	}

	fmt.Printf("✅ Successfully inserted item using AWS SDK v2 pattern!\n")

	// Verify the item was inserted
	result, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("executions"),
		Key: map[string]types.AttributeValue{
			"executionName": &types.AttributeValueMemberS{Value: execution.ExecutionName},
		},
	})
	if err != nil {
		log.Fatalf("Failed to get item: %v", err)
	}

	if result.Item == nil {
		fmt.Printf("❌ Item not found after insertion!\n")
		os.Exit(1)
	}

	// Unmarshal the result
	var retrievedExecution Execution
	err = attributevalue.UnmarshalMap(result.Item, &retrievedExecution)
	if err != nil {
		log.Fatalf("Failed to unmarshal result: %v", err)
	}

	fmt.Printf("✅ Successfully retrieved item: %+v\n", retrievedExecution)
	fmt.Printf("🎉 AWS SDK v2 pattern works with LocalStack!\n")
}
