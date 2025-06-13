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

// TriggerRequest represents the trigger payload from collection.json
type TriggerRequest struct {
	AccountId     string                 `json:"accountId"`
	ExecutionName string                 `json:"executionName"`
	EventDate     string                 `json:"eventDate"`
	EventType     string                 `json:"eventType"`
	EventId       string                 `json:"eventId"`
	Parameters    map[string]interface{} `json:"parameters"`
}

// ScheduleRequest represents the schedule creation payload from collection.json
type ScheduleRequest struct {
	Acronym  string    `json:"acronym"`
	Repo     string    `json:"repo"`
	Routines []Routine `json:"routines"`
}

type Routine struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Cron        string   `json:"cron"`
	Priority    string   `json:"priority"`
	DependsOn   []string `json:"dependsOn"`
}

// Legacy Adapter struct for backward compatibility
type Adapter struct {
	ID          string                 `json:"id" dynamodbav:"id"`
	ScheduleID  string                 `json:"schedule_id" dynamodbav:"schedule_id"`
	AdapterType string                 `json:"adapter_type" dynamodbav:"adapter_type"`
	Config      map[string]interface{} `json:"config" dynamodbav:"config"`
	Status      string                 `json:"status" dynamodbav:"status"`
	CreatedAt   time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" dynamodbav:"updated_at"`
}

