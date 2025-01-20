package main

import (
	"github.com/Atlas-Mesh/user-management/internal/handlers"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func main() {
	log.Println("Starting Lambda...")
	lambda.Start(handlers.UserHandler)
}
