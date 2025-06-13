package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
	
	log.Printf("JMW: Applying artificial processing delay: %dms", delayMs)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

// StartRequest represents the payload from startRoutine.sh
type StartRequest struct {
	ExecutionName    string                 `json:"executionName"`
	AccountId        string                 `json:"accountId"`
	CommonProperties map[string]interface{} `json:"commonProperties"`
	Runtimes         []Runtime              `json:"runtimes"`
	SchedulerRoutine SchedulerRoutine       `json:"schedulerRoutine"`
}

type Runtime struct {
	RuntimeName string                 `json:"runtimeName"`
	Compute     map[string]interface{} `json:"compute"`
	Security    map[string]interface{} `json:"security"`
	Tags        map[string]interface{} `json:"tags"`
}

type SchedulerRoutine struct {
	ExecutionName string `json:"executionName"`
	Cron          string `json:"cron"`
	DependsOn     string `json:"dependsOn"`
	Priority      string `json:"priority"`
	Provisioning  string `json:"provisioning"`
	Steps         []Step `json:"steps"`
}

type Step struct {
	StepId string `json:"stepId"`
	Tasks  []Task `json:"tasks"`
}

type Task struct {
	TaskId      string                 `json:"taskId"`
	RuntimeName string                 `json:"runtimeName"`
	Parameters  map[string]interface{} `json:"parameters"`
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
	WorkerID    string                 `json:"worker_id" dynamodbav:"worker_id"`
}

type JMWService struct {
	jobs          []Job
	workerID      string
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	inQueueURL    string
	outQueueURL   string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewJMWService() *JMWService {
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

	service := &JMWService{
		jobs:          make([]Job, 0),
		workerID:      "jmw-" + time.Now().Format("20060102150405"),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		inQueueURL:    os.Getenv("JMW_QUEUE_URL"),
		outQueueURL:   os.Getenv("JMR_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (j *JMWService) startMessageReceiver() {
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

func (j *JMWService) processMessage(messageBody string) {
	// Apply artificial processing delay if configured
	applyProcessingDelay()
	
	// Try to unmarshal as execution first (from JMI)
	var execution map[string]interface{}
	if err := json.Unmarshal([]byte(messageBody), &execution); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// Check if it's an execution (has executionName) or a job (has id)
	if executionName, hasExecutionName := execution["executionName"]; hasExecutionName {
		log.Printf("Worker %s processing execution %v", j.workerID, executionName)
		
		// Create versioned execution record for JMW processing
		now := time.Now()
		versionedExecution := map[string]interface{}{
			"executionName": execution["executionName"],
			"executionUuid": execution["executionUuid"],
			"status":        "processed",
			"createdAt":     execution["createdAt"], // Keep original creation time
			"updatedAt":     now.Format(time.RFC3339),
			"version":       2, // Increment version for JMW stage
			"stage":         "jmw-process",
			"processedBy":   "JMW",
			"workerID":      j.workerID,
			"timestamp":     now.Unix(),
		}

		// Create composite key for versioning
		executionKey := fmt.Sprintf("%s#v%d#%s", versionedExecution["executionName"], versionedExecution["version"], versionedExecution["stage"])
		
		// Create versioned struct for DynamoDB
		type VersionedExecution struct {
			ExecutionName     string `dynamodbav:"executionName"`     // Chave primária: será o executionKey para uniqueness
			OriginalName      string `dynamodbav:"originalName"`      // Nome original da execução
			ExecutionUuid     string `dynamodbav:"executionUuid"`
			Status            string `dynamodbav:"status"`
			CreatedAt         string `dynamodbav:"createdAt"`
			UpdatedAt         string `dynamodbav:"updatedAt"`
			Version           int    `dynamodbav:"version"`
			Stage             string `dynamodbav:"stage"`
			ProcessedBy       string `dynamodbav:"processedBy"`
			WorkerID          string `dynamodbav:"workerID"`
			Timestamp         int64  `dynamodbav:"timestamp"`
		}

		versionedExec := VersionedExecution{
			ExecutionName: executionKey,                                 // Chave primária única
			OriginalName:  versionedExecution["executionName"].(string), // Nome original
			ExecutionUuid: versionedExecution["executionUuid"].(string),
			Status:        versionedExecution["status"].(string),
			CreatedAt:     versionedExecution["createdAt"].(string),
			UpdatedAt:     versionedExecution["updatedAt"].(string),
			Version:       versionedExecution["version"].(int),
			Stage:         versionedExecution["stage"].(string),
			ProcessedBy:   versionedExecution["processedBy"].(string),
			WorkerID:      versionedExecution["workerID"].(string),
			Timestamp:     versionedExecution["timestamp"].(int64),
		}

		// Store versioned execution in DynamoDB
		item, err := attributevalue.MarshalMap(versionedExec)
		if err != nil {
			log.Printf("Error marshaling versioned execution: %v", err)
			return
		}

		_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: aws.String(j.tableName),
			Item:      item,
		})

		if err != nil {
			log.Printf("Error updating execution in DynamoDB: %v", err)
			return
		}

		// Forward original execution data to JMR queue (not the versioned one)
		executionJSON, err := json.Marshal(execution)
		if err != nil {
			log.Printf("Error marshaling execution for JMR: %v", err)
			return
		}

		_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
			QueueUrl:    aws.String(j.outQueueURL),
			MessageBody: aws.String(string(executionJSON)),
		})

		if err != nil {
			log.Printf("Error sending message to JMR queue: %v", err)
			return
		}

		log.Printf("Worker %s completed execution %v and forwarded to JMR", j.workerID, executionName)
	} else {
		// Legacy job processing (for backward compatibility)
		var job Job
		if err := json.Unmarshal([]byte(messageBody), &job); err != nil {
			log.Printf("Error unmarshaling job message: %v", err)
			return
		}

		log.Printf("Worker %s processing legacy job %s", j.workerID, job.ID)
		
		// Simulate job processing work
		time.Sleep(1 * time.Second)

		// Update job status
		job.Status = "processed"
		job.WorkerID = j.workerID
		job.UpdatedAt = time.Now()

		// For legacy jobs, we need to store in jobs table, not executions table
		// But since JMW is configured for executions table, we'll skip DynamoDB storage for legacy jobs
		// and just forward to JMR
		log.Printf("Legacy job %s processed, forwarding to JMR without DynamoDB storage", job.ID)

		// Add to local cache
		j.jobs = append(j.jobs, job)

		// Forward to JMR queue
		jobJSON, err := json.Marshal(job)
		if err != nil {
			log.Printf("Error marshaling job for JMR: %v", err)
			return
		}

		_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
			QueueUrl:    aws.String(j.outQueueURL),
			MessageBody: aws.String(string(jobJSON)),
		})

		if err != nil {
			log.Printf("Error sending message to JMR queue: %v", err)
			return
		}

		log.Printf("Worker %s completed legacy job %s and forwarded to JMR", j.workerID, job.ID)
	}
}

