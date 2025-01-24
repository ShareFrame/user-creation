package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ShareFrame/user-management/internal/handlers"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logrus.Info("Processing GraphQL request")

	var gqlRequest GraphQLRequest
	err := json.Unmarshal([]byte(request.Body), &gqlRequest)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse GraphQL request body")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid GraphQL request payload",
		}, nil
	}

	logrus.WithField("query", gqlRequest.Query).Info("Parsed GraphQL request")

	switch gqlRequest.OperationName {
	case "CreateUser":
		return handleCreateUser(ctx, gqlRequest)
	default:
		logrus.Warn("Unsupported operation")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotImplemented,
			Body:       fmt.Sprintf("Unsupported operation: %s", gqlRequest.OperationName),
		}, nil
	}
}

func handleCreateUser(ctx context.Context, gqlRequest GraphQLRequest) (events.APIGatewayProxyResponse, error) {
	logrus.Info("Handling CreateUser mutation")

	// Extract and validate input variables
	inputData, ok := gqlRequest.Variables["input"].(map[string]interface{})
	if !ok {
		logrus.Error("Invalid input format: expected a map")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid input format",
		}, nil
	}

	// Convert map[string]interface{} to JSON, then unmarshal into the struct
	inputBytes, err := json.Marshal(inputData)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal input data")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Failed to process input data",
		}, nil
	}

	var input models.UserRequest
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal input to UserRequest")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid input data",
		}, nil
	}

	logrus.WithFields(logrus.Fields{
		"handle": input.Handle,
		"email":  input.Email,
	}).Info("Input validated successfully")

	return handlers.UserHandler(ctx, input)
}
