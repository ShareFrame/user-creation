package main

import (
	"context"

	"github.com/ShareFrame/user-management/internal/handlers"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func main() {
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("Failed to load AWS config: " + err.Error())
	}

	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)

	userHandler := handlers.NewUserHandler(secretsManagerClient)

	lambda.Start(userHandler.Handle)
}
