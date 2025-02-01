package helper

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"regexp"
	"strings"

	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/dynamo"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/sirupsen/logrus"
)

//go:embed blocked_usernames.json
var blockedUsernamesData []byte
var blockedUsernames []string

const PDS_Suffix = ".shareframe.social"

func init() {
	err := json.Unmarshal(blockedUsernamesData, &blockedUsernames)
	if err != nil {
		logrus.Fatalf("Failed to parse embedded blocked usernames JSON: %v", err)
	}
}

func ValidateAndFormatUser(ctx context.Context, event models.UserRequest, dynamoClient dynamo.DynamoDBService) (models.UserRequest, error) {
	if event.Handle == "" || event.Email == "" {
		logrus.WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Warn("Validation failed: handle or email is missing")
		return models.UserRequest{}, fmt.Errorf("handle and email are required fields")
	}

	baseHandle := strings.TrimSuffix(event.Handle, PDS_Suffix)

	if len(baseHandle) < 3 {
		logrus.WithField("handle", baseHandle).Warn("Validation failed: handle too short")
		return models.UserRequest{}, fmt.Errorf("handle must be at least 3 characters long")
	}
	if len(baseHandle) > 18 {
		logrus.WithField("handle", baseHandle).Warn("Validation failed: handle too long")
		return models.UserRequest{}, fmt.Errorf("handle cannot exceed 18 characters")
	}

	for _, blocked := range blockedUsernames {
		if strings.EqualFold(baseHandle, blocked) {
			logrus.WithField("handle", baseHandle).Warn("Validation failed: handle is restricted")
			return models.UserRequest{}, fmt.Errorf("handle is not allowed")
		}
	}

	if !strings.HasSuffix(event.Handle, PDS_Suffix) {
		event.Handle = fmt.Sprintf("%s%s", baseHandle, PDS_Suffix)
		logrus.WithField("updated_handle", event.Handle).Info("Updated handle to include domain")
	}

	regex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !regex.MatchString(baseHandle) {
		logrus.WithField("handle", baseHandle).Warn("Validation failed: handle contains invalid characters")
		return models.UserRequest{}, fmt.Errorf("handle can only include letters and numbers")
	}

	_, err := mail.ParseAddress(event.Email)
	if err != nil {
		logrus.WithField("email", event.Email).Warn("Validation failed: invalid email format")
		return models.UserRequest{}, fmt.Errorf("invalid email format")
	}

	exists, err := dynamoClient.CheckEmailExists(ctx, event.Email)
	if err != nil {
		logrus.WithError(err).Warn("Failed to check email existence in database")
		return models.UserRequest{}, fmt.Errorf("internal error: failed to check email")
	}
	if exists {
		logrus.WithField("email", event.Email).Warn("Validation failed: email already taken")
		return models.UserRequest{}, fmt.Errorf("email is already registered")
	}

	logrus.Info("User request validated successfully")
	return event, nil
}

func retrieveCredentials[T any](ctx context.Context, secretEnvVar string, secretsManagerClient config.SecretsManagerAPI) (T, error) {
	var creds T
	secretName := os.Getenv(secretEnvVar)

	input, err := config.RetrieveSecret(ctx, secretName, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).
			WithField("secret_name", secretName).
			Error("Failed to retrieve credentials from Secrets Manager")
		return creds, fmt.Errorf("error retrieving credentials from Secrets Manager (%s): %w", secretName, err)
	}

	err = json.Unmarshal([]byte(input), &creds)
	if err != nil {
		logrus.WithError(err).
			WithField("secret_content", input).
			Error("Failed to unmarshal credentials")
		return creds, fmt.Errorf("invalid credentials format: %w", err)
	}

	logrus.WithField("credential_type", fmt.Sprintf("%T", creds)).Info("Successfully retrieved credentials")
	return creds, nil
}

func RetrieveAdminCredentials(ctx context.Context, secretsManagerClient config.SecretsManagerAPI) (models.AdminCreds, error) {
	return retrieveCredentials[models.AdminCreds](ctx, "PDS_ADMIN_SECRET_NAME", secretsManagerClient)
}

func RetrieveUtilAccountCreds(ctx context.Context, secretsManagerClient config.SecretsManagerAPI) (models.UtilACcountCreds, error) {
	return retrieveCredentials[models.UtilACcountCreds](ctx, "PDS_UTIL_ACCOUNT_CREDS", secretsManagerClient)
}
