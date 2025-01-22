package handlers

import (
	"context"
	"encoding/json"
	ATProtocol "github.com/Atlas-Mesh/user-management/internal/atproto"
	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/dynamo"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func retrieveAdminCredentials() (AdminCreds, error) {
	input, err := config.RetrieveAdminCreds()
	if err != nil {
		log.Fatalf("Failed to retrieve admin creds: %v", err)
		return AdminCreds{}, err
	}

	var adminCredentials AdminCreds
	err = json.Unmarshal([]byte(input), &adminCredentials)
	if err != nil {
		log.Fatalf("Failed to unmarshal admin creds: %v", err)
	}

	return adminCredentials, nil
}

func UserHandler(ctx context.Context, event UserRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("Processing request...")

	cfg, awsCfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
		return events.APIGatewayProxyResponse{}, err
	}

	adminCreds, err := retrieveAdminCredentials()
	log.Printf("Admin credentials: %v", adminCreds)

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName)
	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL)

	err = atProtoClient.RegisterUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to register user: %v", err)
		return events.APIGatewayProxyResponse{}, err
	}

	err = dynamoClient.StoreUser(event.Handle, event.Email)
	if err != nil {
		log.Fatalf("Failed to store user: %v", err)
		return events.APIGatewayProxyResponse{}, err
	}

	log.Println("User processed successfully")
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "User registered successfully",
	}, nil
}
