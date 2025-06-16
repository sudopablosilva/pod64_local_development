package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// StartExecutionRequest represents the request to start an execution
type StartExecutionRequest struct {
	ExecutionName string `json:"executionName"`
}

// StartExecutionResponse represents the response from JMI
type StartExecutionResponse struct {
	ExecutionName string `json:"executionName"`
	ExecutionUuid string `json:"executionUuid"`
	Message       string `json:"message"`
	Status        string `json:"status"`
}

type ControlMService struct {
	jobs      []JobRequest
	sqsClient *sqs.Client
	queueURL  string
	jmiURL    string
}

func NewControlMService() *ControlMService {
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

	// Determine JMI URL based on environment
	jmiURL := os.Getenv("JMI_URL")
	if jmiURL == "" {
		// Default to Docker internal network address
		jmiURL = "http://jmi:8080"
	}

	return &ControlMService{
		jobs:      make([]JobRequest, 0),
		sqsClient: sqs.NewFromConfig(cfg),
		queueURL:  os.Getenv("SQS_QUEUE_URL"),
		jmiURL:    jmiURL,
	}
}

func (c *ControlMService) StartExecution(ctx *gin.Context) {
	var req StartExecutionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Control-M: Starting execution %s", req.ExecutionName)

	// Call JMI to start the execution
	jmiResponse, err := c.callJMI(req)
	if err != nil {
		log.Printf("Error calling JMI: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start execution via JMI",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Control-M: Successfully started execution %s via JMI", req.ExecutionName)

	// Return the response from JMI
	ctx.JSON(http.StatusOK, jmiResponse)
}

func (c *ControlMService) callJMI(req StartExecutionRequest) (*StartExecutionResponse, error) {
	// Prepare the request to JMI
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make HTTP request to JMI
	jmiEndpoint := fmt.Sprintf("%s/startExecution", c.jmiURL)
	log.Printf("Control-M: Calling JMI at %s", jmiEndpoint)

	resp, err := http.Post(jmiEndpoint, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call JMI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JMI returned status %d", resp.StatusCode)
	}

	// Parse JMI response
	var jmiResponse StartExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&jmiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JMI response: %v", err)
	}

	return &jmiResponse, nil
}

func (c *ControlMService) SubmitJob(ctx *gin.Context) {
	var req JobRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	req.CreatedAt = time.Now()
	req.Status = "submitted"

	// Store job locally
	c.jobs = append(c.jobs, req)

	// Send job to SQS queue
	jobJSON, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshaling job: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process job"})
		return
	}

	_, err = c.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(string(jobJSON)),
	})

	if err != nil {
		log.Printf("Error sending message to SQS: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit job to queue"})
		return
	}

	log.Printf("Job submitted to SQS: %s", req.ID)

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Job submitted successfully",
		"job_id":  req.ID,
		"status":  req.Status,
	})
}

func (c *ControlMService) GetJobs(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.jobs)
}

func (c *ControlMService) GetHealth(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"service":    "control-m",
		"status":     "healthy",
		"timestamp":  time.Now(),
		"jobs_count": len(c.jobs),
	})
}

func main() {
	service := NewControlMService()

	r := gin.Default()

	// Health check
	r.GET("/health", service.GetHealth)

	// Job management endpoints
	r.POST("/jobs", service.SubmitJob)
	r.GET("/jobs", service.GetJobs)

	// Execution management endpoints (NEW - calls JMI)
	r.POST("/startExecution", service.StartExecution)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Control-M service starting on port %s", port)
	log.Printf("Control-M will call JMI at: %s", service.jmiURL)
	log.Fatal(r.Run(":" + port))
}
