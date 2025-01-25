package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"

	"github.com/ShareFrame/user-management/config"
	ATProtocol "github.com/ShareFrame/user-management/internal/atproto"
	"github.com/ShareFrame/user-management/internal/dynamo"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sirupsen/logrus"
)

const defaultTimeZone = "America/Chicago"

func retrieveAdminCredentials(ctx context.Context, secretsManagerClient config.SecretsManagerAPI) (models.AdminCreds, error) {
	input, err := config.RetrieveSecret(ctx, os.Getenv("PDS_ADMIN_SECRET_NAME"), secretsManagerClient)
	if err != nil {
		logrus.WithError(err).WithField("secret_name", os.Getenv("PDS_ADMIN_SECRET_NAME")).Error("Unable to retrieve admin credentials from Secrets Manager")
		return models.AdminCreds{}, fmt.Errorf("could not retrieve admin credentials from Secrets Manager: %w", err)
	}

	var adminCredentials models.AdminCreds
	err = json.Unmarshal([]byte(input), &adminCredentials)
	if err != nil {
		logrus.WithError(err).WithField("secret_content", input).Error("Failed to unmarshal admin credentials")
		return models.AdminCreds{}, fmt.Errorf("invalid admin credentials format in secret: %w", err)
	}

	logrus.Info("Successfully retrieved admin credentials")
	return adminCredentials, nil
}

func validateAndFormatUser(event models.UserRequest) (models.UserRequest, error) {
	if event.Handle == "" || event.Email == "" {
		logrus.WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Warn("Validation failed: handle or email is missing")
		return models.UserRequest{}, fmt.Errorf("handle and email are required fields")
	}

	if !strings.HasSuffix(event.Handle, ".shareframe.social") {
		event.Handle = fmt.Sprintf("%s.shareframe.social", event.Handle)
		logrus.WithField("updated_handle", event.Handle).Info("Updated handle to include domain")
	}

	regex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	baseHandle := strings.TrimSuffix(event.Handle, ".shareframe.social") // Get base handle
	if !regex.MatchString(baseHandle) {
		logrus.WithField("handle", baseHandle).Warn("Validation failed: handle contains invalid characters")
		return models.UserRequest{}, fmt.Errorf("handle must not contain symbols and can only include letters and numbers")
	}

	_, err := mail.ParseAddress(event.Email)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"email": event.Email,
		}).Warn("Validation failed: invalid email format")
		return models.UserRequest{}, fmt.Errorf("invalid email format")
	}

	logrus.Info("User request validated successfully")
	return event, nil
}

func UserHandler(ctx context.Context, event models.UserRequest) (events.APIGatewayProxyResponse, error) {
	logrus.WithField("handle", event.Handle).Info("Processing create account request")

	updatedEvent, err := validateAndFormatUser(event)
	if err != nil {
		logrus.WithError(err).Warn("Validation error")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": %q}`, err.Error()),
		}, nil
	}
	event = updatedEvent

	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to load application configuration")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Could not load application configuration"}`,
		}, nil
	}

	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)
	adminCreds, err := retrieveAdminCredentials(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve admin credentials from Secrets Manager")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Could not retrieve admin credentials"}`,
		}, nil
	}

	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL, &http.Client{})
	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate invite code using AT Protocol")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Could not generate invite code"}`,
		}, nil
	}

	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Error("Failed to register user via AT Protocol")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"User registration failed"}`,
		}, nil
	}

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName, defaultTimeZone)
	err = dynamoClient.StoreUser(user)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": user.DID,
			"handle":  user.Handle,
		}).Error("Failed to store user in DynamoDB")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Failed to store user data"}`,
		}, nil
	}

	responseBody, err := json.Marshal(user)
	if err != nil {
		logrus.WithError(err).WithField("user_id", user.DID).Error("Failed to serialize user response")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Internal Server Error"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}
