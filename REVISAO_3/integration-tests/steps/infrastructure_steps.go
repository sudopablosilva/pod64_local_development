package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cucumber/godog"
)

type InfrastructureContext struct {
	dynamoClient *dynamodb.Client
	sqsClient    *sqs.Client
	httpClient   *http.Client
	lastResponse *http.Response
	lastError    error
	tables       []string
	queues       []string
	testMessage  string
}

func NewInfrastructureContext() *InfrastructureContext {
	cfg, _ := config.LoadDefaultConfig(context.TODO(),
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

	return &InfrastructureContext{
		dynamoClient: dynamodb.NewFromConfig(cfg),
		sqsClient:    sqs.NewFromConfig(cfg),
		httpClient:   &http.Client{},
	}
}

func (ic *InfrastructureContext) localStackIsRunning() error {
	resp, err := ic.httpClient.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		return fmt.Errorf("LocalStack is not running: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LocalStack is not healthy: status %d", resp.StatusCode)
	}
	return nil
}

func (ic *InfrastructureContext) iCheckTheLocalStackHealthEndpoint() error {
	resp, err := ic.httpClient.Get("http://localhost:4566/_localstack/health")
	ic.lastResponse = resp
	ic.lastError = err
	return nil
}

func (ic *InfrastructureContext) localStackShouldRespondWithHealthyStatus() error {
	if ic.lastError != nil {
		return fmt.Errorf("failed to connect to LocalStack: %v", ic.lastError)
	}
	
	if ic.lastResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d", ic.lastResponse.StatusCode)
	}
	
	return nil
}

func (ic *InfrastructureContext) iListTheDynamoDBTables() error {
	result, err := ic.dynamoClient.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		ic.lastError = err
		return nil
	}
	
	ic.tables = result.TableNames
	return nil
}

func (ic *InfrastructureContext) iShouldSeeTheFollowingTables(table *godog.Table) error {
	if ic.lastError != nil {
		return fmt.Errorf("failed to list tables: %v", ic.lastError)
	}
	
	for _, row := range table.Rows[1:] { // Skip header
		expectedTable := row.Cells[0].Value
		found := false
		for _, table := range ic.tables {
			if table == expectedTable {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected table %s not found", expectedTable)
		}
	}
	
	return nil
}

func (ic *InfrastructureContext) iListTheSQSQueues() error {
	result, err := ic.sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		ic.lastError = err
		return nil
	}
	
	ic.queues = result.QueueUrls
	return nil
}

func (ic *InfrastructureContext) iShouldSeeTheFollowingQueues(table *godog.Table) error {
	if ic.lastError != nil {
		return fmt.Errorf("failed to list queues: %v", ic.lastError)
	}
	
	for _, row := range table.Rows[1:] { // Skip header
		expectedQueue := row.Cells[0].Value
		found := false
		for _, queueURL := range ic.queues {
			if contains(queueURL, expectedQueue) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected queue %s not found", expectedQueue)
		}
	}
	
	return nil
}

func (ic *InfrastructureContext) sqsQueuesAreAvailable() error {
	return ic.iListTheSQSQueues()
}

func (ic *InfrastructureContext) iSendATestMessageToTheJobRequestsQueue() error {
	testMessage := map[string]interface{}{
		"id":       "test-message-001",
		"job_name": "test-job",
		"job_type": "test",
		"priority": 1,
	}
	
	messageBody, err := json.Marshal(testMessage)
	if err != nil {
		return err
	}
	
	ic.testMessage = string(messageBody)
	
	queueURL := "http://localhost:4566/000000000000/job-requests"
	_, err = ic.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(ic.testMessage),
	})
	
	ic.lastError = err
	return nil
}

func (ic *InfrastructureContext) theMessageShouldBeSuccessfullyQueued() error {
	if ic.lastError != nil {
		return fmt.Errorf("failed to send message: %v", ic.lastError)
	}
	return nil
}

func (ic *InfrastructureContext) iShouldBeAbleToReceiveTheMessageFromTheQueue() error {
	queueURL := "http://localhost:4566/000000000000/job-requests"
	result, err := ic.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     5,
	})
	
	if err != nil {
		return fmt.Errorf("failed to receive message: %v", err)
	}
	
	if len(result.Messages) == 0 {
		return fmt.Errorf("no messages received from queue")
	}
	
	receivedMessage := *result.Messages[0].Body
	if receivedMessage != ic.testMessage {
		return fmt.Errorf("received message doesn't match sent message")
	}
	
	// Clean up - delete the message
	_, err = ic.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: result.Messages[0].ReceiptHandle,
	})
	
	return err
}

func (ic *InfrastructureContext) iVerifyDynamoDBTablesExist() error {
	tables := []string{"jobs", "schedules", "adapters", "queue_messages"}
	for _, tableName := range tables {
		_, err := ic.dynamoClient.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return fmt.Errorf("table %s does not exist: %v", tableName, err)
		}
	}
	return nil
}

func (ic *InfrastructureContext) iVerifySQSQueuesExist() error {
	queues := []string{"job-requests", "jmw-queue", "jmr-queue", "sp-queue", "spa-queue", "spaq-queue"}
	for _, queue := range queues {
		queueURL := fmt.Sprintf("http://localhost:4566/000000000000/%s", queue)
		_, err := ic.sqsClient.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(queueURL),
		})
		if err != nil {
			return fmt.Errorf("queue %s does not exist: %v", queue, err)
		}
	}
	return nil
}

func InitializeInfrastructureSteps(ctx *godog.ScenarioContext) {
	ic := NewInfrastructureContext()

	ctx.Given(`^LocalStack is running$`, ic.localStackIsRunning)
	ctx.When(`^I check the LocalStack health endpoint$`, ic.iCheckTheLocalStackHealthEndpoint)
	ctx.Then(`^LocalStack should respond with healthy status$`, ic.localStackShouldRespondWithHealthyStatus)
	ctx.When(`^I list the DynamoDB tables$`, ic.iListTheDynamoDBTables)
	ctx.Then(`^I should see the following tables:$`, ic.iShouldSeeTheFollowingTables)
	ctx.When(`^I list the SQS queues$`, ic.iListTheSQSQueues)
	ctx.Then(`^I should see the following queues:$`, ic.iShouldSeeTheFollowingQueues)
	ctx.Given(`^SQS queues are available$`, ic.sqsQueuesAreAvailable)
	ctx.When(`^I send a test message to the job-requests queue$`, ic.iSendATestMessageToTheJobRequestsQueue)
	ctx.Then(`^the message should be successfully queued$`, ic.theMessageShouldBeSuccessfullyQueued)
	ctx.Then(`^I should be able to receive the message from the queue$`, ic.iShouldBeAbleToReceiveTheMessageFromTheQueue)
	ctx.When(`^I verify DynamoDB tables exist$`, ic.iVerifyDynamoDBTablesExist)
	ctx.When(`^I verify SQS queues exist$`, ic.iVerifySQSQueuesExist)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}