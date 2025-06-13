package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExecutionData represents our execution data structure following AWS pattern
type ExecutionData struct {
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

// applyProcessingDelay aplica uma latência artificial baseada na variável de ambiente
func applyProcessingDelay() {
	delayStr := os.Getenv("PROCESSING_DELAY_MS")
	if delayStr == "" {
		return
	}
	
	delayMs, err := strconv.Atoi(delayStr)
	if err != nil || delayMs <= 0 {
		return
	}
	
	log.Printf("Applying artificial processing delay: %dms", delayMs)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

// insertDirectToDynamoDB insere dados diretamente via HTTP para contornar problemas do SDK
func insertDirectToDynamoDB(tableName, executionKey, originalName, executionUuid, status, createdAt, updatedAt, stage, processedBy string, version int, timestamp int64) error {
	dynamoEndpoint := os.Getenv("AWS_ENDPOINT")
	if dynamoEndpoint == "" {
		dynamoEndpoint = "http://localstack:4566" // Use service name instead of localhost
	}

	log.Printf("DEBUG: Using DynamoDB endpoint: %s", dynamoEndpoint)

	// Criar payload DynamoDB diretamente
	payload := map[string]interface{}{
		"TableName": tableName,
		"Item": map[string]interface{}{
			"executionName": map[string]string{"S": executionKey},
			"originalName":  map[string]string{"S": originalName},
			"executionUuid": map[string]string{"S": executionUuid},
			"status":        map[string]string{"S": status},
			"createdAt":     map[string]string{"S": createdAt},
			"updatedAt":     map[string]string{"S": updatedAt},
			"version":       map[string]string{"N": fmt.Sprintf("%d", version)},
			"stage":         map[string]string{"S": stage},
			"processedBy":   map[string]string{"S": processedBy},
			"timestamp":     map[string]string{"N": fmt.Sprintf("%d", timestamp)},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	log.Printf("DEBUG: HTTP payload: %s", string(payloadBytes))

	// Fazer requisição HTTP direta
	req, err := http.NewRequest("POST", dynamoEndpoint+"/", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	respBody := make([]byte, 1024)
	n, _ := resp.Body.Read(respBody)
	log.Printf("DEBUG: HTTP response status: %d, body: %s", resp.StatusCode, string(respBody[:n]))

	if resp.StatusCode != 200 {
		return fmt.Errorf("DynamoDB returned status %d: %s", resp.StatusCode, string(respBody[:n]))
	}

	log.Printf("DEBUG: Direct HTTP insertion successful")
	return nil
}

// Execution represents a job execution request
type Execution struct {
	ExecutionName    string                 `json:"executionName" dynamodbav:"executionName"`
	ExecutionUuid    string                 `json:"executionUuid" dynamodbav:"executionUuid"`
	AccountId        string                 `json:"accountId" dynamodbav:"accountId"`
	CommonProperties map[string]interface{} `json:"commonProperties" dynamodbav:"commonProperties"`
	Runtimes         []Runtime              `json:"runtimes" dynamodbav:"runtimes"`
	SchedulerRoutine SchedulerRoutine       `json:"schedulerRoutine" dynamodbav:"schedulerRoutine"`
	Status           string                 `json:"status" dynamodbav:"status"`
	CreatedAt        time.Time              `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt" dynamodbav:"updatedAt"`
	Retake           *RetakeInfo            `json:"retake,omitempty" dynamodbav:"retake,omitempty"`
}

type Runtime struct {
	RuntimeName string                 `json:"runtimeName" dynamodbav:"runtimeName"`
	Compute     map[string]interface{} `json:"compute" dynamodbav:"compute"`
	Security    map[string]interface{} `json:"security" dynamodbav:"security"`
	Tags        map[string]interface{} `json:"tags" dynamodbav:"tags"`
}

type SchedulerRoutine struct {
	ExecutionName string `json:"executionName" dynamodbav:"executionName"`
	Cron          string `json:"cron" dynamodbav:"cron"`
	DependsOn     string `json:"dependsOn" dynamodbav:"dependsOn"`
	Priority      string `json:"priority" dynamodbav:"priority"`
	Provisioning  string `json:"provisioning" dynamodbav:"provisioning"`
	Steps         []Step `json:"steps" dynamodbav:"steps"`
}

type Step struct {
	StepId string `json:"stepId" dynamodbav:"stepId"`
	Tasks  []Task `json:"tasks" dynamodbav:"tasks"`
}

type Task struct {
	TaskId      string                 `json:"taskId" dynamodbav:"taskId"`
	RuntimeName string                 `json:"runtimeName" dynamodbav:"runtimeName"`
	Parameters  map[string]interface{} `json:"parameters" dynamodbav:"parameters"`
}

type RetakeInfo struct {
	FromStepId      string   `json:"fromStepId" dynamodbav:"fromStepId"`
	ExcludingTasks  []string `json:"excludingTasks" dynamodbav:"excludingTasks"`
}

type StartExecutionRequest struct {
	ExecutionName string      `json:"executionName"`
	Retake        *RetakeInfo `json:"retake,omitempty"`
}

type StopExecutionRequest struct {
	ExecutionName string `json:"executionName"`
	ExecutionUuid string `json:"executionUuid"`
}

// Legacy Job struct for backward compatibility
type Job struct {
	ID          string                 `json:"id" dynamodbav:"id"`
	JobName     string                 `json:"job_name" dynamodbav:"job_name"`
	JobType     string                 `json:"job_type" dynamodbav:"job_type"`
	Parameters  map[string]interface{} `json:"parameters" dynamodbav:"parameters"`
	Priority    int                    `json:"priority" dynamodbav:"priority"`
	ScheduledAt time.Time              `json:"scheduled_at" dynamodbav:"scheduled_at"`
	CreatedAt   time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	Status      string                 `json:"status" dynamodbav:"status"`
}

type JMIService struct {
	jobs          []Job
	executions    []Execution
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	executionTable string
	inQueueURL    string
	outQueueURL   string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewJMIService() *JMIService {
	// Configure AWS SDK with the WORKING configuration from dynamodb-test
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"), // LocalStack usa us-east-1 por padrão
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"", // Session Token não necessário para LocalStack
		)),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				endpoint := os.Getenv("AWS_ENDPOINT")
				if endpoint == "" {
					endpoint = "http://localstack:4566"
				}
				// CORREÇÃO: Adicionar SigningRegion que estava faltando
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: "us-east-1", // Esta linha estava faltando!
				}, nil
			})),
	)

	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &JMIService{
		jobs:          make([]Job, 0),
		executions:    make([]Execution, 0),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		executionTable: os.Getenv("EXECUTION_TABLE"),
		inQueueURL:    os.Getenv("SQS_QUEUE_URL"),
		outQueueURL:   os.Getenv("JMW_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (j *JMIService) startMessageReceiver() {
	for {
		select {
		case <-j.receiveCtx.Done():
			log.Println("Message receiver stopped")
			return
		default:
			// Receive messages from SQS
			result, err := j.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(j.inQueueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20, // Long polling
			})

			if err != nil {
				log.Printf("Error receiving messages: %v", err)
				time.Sleep(5 * time.Second) // Wait before retrying
				continue
			}

			for _, message := range result.Messages {
				// Process the message
				j.processMessage(*message.Body)

				// Delete the message from the queue
				_, err := j.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(j.inQueueURL),
					ReceiptHandle: message.ReceiptHandle,
				})

				if err != nil {
					log.Printf("Error deleting message: %v", err)
				}
			}
		}
	}
}

