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

func ValidateAndFormatUser(event models.UserRequest) (models.UserRequest, error) {
	if event.Handle == "" || event.Email == "" {
		logrus.WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Warn("Validation failed: handle or email is missing")
		return models.UserRequest{}, fmt.Errorf("handle and email are required fields")
	}

	baseHandle := strings.TrimSuffix(event.Handle, PDS_Suffix)
	for _, blocked := range blockedUsernames {
		if strings.EqualFold(baseHandle, blocked) { // Case-insensitive comparison
			logrus.WithField("handle", baseHandle).Warn("Validation failed: handle is a restricted username")
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
