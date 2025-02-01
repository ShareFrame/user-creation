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

const (
	PDS_Suffix = ".shareframe.social"
	Password_Error = "password must be at least 8 characters long and include at least one uppercase letter, one lowercase letter, one digit, and one special character"
)

func init() {
	err := json.Unmarshal(blockedUsernamesData, &blockedUsernames)
	if err != nil {
		logrus.Fatalf("Failed to parse embedded blocked usernames JSON: %v", err)
	}
}

func ValidateAndFormatUser(ctx context.Context, event models.UserRequest, dynamoClient dynamo.DynamoDBService) (models.UserRequest, error) {
	if event.Handle == "" || event.Email == "" || event.Password == "" {
		logrus.Warn("Validation failed: missing handle, email, or password")
		return models.UserRequest{}, fmt.Errorf("handle, email, and password are required fields")
	}

	baseHandle := strings.TrimSuffix(event.Handle, PDS_Suffix)

	if err := validateHandle(baseHandle); err != nil {
		logrus.WithField("handle", baseHandle).Warn("Validation failed:", err)
		return models.UserRequest{}, err
	}

	event.Handle = ensureHandleSuffix(baseHandle)

	if err := validateEmail(event.Email); err != nil {
		logrus.WithField("email", event.Email).Warn("Validation failed:", err)
		return models.UserRequest{}, err
	}

	if err := ValidatePassword(event.Password); err != nil {
		logrus.WithError(err).Warn("Validation failed: invalid password")
		return models.UserRequest{}, fmt.Errorf("password validation failed: %w", err)
	}

	exists, err := dynamoClient.CheckEmailExists(ctx, event.Email)
	if err != nil {
		logrus.WithError(err).Error("Database error: failed to check email existence")
		return models.UserRequest{}, fmt.Errorf("internal error: failed to check email")
	}
	if exists {
		logrus.WithField("email", event.Email).Warn("Validation failed: email already taken")
		return models.UserRequest{}, fmt.Errorf("email is already registered")
	}

	logrus.Info("User request validated successfully")
	return event, nil
}

func validateHandle(handle string) error {
	if len(handle) < 3 {
		return fmt.Errorf("handle must be at least 3 characters long")
	}
	if len(handle) > 18 {
		return fmt.Errorf("handle cannot exceed 18 characters")
	}
	for _, blocked := range blockedUsernames {
		if strings.EqualFold(handle, blocked) {
			return fmt.Errorf("handle is not allowed")
		}
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(handle) {
		return fmt.Errorf("handle can only include letters and numbers")
	}
	return nil
}

func ensureHandleSuffix(handle string) string {
	fullHandle := handle + PDS_Suffix
	logrus.WithField("updated_handle", fullHandle).Info("Updated handle to include domain")
	return fullHandle
}

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf(Password_Error)
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:'",.<>?/\\]`).MatchString(password)

	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return fmt.Errorf(Password_Error)
	}

	return nil
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