func (j *JMIService) processMessage(messageBody string) {
	var job Job
	if err := json.Unmarshal([]byte(messageBody), &job); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// Update job status
	job.Status = "integrated"
	job.UpdatedAt = time.Now()

	// Store job in DynamoDB
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		log.Printf("Error marshaling job: %v", err)
		return
	}

	_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(j.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing job in DynamoDB: %v", err)
		return
	}

	// Add to local cache
	j.jobs = append(j.jobs, job)

	// Forward to JMW queue
	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job for JMW: %v", err)
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to JMW queue: %v", err)
		return
	}

	log.Printf("JMI processed job %s and forwarded to JMW", job.ID)
}

func (j *JMIService) StopExecution(ctx *gin.Context) {
	var req StopExecutionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find execution in DynamoDB
	tableName := j.executionTable
	if tableName == "" {
		tableName = "executions"
	}

	result, err := j.dynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"executionName": &types.AttributeValueMemberS{Value: req.ExecutionName},
		},
	})

	if err != nil {
		log.Printf("Error getting execution from DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find execution"})
		return
	}

	if result.Item == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	var execution Execution
	err = attributevalue.UnmarshalMap(result.Item, &execution)
	if err != nil {
		log.Printf("Error unmarshaling execution: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution data"})
		return
	}

	// Verify UUID matches
	if execution.ExecutionUuid != req.ExecutionUuid {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Execution UUID mismatch"})
		return
	}

	// Update execution status
	execution.Status = "stopped"
	execution.UpdatedAt = time.Now()

	// Store updated execution in DynamoDB
	item, err := attributevalue.MarshalMap(execution)
	if err != nil {
		log.Printf("Error marshaling execution: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution"})
		return
	}

	_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error updating execution in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update execution"})
		return
	}

	log.Printf("JMI stopped execution %s with UUID %s", execution.ExecutionName, execution.ExecutionUuid)

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Execution stopped successfully",
		"executionName": execution.ExecutionName,
		"executionUuid": execution.ExecutionUuid,
		"status":        execution.Status,
	})
}

