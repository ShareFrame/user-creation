package main

import (
	"log"

	"github.com/ShareFrame/user-management/internal/handlers"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	log.Println("Starting Lambda...")
	lambda.Start(handlers.UserHandler)
}
