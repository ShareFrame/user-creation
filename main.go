package main

import (
	"context"
	"log"

	"github.com/ShareFrame/user-management/internal/handlers"
	shareframeDB "github.com/ShareFrame/user-management/internal/db"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func main() {
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)
	dynamoClient := dynamodb.NewFromConfig(awsCfg)

	dbClient := shareframeDB.NewDynamoDBClient(dynamoClient)

	userHandler := handlers.NewUserHandler(secretsManagerClient, dbClient)

	lambda.Start(userHandler.Handle)
}
