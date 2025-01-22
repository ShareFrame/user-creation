package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Atlas-Mesh/user-management/config"
	ATProtocol "github.com/Atlas-Mesh/user-management/internal/atproto"
	"github.com/Atlas-Mesh/user-management/internal/dynamo"
	"github.com/Atlas-Mesh/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"log"
)

func retrieveAdminCredentials(ctx context.Context) (models.AdminCreds, error) {
	input, err := config.RetrieveAdminCreds(ctx)
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

func UserHandler(ctx context.Context, event models.UserRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Processing create account request for user: %s", event.Handle)

	if event.Handle == "" || event.Email == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request: handle and email are required",
		}, nil
	}

	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	adminCreds, err := retrieveAdminCredentials(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to retrieve admin credentials: %w", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient, err := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName, "America/Chicago")
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL)

	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to create invite code: %w", err)
	}

	event.Handle = fmt.Sprintf("%s.shareframe.social", event.Handle)
	log.Printf("Registering user with handle: %s", event.Handle)

	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to register user: %w", err)
	}
	log.Printf("User registered successfully: %s", user.Handle)

	err = dynamoClient.StoreUser(user)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to store user: %w", err)
	}

	log.Printf("Account created successfully: %s", user.DID)
	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       "User registered successfully",
	}, nil
}
