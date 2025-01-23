package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/ShareFrame/user-management/config"
	ATProtocol "github.com/ShareFrame/user-management/internal/atproto"
	"github.com/ShareFrame/user-management/internal/dynamo"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const defaultTimeZone = "America/Chicago"

func retrieveAdminCredentials(ctx context.Context,
	secretsManagerClient config.SecretsManagerAPI) (models.AdminCreds, error) {
	input, err := config.RetrieveSecret(ctx, os.Getenv("PDS_ADMIN_SECRET_NAME"), secretsManagerClient)
	if err != nil {
		return models.AdminCreds{}, fmt.Errorf("failed to retrieve admin credentials: %w", err)
	}

	var adminCredentials models.AdminCreds
	err = json.Unmarshal([]byte(input), &adminCredentials)
	if err != nil {
		return models.AdminCreds{}, fmt.Errorf("failed to unmarshal admin credentials: %w", err)
	}

	return adminCredentials, nil
}

func validateAndFormatUser(event models.UserRequest) (models.UserRequest, *events.APIGatewayProxyResponse, error) {
	if event.Handle == "" || event.Email == "" {
		log.Printf("Invalid request: missing handle or email. Handle: %q, Email: %q", event.Handle, event.Email)
		return models.UserRequest{}, &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request: handle and email are required",
		}, nil
	}

	if !strings.HasSuffix(event.Handle, ".shareframe.social") {
		event.Handle = fmt.Sprintf("%s.shareframe.social", event.Handle)
		log.Printf("Updated handle: %s", event.Handle)
	}

	_, err := mail.ParseAddress(event.Email)
	if err != nil {
		log.Printf("Invalid email format: %s", event.Email)
		return models.UserRequest{}, &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request: email address is not properly formatted",
		}, nil
	}

	return event, nil, nil
}

func UserHandler(ctx context.Context, event models.UserRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Processing create account request for user: %s", event.Handle)

	// Validate request input
	updatedEvent, validationResponse, err := validateAndFormatUser(event)
	if err != nil {
		log.Printf("Validation error: %v", err)
		return *validationResponse, nil
	}
	event = updatedEvent

	// Load configuration
	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize Secrets Manager client
	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)

	// Retrieve admin credentials
	adminCreds, err := retrieveAdminCredentials(ctx, secretsManagerClient)
	if err != nil {
		log.Printf("Failed to retrieve admin credentials: %v", err)
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to retrieve admin credentials: %w", err)
	}

	// Initialize DynamoDB client
	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName, defaultTimeZone)

	// Initialize ATProtocol client
	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL, &http.Client{})

	// Create invite code
	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		log.Printf("Failed to create invite code: %v", err)
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to create invite code: %w", err)
	}

	// Register user
	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to register user: %w", err)
	}
	log.Printf("User registered successfully: %s", user.Handle)

	// Store user in DynamoDB
	err = dynamoClient.StoreUser(user)
	if err != nil {
		log.Printf("Failed to store user in DynamoDB: %v", err)
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to store user: %w", err)
	}

	log.Printf("Account created successfully: %s", user.DID)
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       "User registered successfully",
	}, nil
}
