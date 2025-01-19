package handlers

import (
	"context"
	"github.com/Atlas-Mesh/user-management/config"
	"github.com/Atlas-Mesh/user-management/internal/atproto"
	"github.com/Atlas-Mesh/user-management/internal/db"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"log"
)

type UserRequest struct {
	Handle string `json:"handle"`
	Email  string `json:"email"`
}

func UserHandler(ctx context.Context, event UserRequest) (string, error) {
	log.Println("Processing request...")

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		return "Internal Server Error", err
	}

	// Initialize Clients
	dynamoClient := db.NewDynamoClient(dynamodb.NewFromConfig(cfg.AwsConfig), cfg.DynamoTableName)
	atProtoClient := atproto.NewAtProtoClient(cfg.AtProtoBaseURL)

	// Register user via AT Protocol
	err = atProtoClient.RegisterUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to register user: %v", err)
		return "Failed to register user", err
	}

	// Store user in DynamoDB
	err = dynamoClient.StoreUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to store user: %v", err)
		return "Failed to store user", err
	}

	log.Println("User processed successfully")
	return "User registered and stored", nil
}