func (j *JMWService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":   "jmw",
		"status":    "healthy",
		"worker_id": j.workerID,
		"timestamp": time.Now(),
	})
}

func (j *JMWService) GetStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"worker_id":      j.workerID,
		"jobs_processed": len(j.jobs),
		"timestamp":      time.Now(),
	})
}

func (j *JMWService) ProcessJob(ctx *gin.Context) {
	var job Job
	if err := ctx.ShouldBindJSON(&job); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate job processing work
	log.Printf("Worker %s processing job %s", j.workerID, job.ID)
	time.Sleep(1 * time.Second) // Simulate work

	// Update job status
	job.Status = "processed"
	job.WorkerID = j.workerID
	job.UpdatedAt = time.Now()

	// Update job in DynamoDB
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
		log.Printf("Error updating job in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store job"})
		return
	}

	// Add to local cache
	j.jobs = append(j.jobs, job)

	// Forward to JMR queue
	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job for JMR: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process job"})
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to JMR queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward job"})
		return
	}

	log.Printf("Worker %s completed job %s", j.workerID, job.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":   "Job processed successfully",
		"job_id":    job.ID,
		"status":    job.Status,
		"worker_id": j.workerID,
	})
}

func (j *JMWService) Start(ctx *gin.Context) {
	var req StartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate execution UUID
	executionUuid := uuid.New().String()

	// Create execution record for processing
	execution := map[string]interface{}{
		"executionName":    req.ExecutionName,
		"executionUuid":    executionUuid,
		"accountId":        req.AccountId,
		"commonProperties": req.CommonProperties,
		"runtimes":         req.Runtimes,
		"schedulerRoutine": req.SchedulerRoutine,
		"status":           "processing",
		"createdAt":        time.Now(),
		"updatedAt":        time.Now(),
	}

	// Store execution in DynamoDB
	item, err := attributevalue.MarshalMap(execution)
	if err != nil {
		log.Printf("Error marshaling execution: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution"})
		return
	}

	_, err = j.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(j.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing execution in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store execution"})
		return
	}

	// Forward to JMR queue
	executionJSON, err := json.Marshal(execution)
	if err != nil {
		log.Printf("Error marshaling execution for JMR: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process execution"})
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(executionJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to JMR queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward execution"})
		return
	}

	log.Printf("JMW started processing execution %s with UUID %s", req.ExecutionName, executionUuid)

	ctx.JSON(http.StatusOK, executionUuid)
}

func main() {
	service := NewJMWService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// Worker stats
	r.GET("/stats", service.GetStats)

	// Start execution endpoint (new)
	r.POST("/start", service.Start)

	// Job processing (legacy)
	r.POST("/process", service.ProcessJob)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("JMW service starting on port %s with worker ID %s", port, service.workerID)
	log.Fatal(r.Run(":" + port))
}