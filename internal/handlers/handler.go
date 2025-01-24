package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/ShareFrame/user-management/config"
	ATProtocol "github.com/ShareFrame/user-management/internal/atproto"
	"github.com/ShareFrame/user-management/internal/dynamo"
	"github.com/ShareFrame/user-management/internal/email"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sirupsen/logrus"
)

const defaultTimeZone = "America/Chicago"

func retrieveAdminCredentials(ctx context.Context,
	secretsManagerClient config.SecretsManagerAPI) (models.AdminCreds, error) {
	input, err := config.RetrieveSecret(ctx, os.Getenv("PDS_ADMIN_SECRET_NAME"), secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve admin credentials")
		return models.AdminCreds{}, fmt.Errorf("failed to retrieve admin credentials: %w", err)
	}

	var adminCredentials models.AdminCreds
	err = json.Unmarshal([]byte(input), &adminCredentials)
	if err != nil {
		logrus.WithError(err).Error("Failed to unmarshal admin credentials")
		return models.AdminCreds{}, fmt.Errorf("failed to unmarshal admin credentials: %w", err)
	}

	logrus.Info("Successfully retrieved admin credentials")
	return adminCredentials, nil
}

func validateAndFormatUser(event models.UserRequest) (models.UserRequest, *events.APIGatewayProxyResponse, error) {
	if event.Handle == "" || event.Email == "" {
		logrus.WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Warn("Invalid request: missing handle or email")
		return models.UserRequest{}, &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request: handle and email are required",
		}, nil
	}

	if !strings.HasSuffix(event.Handle, ".shareframe.social") {
		event.Handle = fmt.Sprintf("%s.shareframe.social", event.Handle)
		logrus.WithField("updated_handle", event.Handle).Info("Updated handle to include domain")
	}

	_, err := mail.ParseAddress(event.Email)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"email": event.Email,
		}).Warn("Invalid email format")
		return models.UserRequest{}, &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Invalid request: email address is not properly formatted",
		}, nil
	}

	logrus.Info("User request validated successfully")
	return event, nil, nil
}

func UserHandler(ctx context.Context, event models.UserRequest) (events.APIGatewayProxyResponse, error) {
	logrus.WithField("handle", event.Handle).Info("Processing create account request")

	updatedEvent, validationResponse, err := validateAndFormatUser(event)
	if err != nil {
		logrus.WithError(err).Error("Validation error")
		return *validationResponse, nil
	}
	event = updatedEvent

	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to load configuration")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)

	adminCreds, err := retrieveAdminCredentials(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve admin credentials")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to retrieve admin credentials: %w", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName, defaultTimeZone)

	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL, &http.Client{})

	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		logrus.WithError(err).Error("Failed to create invite code")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to create invite code: %w", err)
	}

	emailCreds, err := email.GetEmailCreds(ctx, secretsManagerClient, os.Getenv("EMAIL_SERVICE_KEY"))
	if err != nil {
		logrus.WithError(err).Error("Failed to get email credentials")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to get email credentials: %w", err)
	}

	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to register user: %w", err)
	}
	logrus.WithField("handle", user.Handle).Info("User registered successfully")

	emailClient := email.NewResendEmailClient(emailCreds.APIKey)

	sendId, err := email.SendEmail(ctx, event.Email, emailClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to send email")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to send email: %w", err)
	}
	logrus.WithField("send_id", sendId).Info("Email sent successfully")

	// Store user in DynamoDB
	err = dynamoClient.StoreUser(user)
	if err != nil {
		logrus.WithError(err).Error("Failed to store user in DynamoDB")
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to store user: %w", err)
	}

	logrus.WithField("did", user.DID).Info("Account created successfully")

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       "User registered successfully",
	}, nil
}
