package handlers

import (
	"context"
	"github.com/Atlas-Mesh/user-management/config"
	"github.com/Atlas-Mesh/user-management/internal/atproto"
	"github.com/Atlas-Mesh/user-management/internal/dynamo"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type UserRequest struct {
	Handle string `json:"handle"` // Represents the AT Protocol handle
	Email  string `json:"email"`  // User's email
}

func UserHandler(ctx context.Context, event UserRequest) (string, error) {
	log.Println("Processing request...")

	// Load Custom Config
	cfg, awsCfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
		return "Internal Server Error", err
	}

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)

	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName)

	atProtoClient := atproto.NewAtProtoClient(cfg.AtProtoBaseURL)

	err = atProtoClient.RegisterUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to register user: %v", err)
		return "Failed to register user", err
	}

	err = dynamoClient.StoreUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to store user: %v", err)
		return "Failed to store user", err
	}

	log.Println("User processed successfully")
	return "User registered and stored", nil
}
