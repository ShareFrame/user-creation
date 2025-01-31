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

func retrieveUtilAccountCreds(ctx context.Context, secretsManagerClient config.SecretsManagerAPI) (models.UtilACcountCreds, error) {
	secretName := os.Getenv("PDS_UTIL_ACCOUNT_CREDS")
	input, err := config.RetrieveSecret(ctx, secretName, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).
			WithField("secret_name", secretName).
			Error("Failed to retrieve util account credentials from Secrets Manager")
		return models.UtilACcountCreds{}, fmt.Errorf("error retrieving util account credentials from Secrets Manager (%s): %w", secretName, err)
	}

	var utilAccountCreds models.UtilACcountCreds
	err = json.Unmarshal([]byte(input), &utilAccountCreds)
	if err != nil {
		logrus.WithError(err).
			WithField("secret_content", input).
			Error("Failed to unmarshal util account credentials")
		return models.UtilACcountCreds{}, fmt.Errorf("invalid util account credentials format: %w", err)
	}

	logrus.WithField("username", utilAccountCreds.Username).Info("Successfully retrieved util account credentials")
	return utilAccountCreds, nil
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

func UserHandler(ctx context.Context, event models.UserRequest) (*models.CreateUserResponse, error) {
	logrus.WithField("handle", event.Handle).Info("Processing create account request")

	updatedEvent, err := validateAndFormatUser(event)
	if err != nil {
		logrus.WithError(err).Warn("Validation error")
		return nil, fmt.Errorf("validation error: %w", err)
	}
	event = updatedEvent

	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to load application configuration")
		return nil, fmt.Errorf("internal error: failed to load application configuration")
	}

	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)
	adminCreds, err := retrieveAdminCredentials(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve admin credentials from Secrets Manager")
		return nil, fmt.Errorf("internal error: could not retrieve admin credentials")
	}

	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL, &http.Client{})
	logrus.WithField("base_url", cfg.AtProtoBaseURL).Info("Initializing ATProtocol client")

	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate invite code using AT Protocol")
		return nil, fmt.Errorf("internal error: failed to generate invite code")
	}

	utilAccountCreds, err := retrieveUtilAccountCreds(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve util account credentials from Secrets Manager")
		return nil, fmt.Errorf("internal error: could not retrieve authentication credentials")
	}

	session, err := atProtoClient.CreateSession(utilAccountCreds.Username, utilAccountCreds.Password)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": utilAccountCreds.Username,
			"error":    err.Error(),
		}).Error("Failed to authenticate with AT Protocol")
		return nil, fmt.Errorf("authentication failed for user %s", utilAccountCreds.Username)
	}

	logrus.Info("Session created successfully")

	exists, err := atProtoClient.CheckUserExists(event.Handle, session.AccessJwt)
	if err != nil {
		logrus.WithError(err).WithField("handle", event.Handle).Error("Failed to check user existence")
		return nil, fmt.Errorf("internal error: failed to check if user exists")
	}

	if exists {
		logrus.WithField("handle", event.Handle).Warn("User already exists on PDS")
		return nil, fmt.Errorf("user already exists with handle: %s", event.Handle)
	}

	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Error("Failed to register user via AT Protocol")
		return nil, fmt.Errorf("failed to register user: %s", err)
	}

	dynamoDBClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := dynamo.NewDynamoClient(dynamoDBClient, cfg.DynamoTableName, defaultTimeZone)
	err = dynamoClient.StoreUser(user)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": user.DID,
			"handle":  user.Handle,
		}).Error("Failed to store user in DynamoDB")
		return nil, fmt.Errorf("internal error: failed to store user data")
	}

	logrus.WithFields(logrus.Fields{
		"did":    user.DID,
		"handle": user.Handle,
	}).Info("Successfully created and stored user")

	return &user, nil
}