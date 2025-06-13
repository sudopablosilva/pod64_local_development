package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
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

type Schedule struct {
	ID        string    `json:"id" dynamodbav:"id"`
	JobID     string    `json:"job_id" dynamodbav:"job_id"`
	CronExpr  string    `json:"cron_expr" dynamodbav:"cron_expr"`
	NextRun   time.Time `json:"next_run" dynamodbav:"next_run"`
	LastRun   time.Time `json:"last_run" dynamodbav:"last_run"`
	IsActive  bool      `json:"is_active" dynamodbav:"is_active"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

type SchedulerPluginService struct {
	schedules     []Schedule
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	inQueueURL    string
	outQueueURL   string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewSchedulerPluginService() *SchedulerPluginService {
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

	service := &SchedulerPluginService{
		schedules:     make([]Schedule, 0),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		inQueueURL:    os.Getenv("SP_QUEUE_URL"),
		outQueueURL:   os.Getenv("SPA_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (s *SchedulerPluginService) startMessageReceiver() {
	for {
		select {
		case <-s.receiveCtx.Done():
			log.Println("Message receiver stopped")
			return
		default:
			// Receive messages from SQS
			result, err := s.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(s.inQueueURL),
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
				s.processMessage(*message.Body)

				// Delete the message from the queue
				_, err := s.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(s.inQueueURL),
					ReceiptHandle: message.ReceiptHandle,
				})

				if err != nil {
					log.Printf("Error deleting message: %v", err)
				}
			}
		}
	}
}

func (s *SchedulerPluginService) processMessage(messageBody string) {
	var job map[string]interface{}
	if err := json.Unmarshal([]byte(messageBody), &job); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	jobID, _ := job["id"].(string)
	if jobID == "" {
		log.Printf("Invalid job ID in message")
		return
	}

	// Create schedule entry
	schedule := Schedule{
		ID:        uuid.New().String(),
		JobID:     jobID,
		CronExpr:  "0 */5 * * * *", // Every 5 minutes (default)
		NextRun:   time.Now().Add(5 * time.Minute),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store schedule in DynamoDB
	item, err := attributevalue.MarshalMap(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule: %v", err)
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing schedule in DynamoDB: %v", err)
		return
	}

	// Add to local cache
	s.schedules = append(s.schedules, schedule)

	// Forward to SPA queue
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule for SPA: %v", err)
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(scheduleJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPA queue: %v", err)
		return
	}

	log.Printf("Scheduler Plugin processed job %s and created schedule %s", jobID, schedule.ID)
}

func (s *SchedulerPluginService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":   "scheduler-plugin",
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (s *SchedulerPluginService) GetSchedules(ctx *gin.Context) {
	// Query DynamoDB for all schedules
	result, err := s.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules"})
		return
	}

	var schedules []Schedule
	err = attributevalue.UnmarshalListOfMaps(result.Items, &schedules)
	if err != nil {
		log.Printf("Error unmarshaling schedules: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedules data"})
		return
	}

	ctx.JSON(http.StatusOK, schedules)
}

func (s *SchedulerPluginService) CreateSchedule(ctx *gin.Context) {
	var schedule Schedule
	if err := ctx.ShouldBindJSON(&schedule); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()
	schedule.IsActive = true

	// Set default cron expression if not provided
	if schedule.CronExpr == "" {
		schedule.CronExpr = "0 */5 * * * *" // Every 5 minutes
	}

	// Set next run time
	schedule.NextRun = time.Now().Add(5 * time.Minute)

	// Store schedule in DynamoDB
	item, err := attributevalue.MarshalMap(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedule"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing schedule in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store schedule"})
		return
	}

	// Add to local cache
	s.schedules = append(s.schedules, schedule)

	// Forward to SPA queue
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule for SPA: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedule"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(scheduleJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPA queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward schedule"})
		return
	}

	log.Printf("Scheduler Plugin created schedule %s for job %s", schedule.ID, schedule.JobID)

	ctx.JSON(http.StatusCreated, schedule)
}

func (s *SchedulerPluginService) ProcessJob(ctx *gin.Context) {
	var job map[string]interface{}
	if err := ctx.ShouldBindJSON(&job); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID, _ := job["id"].(string)
	
	// Create schedule entry
	schedule := Schedule{
		ID:        uuid.New().String(),
		JobID:     jobID,
		CronExpr:  "0 */5 * * * *", // Every 5 minutes (default)
		NextRun:   time.Now().Add(5 * time.Minute),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store schedule in DynamoDB
	item, err := attributevalue.MarshalMap(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedule"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing schedule in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store schedule"})
		return
	}

	// Add to local cache
	s.schedules = append(s.schedules, schedule)

	// Forward to SPA queue
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule for SPA: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedule"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(scheduleJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPA queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward schedule"})
		return
	}

	log.Printf("Scheduler Plugin processed job %s and created schedule %s", jobID, schedule.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "Job scheduled successfully",
		"job_id":      jobID,
		"schedule_id": schedule.ID,
	})
}

func main() {
	service := NewSchedulerPluginService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// Schedule endpoints
	r.GET("/schedules", service.GetSchedules)
	r.POST("/schedules", service.CreateSchedule)
	r.POST("/process", service.ProcessJob)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Scheduler Plugin service starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}