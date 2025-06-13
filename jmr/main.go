package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
)

type Job struct {
	ID           string                 `json:"id" dynamodbav:"id"`
	JobName      string                 `json:"job_name" dynamodbav:"job_name"`
	JobType      string                 `json:"job_type" dynamodbav:"job_type"`
	Parameters   map[string]interface{} `json:"parameters" dynamodbav:"parameters"`
	Priority     int                    `json:"priority" dynamodbav:"priority"`
	ScheduledAt  time.Time              `json:"scheduled_at" dynamodbav:"scheduled_at"`
	CreatedAt    time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	Status       string                 `json:"status" dynamodbav:"status"`
	WorkerID     string                 `json:"worker_id" dynamodbav:"worker_id"`
	RunnerID     string                 `json:"runner_id" dynamodbav:"runner_id"`
	ExecutionLog string                 `json:"execution_log" dynamodbav:"execution_log"`
}

type JMRService struct {
	jobs          []Job
	runnerID      string
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	inQueueURL    string
	outQueueURL   string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewJMRService() *JMRService {
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

	service := &JMRService{
		jobs:          make([]Job, 0),
		runnerID:      "jmr-" + time.Now().Format("20060102150405"),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		inQueueURL:    os.Getenv("JMR_QUEUE_URL"),
		outQueueURL:   os.Getenv("SP_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (j *JMRService) startMessageReceiver() {
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

func (j *JMRService) processMessage(messageBody string) {
	var job Job
	if err := json.Unmarshal([]byte(messageBody), &job); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	log.Printf("Runner %s executing job %s", j.runnerID, job.ID)

	// Execute job
	executionResult := j.executeJob(job)

	// Update job status
	job.Status = "executed"
	job.RunnerID = j.runnerID
	job.UpdatedAt = time.Now()
	job.ExecutionLog = executionResult

	// Update job in DynamoDB
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
		log.Printf("Error updating job in DynamoDB: %v", err)
		return
	}

	// Add to local cache
	j.jobs = append(j.jobs, job)

	// Forward to Scheduler Plugin queue
	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job for SP: %v", err)
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SP queue: %v", err)
		return
	}

	log.Printf("Runner %s completed execution of job %s and forwarded to Scheduler Plugin", j.runnerID, job.ID)
}

func (j *JMRService) executeJob(job Job) string {
	// Simulate different job types
	switch job.JobType {
	case "shell":
		return j.executeShellJob(job)
	case "python":
		return j.executePythonJob(job)
	case "sql":
		return j.executeSQLJob(job)
	default:
		return j.executeDefaultJob(job)
	}
}

func (j *JMRService) executeShellJob(job Job) string {
	// Simulate shell command execution
	cmd := exec.Command("echo", "Executing shell job: "+job.JobName)
	output, err := cmd.Output()
	if err != nil {
		return "Error executing shell job: " + err.Error()
	}
	return "Shell execution result: " + string(output)
}

func (j *JMRService) executePythonJob(job Job) string {
	// Simulate Python script execution
	time.Sleep(500 * time.Millisecond)
	return "Python script executed successfully for job: " + job.JobName
}

func (j *JMRService) executeSQLJob(job Job) string {
	// Simulate SQL query execution
	time.Sleep(300 * time.Millisecond)
	return "SQL query executed successfully for job: " + job.JobName
}

func (j *JMRService) executeDefaultJob(job Job) string {
	// Default job execution
	time.Sleep(500 * time.Millisecond)
	return "Default job executed successfully: " + job.JobName
}

func (j *JMRService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":   "jmr",
		"status":    "healthy",
		"runner_id": j.runnerID,
		"timestamp": time.Now(),
	})
}

func (j *JMRService) GetStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"runner_id":     j.runnerID,
		"jobs_executed": len(j.jobs),
		"timestamp":     time.Now(),
	})
}

func (j *JMRService) ExecuteJob(ctx *gin.Context) {
	var job Job
	if err := ctx.ShouldBindJSON(&job); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Runner %s executing job %s", j.runnerID, job.ID)

	// Execute job
	executionResult := j.executeJob(job)

	// Update job status
	job.Status = "executed"
	job.RunnerID = j.runnerID
	job.UpdatedAt = time.Now()
	job.ExecutionLog = executionResult

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

	// Forward to Scheduler Plugin queue
	jobJSON, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job for SP: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process job"})
		return
	}

	_, err = j.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(j.outQueueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SP queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward job"})
		return
	}

	log.Printf("Runner %s completed execution of job %s", j.runnerID, job.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Job executed successfully",
		"job_id":        job.ID,
		"status":        job.Status,
		"runner_id":     j.runnerID,
		"execution_log": executionResult,
	})
}

func main() {
	service := NewJMRService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// Runner stats
	r.GET("/stats", service.GetStats)

	// Job execution
	r.POST("/execute", service.ExecuteJob)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("JMR service starting on port %s with runner ID %s", port, service.runnerID)
	log.Fatal(r.Run(":" + port))
}