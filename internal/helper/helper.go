package helper

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/sirupsen/logrus"
)

//go:embed blocked_usernames.json
var blockedUsernamesData []byte
var blockedUsernames []string

const (
	PDS_Suffix     = ".shareframe.social"
	PasswordError  = "password must be at least 8 characters long and include at least one uppercase letter, one lowercase letter, one digit, and one special character"
	MissingFields  = "handle, email, and password are required fields"
	EmailTaken     = "email is already registered"
	InvalidHandle  = "handle can only include letters and numbers"
	BlockedHandle  = "handle is not allowed"
	HandleTooShort = "handle must be at least 3 characters long"
	HandleTooLong  = "handle cannot exceed 18 characters"
	QueryTimeout   = 3 * time.Second
)

var (
	emailRegex       = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	handleRegex      = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	upperCaseRegex   = regexp.MustCompile(`[A-Z]`)
	lowerCaseRegex   = regexp.MustCompile(`[a-z]`)
	digitRegex       = regexp.MustCompile(`\d`)
	specialCharRegex = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:'",.<>?/\\]`)
)

func init() {
	if err := json.Unmarshal(blockedUsernamesData, &blockedUsernames); err != nil {
		logrus.Fatalf("Failed to parse blocked usernames JSON: %v", err)
	}
}

type DatabaseService interface {
	CheckEmailExists(ctx context.Context, email string) (bool, error)
	StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error
}

func ValidateAndFormatUser(ctx context.Context, event models.UserRequest, dbClient DatabaseService) (models.UserRequest, error) {
	if event.Handle == "" || event.Email == "" || event.Password == "" {
		logrus.Warn("Validation failed: missing required fields")
		return models.UserRequest{}, fmt.Errorf("%v", MissingFields)
	}

	baseHandle := strings.TrimSuffix(event.Handle, PDS_Suffix)

	if err := ValidateHandle(baseHandle); err != nil {
		logrus.WithField("handle", baseHandle).Warnf("Validation failed: %v", err)
		return models.UserRequest{}, err
	}
	event.Handle = EnsureHandleSuffix(baseHandle)

	if err := ValidateEmail(event.Email); err != nil {
		logrus.WithField("email", event.Email).Warnf("Validation failed: %v", err)
		return models.UserRequest{}, err
	}

	if err := ValidatePassword(event.Password); err != nil {
		logrus.WithError(err).Warn("Validation failed: invalid password")
		return models.UserRequest{}, fmt.Errorf("password validation failed: %w", err)
	}

	logrus.Infof("Calling CheckEmailExists for email: %s", event.Email)
	exists, err := dbClient.CheckEmailExists(ctx, event.Email)
	if err != nil {
		logrus.WithError(err).Error("Database error: failed to check email existence")
		return models.UserRequest{}, fmt.Errorf("internal error: failed to check email")
	}
	if exists {
		logrus.WithField("email", event.Email).Warn("Validation failed: email already taken")
		return models.UserRequest{}, fmt.Errorf("%v", EmailTaken)
	}

	logrus.Info("User request validated successfully")
	return event, nil
}

func ValidateHandle(handle string) error {
	if len(handle) < 3 {
		return fmt.Errorf("%v: %v", HandleTooShort, handle)
	}
	if len(handle) > 18 {
		return fmt.Errorf("%v: %v", HandleTooLong, handle)
	}
	for _, blocked := range blockedUsernames {
		if strings.EqualFold(handle, blocked) {
			return fmt.Errorf("%v: %v", BlockedHandle, handle)
		}
	}
	if !handleRegex.MatchString(handle) {
		return fmt.Errorf("%v: %v", InvalidHandle, handle)
	}
	return nil
}

func EnsureHandleSuffix(handle string) string {
    trimmedHandle := strings.TrimSpace(handle)

    if strings.HasSuffix(trimmedHandle, PDS_Suffix) {
        return trimmedHandle
    }
	
    return trimmedHandle + PDS_Suffix
}


func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 ||
		!upperCaseRegex.MatchString(password) ||
		!lowerCaseRegex.MatchString(password) ||
		!digitRegex.MatchString(password) ||
		!specialCharRegex.MatchString(password) {
		return fmt.Errorf(PasswordError)
	}
	return nil
}

func retrieveCredentials[T any](ctx context.Context, secretEnvVar string, secretsManagerClient config.SecretsManagerAPI) (T, error) {
	var creds T
	secretName := os.Getenv(secretEnvVar)

	input, err := config.RetrieveSecret(ctx, secretName, secretsManagerClient)
	if err != nil {
		logrus.WithFields(logrus.Fields{"secret_name": secretName}).WithError(err).Error("Failed to retrieve credentials")
		return creds, fmt.Errorf("error retrieving credentials from Secrets Manager (%s): %w", secretName, err)
	}

	if err := json.Unmarshal([]byte(input), &creds); err != nil {
		logrus.WithFields(logrus.Fields{"secret_name": secretName, "secret_value": input}).WithError(err).Error("Failed to unmarshal credentials")
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
