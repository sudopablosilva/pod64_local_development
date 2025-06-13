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

type QueueMessage struct {
	ID          string                 `json:"id" dynamodbav:"id"`
	AdapterID   string                 `json:"adapter_id" dynamodbav:"adapter_id"`
	MessageType string                 `json:"message_type" dynamodbav:"message_type"`
	Payload     map[string]interface{} `json:"payload" dynamodbav:"payload"`
	Status      string                 `json:"status" dynamodbav:"status"`
	Priority    int                    `json:"priority" dynamodbav:"priority"`
	RetryCount  int                    `json:"retry_count" dynamodbav:"retry_count"`
	CreatedAt   time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty" dynamodbav:"processed_at,omitempty"`
}

type SPAQService struct {
	messages      []QueueMessage
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	tableName     string
	inQueueURL    string
	receiveCtx    context.Context
	receiveCancel context.CancelFunc
}

func NewSPAQService() *SPAQService {
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

	service := &SPAQService{
		messages:      make([]QueueMessage, 0),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		tableName:     os.Getenv("DYNAMODB_TABLE"),
		inQueueURL:    os.Getenv("SPAQ_QUEUE_URL"),
		receiveCtx:    ctx,
		receiveCancel: cancel,
	}

	// Start message receiver
	go service.startMessageReceiver()

	return service
}

func (s *SPAQService) startMessageReceiver() {
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

func (s *SPAQService) processMessage(messageBody string) {
	var adapter map[string]interface{}
	if err := json.Unmarshal([]byte(messageBody), &adapter); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	adapterID, _ := adapter["id"].(string)
	if adapterID == "" {
		log.Printf("Invalid adapter ID in message")
		return
	}

	adapterType, _ := adapter["adapter_type"].(string)
	scheduleID, _ := adapter["schedule_id"].(string)

	log.Printf("SPAQ processing adapter %s", adapterID)

	// Create queue message entry
	queueMessage := QueueMessage{
		ID:          uuid.New().String(),
		AdapterID:   adapterID,
		MessageType: "adapter_configuration",
		Payload: map[string]interface{}{
			"adapter_type": adapterType,
			"schedule_id":  scheduleID,
		},
		Status:     "queued",
		Priority:   s.calculatePriority(adapterType),
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Store message in DynamoDB
	item, err := attributevalue.MarshalMap(queueMessage)
	if err != nil {
		log.Printf("Error marshaling queue message: %v", err)
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing queue message in DynamoDB: %v", err)
		return
	}

	// Add to local cache
	s.messages = append(s.messages, queueMessage)

	// Process the message immediately (simulate queue processing)
	go s.processQueueMessage(queueMessage)

	log.Printf("SPAQ processed adapter %s and created queue message %s", adapterID, queueMessage.ID)
}

func (s *SPAQService) calculatePriority(adapterType string) int {
	// Calculate priority based on adapter type
	switch adapterType {
	case "frequent":
		return 1 // High priority
	case "hourly":
		return 2 // Medium priority
	case "daily":
		return 3 // Low priority
	default:
		return 2 // Default medium priority
	}
}

func (s *SPAQService) processQueueMessage(queueMessage QueueMessage) {
	// Simulate message processing
	time.Sleep(500 * time.Millisecond)

	// Update message status
	queueMessage.Status = "processed"
	now := time.Now()
	queueMessage.ProcessedAt = &now
	queueMessage.UpdatedAt = now

	// Update in DynamoDB
	item, err := attributevalue.MarshalMap(queueMessage)
	if err != nil {
		log.Printf("Error marshaling updated queue message: %v", err)
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error updating queue message in DynamoDB: %v", err)
		return
	}

	// Update in memory
	for i, msg := range s.messages {
		if msg.ID == queueMessage.ID {
			s.messages[i] = queueMessage
			break
		}
	}

	log.Printf("SPAQ completed processing queue message %s", queueMessage.ID)
}

func (s *SPAQService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":   "spaq",
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (s *SPAQService) GetMessages(ctx *gin.Context) {
	// Query DynamoDB for all messages
	result, err := s.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	var messages []QueueMessage
	err = attributevalue.UnmarshalListOfMaps(result.Items, &messages)
	if err != nil {
		log.Printf("Error unmarshaling messages: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process messages data"})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

func (s *SPAQService) GetStats(ctx *gin.Context) {
	// Query DynamoDB for all messages
	result, err := s.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		log.Printf("Error scanning DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	var messages []QueueMessage
	err = attributevalue.UnmarshalListOfMaps(result.Items, &messages)
	if err != nil {
		log.Printf("Error unmarshaling messages: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process messages data"})
		return
	}

	stats := map[string]int{
		"total":     len(messages),
		"queued":    0,
		"processed": 0,
		"failed":    0,
	}

	for _, msg := range messages {
		stats[msg.Status]++
	}

	ctx.JSON(http.StatusOK, gin.H{
		"queue_stats": stats,
		"timestamp":   time.Now(),
	})
}

func (s *SPAQService) ProcessAdapter(ctx *gin.Context) {
	var adapter map[string]interface{}
	if err := ctx.ShouldBindJSON(&adapter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adapterID, _ := adapter["id"].(string)
	adapterType, _ := adapter["adapter_type"].(string)
	scheduleID, _ := adapter["schedule_id"].(string)

	log.Printf("SPAQ processing adapter %s", adapterID)

	// Create queue message entry
	queueMessage := QueueMessage{
		ID:          uuid.New().String(),
		AdapterID:   adapterID,
		MessageType: "adapter_configuration",
		Payload: map[string]interface{}{
			"adapter_type": adapterType,
			"schedule_id":  scheduleID,
		},
		Status:     "queued",
		Priority:   s.calculatePriority(adapterType),
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Store message in DynamoDB
	item, err := attributevalue.MarshalMap(queueMessage)
	if err != nil {
		log.Printf("Error marshaling queue message: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process adapter"})
		return
	}

	_, err = s.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("Error storing queue message in DynamoDB: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store message"})
		return
	}

	// Add to local cache
	s.messages = append(s.messages, queueMessage)

	// Process the message immediately (simulate queue processing)
	go s.processQueueMessage(queueMessage)

	log.Printf("SPAQ processed adapter %s and created queue message %s", adapterID, queueMessage.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"message":         "Adapter processed successfully",
		"adapter_id":      adapterID,
		"queue_message_id": queueMessage.ID,
	})
}

func main() {
	service := NewSPAQService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// Queue endpoints
	r.GET("/messages", service.GetMessages)
	r.GET("/stats", service.GetStats)
	r.POST("/process", service.ProcessAdapter)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("SPAQ service starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}