type SPAService struct {
	adapters      []Adapter
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	inQueueURL    string
	outQueueURL   string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewSPAService() *SPAService {
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

	service := &SPAService{
		adapters:      make([]Adapter, 0),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		inQueueURL:    os.Getenv("SPA_QUEUE_URL"),
		outQueueURL:   os.Getenv("SPAQ_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (s *SPAService) startMessageReceiver() {
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

func (s *SPAService) processMessage(messageBody string) {
	var schedule map[string]interface{}
	if err := json.Unmarshal([]byte(messageBody), &schedule); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	scheduleID, _ := schedule["id"].(string)
	if scheduleID == "" {
		log.Printf("Invalid schedule ID in message")
		return
	}

	cronExpr, _ := schedule["cron_expr"].(string)
	if cronExpr == "" {
		cronExpr = "0 */5 * * * *"
	}

	// Create adapter configuration
	adapter := Adapter{
		ID:          uuid.New().String(),
		ScheduleID:  scheduleID,
		AdapterType: s.determineAdapterType(cronExpr),
		Config:      s.createAdapterConfig(cronExpr),
		Status:      "configured",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store adapter in DynamoDB
	item, err := attributevalue.MarshalMap(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter: %v", err)
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing adapter in DynamoDB: %v", err)
		return
	}

	// Add to local cache
	s.adapters = append(s.adapters, adapter)

	// Forward to SPAQ queue
	adapterJSON, err := json.Marshal(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter for SPAQ: %v", err)
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(adapterJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPAQ queue: %v", err)
		return
	}

	log.Printf("SPA created adapter %s for schedule %s", adapter.ID, scheduleID)
}

func (s *SPAService) determineAdapterType(cronExpr string) string {
	// Determine adapter type based on cron expression
	switch cronExpr {
	case "0 */5 * * * *":
		return "frequent"
	case "0 0 * * * *":
		return "hourly"
	case "0 0 0 * * *":
		return "daily"
	default:
		return "custom"
	}
}

func (s *SPAService) createAdapterConfig(cronExpr string) map[string]interface{} {
	return map[string]interface{}{
		"cron_expression": cronExpr,
		"retry_count":     3,
		"timeout":         "30s",
		"priority":        "normal",
	}
}

func (s *SPAService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":   "spa",
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (s *SPAService) GetAdapters(ctx *gin.Context) {
	// Query DynamoDB for all adapters
	result, err := s.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve adapters"})
		return
	}

	var adapters []Adapter
	err = attributevalue.UnmarshalListOfMaps(result.Items, &adapters)
	if err != nil {
		log.Printf("Error unmarshaling adapters: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapters data"})
		return
	}

	ctx.JSON(http.StatusOK, adapters)
}

func (s *SPAService) CreateAdapter(ctx *gin.Context) {
	var adapter Adapter
	if err := ctx.ShouldBindJSON(&adapter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if adapter.ID == "" {
		adapter.ID = uuid.New().String()
	}
	adapter.CreatedAt = time.Now()
	adapter.UpdatedAt = time.Now()
	adapter.Status = "configured"

	// Store adapter in DynamoDB
	item, err := attributevalue.MarshalMap(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapter"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing adapter in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store adapter"})
		return
	}

	// Add to local cache
	s.adapters = append(s.adapters, adapter)

	// Forward to SPAQ queue
	adapterJSON, err := json.Marshal(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter for SPAQ: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapter"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(adapterJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPAQ queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward adapter"})
		return
	}

	ctx.JSON(http.StatusCreated, adapter)
}

func (s *SPAService) ProcessSchedule(ctx *gin.Context) {
	var schedule map[string]interface{}
	if err := ctx.ShouldBindJSON(&schedule); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scheduleID, _ := schedule["id"].(string)
	cronExpr, _ := schedule["cron_expr"].(string)
	if cronExpr == "" {
		cronExpr = "0 */5 * * * *"
	}

	// Create adapter configuration
	adapter := Adapter{
		ID:          uuid.New().String(),
		ScheduleID:  scheduleID,
		AdapterType: s.determineAdapterType(cronExpr),
		Config:      s.createAdapterConfig(cronExpr),
		Status:      "configured",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store adapter in DynamoDB
	item, err := attributevalue.MarshalMap(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapter"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing adapter in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store adapter"})
		return
	}

	// Add to local cache
	s.adapters = append(s.adapters, adapter)

	// Forward to SPAQ queue
	adapterJSON, err := json.Marshal(adapter)
	if err != nil {
		log.Printf("Error marshaling adapter for SPAQ: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapter"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(adapterJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPAQ queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward adapter"})
		return
	}

	log.Printf("SPA created adapter %s for schedule %s", adapter.ID, scheduleID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":    "Schedule processed successfully",
		"schedule_id": scheduleID,
		"adapter_id": adapter.ID,
	})
}

func (s *SPAService) Trigger(ctx *gin.Context) {
	var req TriggerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create trigger record
	trigger := map[string]interface{}{
		"id":            uuid.New().String(),
		"accountId":     req.AccountId,
		"executionName": req.ExecutionName,
		"eventDate":     req.EventDate,
		"eventType":     req.EventType,
		"eventId":       req.EventId,
		"parameters":    req.Parameters,
		"status":        "triggered",
		"createdAt":     time.Now(),
		"updatedAt":     time.Now(),
	}

	// Store trigger in DynamoDB
	item, err := attributevalue.MarshalMap(trigger)
	if err != nil {
		log.Printf("Error marshaling trigger: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process trigger"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing trigger in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store trigger"})
		return
	}

	// Forward to SPAQ queue
	triggerJSON, err := json.Marshal(trigger)
	if err != nil {
		log.Printf("Error marshaling trigger for SPAQ: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process trigger"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(triggerJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPAQ queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward trigger"})
		return
	}

	log.Printf("SPA processed trigger for execution %s", req.ExecutionName)

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Trigger processed successfully",
		"executionName": req.ExecutionName,
		"eventId":       req.EventId,
		"status":        "triggered",
	})
}

func (s *SPAService) Schedule(ctx *gin.Context) {
	var req ScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create schedule record
	schedule := map[string]interface{}{
		"id":        uuid.New().String(),
		"acronym":   req.Acronym,
		"repo":      req.Repo,
		"routines":  req.Routines,
		"status":    "created",
		"createdAt": time.Now(),
		"updatedAt": time.Now(),
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

	// Forward to SPAQ queue
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		log.Printf("Error marshaling schedule for SPAQ: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process schedule"})
		return
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.outQueueURL),
		MessageBody: aws.String(string(scheduleJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SPAQ queue: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward schedule"})
		return
	}

	log.Printf("SPA created schedule for repo %s with %d routines", req.Repo, len(req.Routines))

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "Schedule created successfully",
		"acronym":  req.Acronym,
		"repo":     req.Repo,
		"routines": len(req.Routines),
		"status":   "created",
	})
}

func main() {
	service := NewSPAService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// New endpoints from collection.json
	r.POST("/v1/trigger", service.Trigger)
	r.POST("/v1/schedule", service.Schedule)

	// Legacy adapter endpoints
	r.GET("/adapters", service.GetAdapters)
	r.POST("/adapters", service.CreateAdapter)
	r.POST("/process", service.ProcessSchedule)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("SPA service starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}