func (j *JMIService) GetQueues(ctx *gin.Context) {
	log.Printf("DEBUG: Listing SQS queues")
	
	// List queues using AWS SDK (replacement for SQS queue monitoring)
	listOutput, err := j.sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		log.Printf("ERROR: Failed to list queues: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list queues"})
		return
	}

	// Get attributes for each queue
	var queueDetails []map[string]interface{}
	for _, queueUrl := range listOutput.QueueUrls {
		// Extract queue name from URL
		queueName := queueUrl[strings.LastIndex(queueUrl, "/")+1:]
		
		// Get queue attributes
		attrs, err := j.sqsClient.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(queueUrl),
			AttributeNames: []sqstypes.QueueAttributeName{
				sqstypes.QueueAttributeNameApproximateNumberOfMessages,
				sqstypes.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			},
		})
		
		queueInfo := map[string]interface{}{
			"name": queueName,
			"url":  queueUrl,
		}
		
		if err == nil && attrs.Attributes != nil {
			if visibleCount, ok := attrs.Attributes[string(sqstypes.QueueAttributeNameApproximateNumberOfMessages)]; ok {
				queueInfo["visibleMessages"] = visibleCount
			}
			if notVisibleCount, ok := attrs.Attributes[string(sqstypes.QueueAttributeNameApproximateNumberOfMessagesNotVisible)]; ok {
				queueInfo["notVisibleMessages"] = notVisibleCount
			}
		} else {
			queueInfo["error"] = "Failed to get attributes"
		}
		
		queueDetails = append(queueDetails, queueInfo)
	}

	log.Printf("DEBUG: Found %d queues", len(listOutput.QueueUrls))
	
	ctx.JSON(http.StatusOK, gin.H{
		"queues":  queueDetails,
		"count":   len(listOutput.QueueUrls),
		"service": "jmi",
	})
}

func (j *JMIService) GetTables(ctx *gin.Context) {
	log.Printf("DEBUG: Listing DynamoDB tables")
	
	// List tables using AWS SDK (replacement for awslocal dynamodb list-tables)
	listOutput, err := j.dynamoClient.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		log.Printf("ERROR: Failed to list tables: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tables"})
		return
	}

	log.Printf("DEBUG: Found %d tables", len(listOutput.TableNames))
	
	ctx.JSON(http.StatusOK, gin.H{
		"tables": listOutput.TableNames,
		"count":  len(listOutput.TableNames),
		"service": "jmi",
	})
}

func (j *JMIService) GetExecutions(ctx *gin.Context) {
	tableName := "executions"
	
	log.Printf("DEBUG: Listing executions from table: %s", tableName)
	
	// Scan table using AWS SDK (same pattern as dynamodb-test)
	scanOutput, err := j.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Printf("ERROR: Failed to scan executions table: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list executions"})
		return
	}

	// Unmarshal results
	var executions []ExecutionData
	err = attributevalue.UnmarshalListOfMaps(scanOutput.Items, &executions)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal executions: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process executions"})
		return
	}

	log.Printf("DEBUG: Found %d executions in table via AWS SDK", len(executions))
	
	ctx.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"count":      len(executions),
		"table":      tableName,
		"service":    "jmi",
	})
}

func (j *JMIService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":        "jmi",
		"status":         "healthy",
		"timestamp":      time.Now(),
		"jobs_processed": len(j.jobs),
	})
}

func (j *JMIService) GetJobs(ctx *gin.Context) {
	// Query DynamoDB for all jobs
	result, err := j.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(j.tableName),
	})

	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve jobs"})
		return
	}

	var jobs []Job
	err = attributevalue.UnmarshalListOfMaps(result.Items, &jobs)
	if err != nil {
		log.Printf("Error unmarshaling jobs: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process jobs data"})
		return
	}

	ctx.JSON(http.StatusOK, jobs)
}

