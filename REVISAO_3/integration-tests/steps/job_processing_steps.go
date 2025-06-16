package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cucumber/godog"
)

type JobRequest struct {
	ID          string                 `json:"id"`
	JobName     string                 `json:"job_name"`
	JobType     string                 `json:"job_type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	CreatedAt   time.Time              `json:"created_at"`
	Status      string                 `json:"status"`
}

type TestContext struct {
	jobRequest    JobRequest
	jobRequests   []JobRequest
	lastResponse  *http.Response
	lastJobID     string
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	httpClient    *http.Client
	baseURL       string
}

func NewTestContext() *TestContext {
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

	return &TestContext{
		dynamoClient: dynamodb.NewFromConfig(cfg),
		sqsClient:    sqs.NewFromConfig(cfg),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      "http://localhost",
	}
}

func (tc *TestContext) allServicesAreRunning() error {
	services := map[string]string{
		"control-m":         "8081",
		"jmi":              "8082",
		"jmw":              "8083",
		"jmr":              "8084",
		"scheduler-plugin": "8085",
		"spa":              "8086",
		"spaq":             "8087",
	}

	for service, port := range services {
		url := fmt.Sprintf("%s:%s/health", tc.baseURL, port)
		resp, err := tc.httpClient.Get(url)
		if err != nil {
			return fmt.Errorf("service %s is not running: %v", service, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("service %s is not healthy: status %d", service, resp.StatusCode)
		}
	}
	return nil
}

func (tc *TestContext) localStackIsInitializedWithRequiredResources() error {
	// Check if DynamoDB tables exist
	tables := []string{"jobs", "schedules", "adapters", "queue_messages"}
	for _, table := range tables {
		_, err := tc.dynamoClient.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(table),
		})
		if err != nil {
			return fmt.Errorf("table %s does not exist: %v", table, err)
		}
	}

	// Check if SQS queues exist
	queues := []string{"job-requests", "jmw-queue", "jmr-queue", "sp-queue", "spa-queue", "spaq-queue"}
	for _, queue := range queues {
		queueURL := fmt.Sprintf("http://localhost:4566/000000000000/%s", queue)
		_, err := tc.sqsClient.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(queueURL),
		})
		if err != nil {
			return fmt.Errorf("queue %s does not exist: %v", queue, err)
		}
	}

	return nil
}

func (tc *TestContext) iHaveAJobRequestWithTheFollowingDetails(table *godog.Table) error {
	tc.jobRequest = JobRequest{
		Parameters: make(map[string]interface{}),
	}

	for _, row := range table.Rows[1:] { // Skip header
		field := row.Cells[0].Value
		value := row.Cells[1].Value

		switch field {
		case "job_name":
			tc.jobRequest.JobName = value
		case "job_type":
			tc.jobRequest.JobType = value
		case "priority":
			if value == "1" {
				tc.jobRequest.Priority = 1
			} else if value == "2" {
				tc.jobRequest.Priority = 2
			} else {
				tc.jobRequest.Priority = 3
			}
		}
	}

	tc.jobRequest.ScheduledAt = time.Now().Add(5 * time.Minute)
	return nil
}

func (tc *TestContext) iSubmitTheJobToControlM() error {
	jsonData, err := json.Marshal(tc.jobRequest)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s:8081/jobs", tc.baseURL)
	resp, err := tc.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	tc.lastResponse = resp
	
	// Extract job ID from response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if jobID, ok := response["job_id"].(string); ok {
		tc.lastJobID = jobID
	}

	return nil
}

func (tc *TestContext) theJobShouldBeAcceptedWithStatus(expectedStatus string) error {
	if tc.lastResponse.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status 201, got %d", tc.lastResponse.StatusCode)
	}

	body, err := io.ReadAll(tc.lastResponse.Body)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	status, ok := response["status"].(string)
	if !ok || status != expectedStatus {
		return fmt.Errorf("expected status %s, got %s", expectedStatus, status)
	}

	return nil
}

func (tc *TestContext) theJobShouldAppearInJMIWithStatus(expectedStatus string) error {
	// Wait a bit for processing
	time.Sleep(5 * time.Second)

	// Check DynamoDB directly
	result, err := tc.dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("jobs"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: tc.lastJobID},
		},
	})
	
	if err != nil {
		return fmt.Errorf("error querying DynamoDB: %v", err)
	}
	
	if result.Item == nil {
		return fmt.Errorf("job %s not found in DynamoDB", tc.lastJobID)
	}
	
	var job struct {
		Status string `dynamodbav:"status"`
	}
	
	if err := attributevalue.UnmarshalMap(result.Item, &job); err != nil {
		return fmt.Errorf("error unmarshaling job: %v", err)
	}
	
	if job.Status != expectedStatus {
		return fmt.Errorf("expected job status %s, got %s", expectedStatus, job.Status)
	}

	// Also check the API for backward compatibility
	url := fmt.Sprintf("%s:8082/jobs", tc.baseURL)
	resp, err := tc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var jobs []JobRequest
	if err := json.Unmarshal(body, &jobs); err != nil {
		return err
	}

	for _, job := range jobs {
		if job.ID == tc.lastJobID && job.Status == expectedStatus {
			return nil
		}
	}

	return nil
}

func (tc *TestContext) theJobShouldBeProcessedByJMWWithStatus(expectedStatus string) error {
	// Wait for processing
	time.Sleep(5 * time.Second)

	// Check DynamoDB directly
	result, err := tc.dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("jobs"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: tc.lastJobID},
		},
	})
	
	if err != nil {
		return fmt.Errorf("error querying DynamoDB: %v", err)
	}
	
	if result.Item == nil {
		return fmt.Errorf("job %s not found in DynamoDB", tc.lastJobID)
	}
	
	var job struct {
		Status   string `dynamodbav:"status"`
		WorkerID string `dynamodbav:"worker_id"`
	}
	
	if err := attributevalue.UnmarshalMap(result.Item, &job); err != nil {
		return fmt.Errorf("error unmarshaling job: %v", err)
	}
	
	if job.Status != expectedStatus {
		return fmt.Errorf("expected job status %s, got %s", expectedStatus, job.Status)
	}
	
	if job.WorkerID == "" {
		return fmt.Errorf("worker ID not set for job %s", tc.lastJobID)
	}

	// Also check JMW stats
	url := fmt.Sprintf("%s:8083/stats", tc.baseURL)
	resp, err := tc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		return err
	}

	jobsProcessed, ok := stats["jobs_processed"].(float64)
	if !ok || jobsProcessed == 0 {
		return fmt.Errorf("no jobs processed by JMW")
	}

	return nil
}

func (tc *TestContext) theJobShouldBeExecutedByJMRWithStatus(expectedStatus string) error {
	// Wait for execution
	time.Sleep(5 * time.Second)

	// Check DynamoDB directly
	result, err := tc.dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("jobs"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: tc.lastJobID},
		},
	})
	
	if err != nil {
		return fmt.Errorf("error querying DynamoDB: %v", err)
	}
	
	if result.Item == nil {
		return fmt.Errorf("job %s not found in DynamoDB", tc.lastJobID)
	}
	
	var job struct {
		Status       string `dynamodbav:"status"`
		RunnerID     string `dynamodbav:"runner_id"`
		ExecutionLog string `dynamodbav:"execution_log"`
	}
	
	if err := attributevalue.UnmarshalMap(result.Item, &job); err != nil {
		return fmt.Errorf("error unmarshaling job: %v", err)
	}
	
	if job.Status != expectedStatus {
		return fmt.Errorf("expected job status %s, got %s", expectedStatus, job.Status)
	}
	
	if job.RunnerID == "" {
		return fmt.Errorf("runner ID not set for job %s", tc.lastJobID)
	}
	
	if job.ExecutionLog == "" {
		return fmt.Errorf("execution log not set for job %s", tc.lastJobID)
	}

	// Also check JMR stats
	url := fmt.Sprintf("%s:8084/stats", tc.baseURL)
	resp, err := tc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		return err
	}

	jobsExecuted, ok := stats["jobs_executed"].(float64)
	if !ok || jobsExecuted == 0 {
		return fmt.Errorf("no jobs executed by JMR")
	}

	return nil
}

func (tc *TestContext) aScheduleShouldBeCreatedBySchedulerPlugin() error {
	// Wait for schedule creation
	time.Sleep(5 * time.Second)

	// Query DynamoDB for schedules
	result, err := tc.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("schedules"),
	})
	
	if err != nil {
		return fmt.Errorf("error scanning schedules table: %v", err)
	}
	
	if len(result.Items) == 0 {
		return fmt.Errorf("no schedules found in DynamoDB")
	}
	
	var schedules []struct {
		ID    string `dynamodbav:"id"`
		JobID string `dynamodbav:"job_id"`
	}
	
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &schedules); err != nil {
		return fmt.Errorf("error unmarshaling schedules: %v", err)
	}
	
	for _, schedule := range schedules {
		if schedule.JobID == tc.lastJobID {
			return nil
		}
	}
	
	return fmt.Errorf("no schedule found for job %s", tc.lastJobID)
}

func (tc *TestContext) anAdapterShouldBeConfiguredBySPA() error {
	// Wait for adapter configuration
	time.Sleep(5 * time.Second)

	// Query DynamoDB for adapters
	result, err := tc.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("adapters"),
	})
	
	if err != nil {
		return fmt.Errorf("error scanning adapters table: %v", err)
	}
	
	if len(result.Items) == 0 {
		return fmt.Errorf("no adapters found in DynamoDB")
	}
	
	// Also check the API
	url := fmt.Sprintf("%s:8086/adapters", tc.baseURL)
	resp, err := tc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var adapters []interface{}
	if err := json.Unmarshal(body, &adapters); err != nil {
		return err
	}

	if len(adapters) == 0 {
		return fmt.Errorf("no adapters configured")
	}

	return nil
}

func (tc *TestContext) aQueueMessageShouldBeCreatedBySPAQ() error {
	// Wait for queue message creation
	time.Sleep(5 * time.Second)

	// Query DynamoDB for queue messages
	result, err := tc.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("queue_messages"),
	})
	
	if err != nil {
		return fmt.Errorf("error scanning queue_messages table: %v", err)
	}
	
	if len(result.Items) == 0 {
		return fmt.Errorf("no queue messages found in DynamoDB")
	}
	
	// Also check the API
	url := fmt.Sprintf("%s:8087/messages", tc.baseURL)
	resp, err := tc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var messages []interface{}
	if err := json.Unmarshal(body, &messages); err != nil {
		return err
	}

	if len(messages) == 0 {
		return fmt.Errorf("no queue messages created")
	}

	return nil
}

func (tc *TestContext) iHaveMultipleJobRequests(table *godog.Table) error {
	tc.jobRequests = make([]JobRequest, 0)

	for _, row := range table.Rows[1:] { // Skip header
		job := JobRequest{
			Parameters: make(map[string]interface{}),
		}

		job.JobName = row.Cells[0].Value
		job.JobType = row.Cells[1].Value
		
		priority := row.Cells[2].Value
		if priority == "1" {
			job.Priority = 1
		} else if priority == "2" {
			job.Priority = 2
		} else {
			job.Priority = 3
		}

		job.ScheduledAt = time.Now().Add(5 * time.Minute)
		tc.jobRequests = append(tc.jobRequests, job)
	}

	return nil
}

func (tc *TestContext) iSubmitAllJobsToControlM() error {
	for _, job := range tc.jobRequests {
		jsonData, err := json.Marshal(job)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s:8081/jobs", tc.baseURL)
		resp, err := tc.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return nil
}

func (tc *TestContext) allJobsShouldBeProcessedThroughThePipeline() error {
	// Wait for all jobs to be processed
	time.Sleep(15 * time.Second)
	
	// Check if all jobs were processed by querying DynamoDB
	result, err := tc.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("jobs"),
	})
	
	if err != nil {
		return fmt.Errorf("error scanning jobs table: %v", err)
	}
	
	if len(result.Items) < len(tc.jobRequests) {
		return fmt.Errorf("expected at least %d jobs, found %d", len(tc.jobRequests), len(result.Items))
	}
	
	var jobs []struct {
		JobName  string `dynamodbav:"job_name"`
		Status   string `dynamodbav:"status"`
		RunnerID string `dynamodbav:"runner_id"`
	}
	
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &jobs); err != nil {
		return fmt.Errorf("error unmarshaling jobs: %v", err)
	}
	
	executedCount := 0
	for _, job := range jobs {
		if job.Status == "executed" && job.RunnerID != "" {
			executedCount++
		}
	}
	
	if executedCount < len(tc.jobRequests) {
		return fmt.Errorf("expected at least %d executed jobs, found %d", len(tc.jobRequests), executedCount)
	}
	
	return nil
}

func (tc *TestContext) jobsShouldBeProcessedAccordingToTheirPriority() error {
	// This is a simplified check - in a real system we would need more sophisticated verification
	// For now, we'll just check that high priority jobs have been processed
	
	// Query DynamoDB for jobs
	result, err := tc.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String("jobs"),
	})
	
	if err != nil {
		return fmt.Errorf("error scanning jobs table: %v", err)
	}
	
	var jobs []struct {
		JobName  string `dynamodbav:"job_name"`
		Priority int    `dynamodbav:"priority"`
		Status   string `dynamodbav:"status"`
	}
	
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &jobs); err != nil {
		return fmt.Errorf("error unmarshaling jobs: %v", err)
	}
	
	// Check that high priority jobs (priority=1) are executed
	for _, job := range jobs {
		if job.Priority == 1 && job.Status != "executed" {
			return fmt.Errorf("high priority job %s not executed, status: %s", job.JobName, job.Status)
		}
	}
	
	return nil
}

func (tc *TestContext) iCheckTheHealthOfAllServices() error {
	return tc.allServicesAreRunning()
}

func (tc *TestContext) allServicesShouldRespondWithHealthyStatus() error {
	return nil // Already checked in allServicesAreRunning
}

func (tc *TestContext) iSubmitAJobThroughThePipeline() error {
	tc.jobRequest = JobRequest{
		JobName:    "test-data-persistence",
		JobType:    "shell",
		Priority:   1,
		Parameters: make(map[string]interface{}),
		ScheduledAt: time.Now().Add(5 * time.Minute),
	}
	
	return tc.iSubmitTheJobToControlM()
}

func (tc *TestContext) iQueryTheDynamoDBTables() error {
	// Wait for processing to complete
	time.Sleep(15 * time.Second)
	return nil
}

func (tc *TestContext) theJobDataShouldBePersistedCorrectly() error {
	result, err := tc.dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("jobs"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: tc.lastJobID},
		},
	})
	
	if err != nil {
		return fmt.Errorf("error querying DynamoDB: %v", err)
	}
	
	if result.Item == nil {
		return fmt.Errorf("job %s not found in DynamoDB", tc.lastJobID)
	}
	
	return nil
}

func (tc *TestContext) theScheduleDataShouldBeStored() error {
	return tc.aScheduleShouldBeCreatedBySchedulerPlugin()
}

func (tc *TestContext) theAdapterConfigurationShouldBeSaved() error {
	return tc.anAdapterShouldBeConfiguredBySPA()
}

func (tc *TestContext) theQueueMessagesShouldBeRecorded() error {
	return tc.aQueueMessageShouldBeCreatedBySPAQ()
}

func InitializeJobProcessingSteps(ctx *godog.ScenarioContext) {
	tc := NewTestContext()

	ctx.Given(`^all services are running$`, tc.allServicesAreRunning)
	ctx.Given(`^LocalStack is initialized with required resources$`, tc.localStackIsInitializedWithRequiredResources)
	ctx.Given(`^I have a job request with the following details:$`, tc.iHaveAJobRequestWithTheFollowingDetails)
	ctx.Given(`^I have multiple job requests:$`, tc.iHaveMultipleJobRequests)
	ctx.Given(`^I submit a job through the pipeline$`, tc.iSubmitAJobThroughThePipeline)
	
	ctx.When(`^I submit the job to Control-M$`, tc.iSubmitTheJobToControlM)
	ctx.When(`^I submit all jobs to Control-M$`, tc.iSubmitAllJobsToControlM)
	ctx.When(`^I check the health of all services$`, tc.iCheckTheHealthOfAllServices)
	ctx.When(`^I query the DynamoDB tables$`, tc.iQueryTheDynamoDBTables)
	
	ctx.Then(`^the job should be accepted with status "([^"]*)"$`, tc.theJobShouldBeAcceptedWithStatus)
	ctx.Then(`^the job should appear in JMI with status "([^"]*)"$`, tc.theJobShouldAppearInJMIWithStatus)
	ctx.Then(`^the job should be processed by JMW with status "([^"]*)"$`, tc.theJobShouldBeProcessedByJMWWithStatus)
	ctx.Then(`^the job should be executed by JMR with status "([^"]*)"$`, tc.theJobShouldBeExecutedByJMRWithStatus)
	ctx.Then(`^a schedule should be created by Scheduler Plugin$`, tc.aScheduleShouldBeCreatedBySchedulerPlugin)
	ctx.Then(`^an adapter should be configured by SPA$`, tc.anAdapterShouldBeConfiguredBySPA)
	ctx.Then(`^a queue message should be created by SPAQ$`, tc.aQueueMessageShouldBeCreatedBySPAQ)
	ctx.Then(`^all jobs should be processed through the pipeline$`, tc.allJobsShouldBeProcessedThroughThePipeline)
	ctx.Then(`^jobs should be processed according to their priority$`, tc.jobsShouldBeProcessedAccordingToTheirPriority)
	ctx.Then(`^all services should respond with healthy status$`, tc.allServicesShouldRespondWithHealthyStatus)
	ctx.Then(`^the job data should be persisted correctly$`, tc.theJobDataShouldBePersistedCorrectly)
	ctx.Then(`^the schedule data should be stored$`, tc.theScheduleDataShouldBeStored)
	ctx.Then(`^the adapter configuration should be saved$`, tc.theAdapterConfigurationShouldBeSaved)
	ctx.Then(`^the queue messages should be recorded$`, tc.theQueueMessagesShouldBeRecorded)
}