func (j *JMIService) StartExecution(ctx *gin.Context) {
	var req StartExecutionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply artificial processing delay if configured
	applyProcessingDelay()

	// Generate execution UUID
	executionUuid := uuid.New().String()

	// Create simple execution record for basic start execution with versioning
	now := time.Now()
	execution := map[string]interface{}{
		"executionName": req.ExecutionName,
		"executionUuid": executionUuid,
		"status":        "started",
		"createdAt":     now.Format(time.RFC3339),
		"updatedAt":     now.Format(time.RFC3339),
		"version":       1,
		"stage":         "jmi-start",
		"processedBy":   "JMI",
		"timestamp":     now.Unix(),
	}

	// Add retake info if provided
	if req.Retake != nil {
		execution["retake"] = req.Retake
	}

	tableName := j.executionTable
	if tableName == "" {
		tableName = "executions"
	}

	log.Printf("DEBUG: Storing execution in table: %s", tableName)
	log.Printf("DEBUG: Execution data: %+v", execution)

	// Create composite key for versioning: executionName#version#stage
	executionKey := fmt.Sprintf("%s#v%d#%s", execution["executionName"], execution["version"], execution["stage"])
	log.Printf("DEBUG: Generated execution key: %s", executionKey)

	log.Printf("DEBUG: About to store in DynamoDB using WORKING pattern from dynamodb-test")
	log.Printf("DEBUG: Table name: %s", tableName)
	log.Printf("DEBUG: Item key: %s", executionKey)
	
	// Create ExecutionData struct following the WORKING pattern
	executionStruct := ExecutionData{
		ExecutionName: executionKey,
		OriginalName:  execution["executionName"].(string),
		ExecutionUuid: execution["executionUuid"].(string),
		Status:        execution["status"].(string),
		CreatedAt:     execution["createdAt"].(string),
		UpdatedAt:     execution["updatedAt"].(string),
		Version:       execution["version"].(int),
		Stage:         execution["stage"].(string),
		ProcessedBy:   execution["processedBy"].(string),
		Timestamp:     execution["timestamp"].(int64),
	}

	log.Printf("DEBUG: Execution struct: %+v", executionStruct)

	// Use attributevalue.MarshalMap exactly like the WORKING dynamodb-test
	item, err := attributevalue.MarshalMap(executionStruct)
	if err != nil {
		log.Printf("ERROR: Failed to marshal execution: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution"})
		return
	}

	log.Printf("DEBUG: Marshaled item successfully with %d fields", len(item))

	// Use PutItem exactly like the WORKING dynamodb-test
	_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("ERROR: Failed to store execution in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store execution"})
		return
	}

	log.Printf("DEBUG: Successfully stored execution in DynamoDB using WORKING pattern")

	// Forward to JMW queue
	executionJSON, err := json.Marshal(execution)
	if err != nil {
		log.Printf("Error marshaling execution for JMW: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution"})
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(executionJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to JMW queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward execution"})
		return
	}

	log.Printf("JMI started execution %s with UUID %s", execution["executionName"], execution["executionUuid"])

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Execution started successfully",
		"executionName": execution["executionName"],
		"executionUuid": execution["executionUuid"],
		"status":        execution["status"],
	})
}

func (j *JMIService) ProcessJob(ctx *gin.Context) {
	var job Job
	if err := ctx.ShouldBindJSON(&job); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update job status
	job.Status = "integrated"
	job.UpdatedAt = time.Now()

	// Store job in DynamoDB
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		log.Printf("Error marshaling job: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process job"})
		return
	}

	_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(j.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing job in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store job"})
		return
	}

	// Add to local cache
	j.jobs = append(j.jobs, job)

	// Forward to JMW queue
	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job for JMW: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process job"})
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to JMW queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward job"})
		return
	}

	log.Printf("JMI processed job %s", job.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Job integrated successfully",
		"job_id":  job.ID,
		"status":  job.Status,
	})
}

func main() {
	service := NewJMIService()

	r := gin.Default()

	// List queues endpoint (replacement for SQS queue monitoring)
	r.GET("/queues", service.GetQueues)
	
	// List tables endpoint (replacement for awslocal dynamodb list-tables)
	r.GET("/tables", service.GetTables)
	
	// List executions endpoint (following dynamodb-test pattern)
	r.GET("/executions", service.GetExecutions)

	// Health check
	r.GET("/health", service.GetHealth)

	// Execution endpoints (new)
	r.POST("/startExecution", service.StartExecution)
	r.POST("/stopExecution", service.StopExecution)

	// Job endpoints (legacy)
	r.GET("/jobs", service.GetJobs)
	r.POST("/process", service.ProcessJob)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("JMI service starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}