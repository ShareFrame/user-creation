package email

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/resend/resend-go/v2"
	"github.com/sirupsen/logrus"
)

//go:embed email_template.html
var emailTemplate string

type EmailClient interface {
	Send(params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

type ResendEmailClient struct {
	client *resend.Client
}

func NewResendEmailClient(apiKey string) *ResendEmailClient {
	if apiKey == "" {
		logrus.Fatal("API key for ResendEmailClient is missing")
	}
	return &ResendEmailClient{
		client: resend.NewClient(apiKey),
	}
}

func (c *ResendEmailClient) Send(params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	return c.client.Emails.Send(params)
}

func SendEmail(ctx context.Context, recipient string, client EmailClient) (string, error) {
	if recipient == "" {
		return "", errors.New("recipient email address is required")
	}

	validationLink := "https://shareframe.social/verify?token=abc123"
	emailBody := strings.Replace(emailTemplate, "[VALIDATION_LINK]", validationLink, -1)

	params := &resend.SendEmailRequest{
		From:    "Admin <admin@shareframe.social>",
		To:      []string{recipient},
		Html:    emailBody,
		Subject: "Welcome to ShareFrame",
		ReplyTo: "replyto@example.com",
	}

	response, err := client.Send(params)
	if err != nil {
		logrus.WithError(err).WithField("recipient", recipient).Error("Failed to send email")
		return "", fmt.Errorf("failed to send email to %s: %w", recipient, err)
	}

	logrus.WithFields(logrus.Fields{
		"email_id":  response.Id,
		"recipient": recipient,
	}).Info("Email sent successfully")
	return response.Id, nil
}

func GetEmailCreds(ctx context.Context, svc config.SecretsManagerAPI, secretKey string) (models.EmailCreds, error) {
	if secretKey == "" {
		return models.EmailCreds{}, errors.New("email service secret key is required")
	}

	secretValue, err := config.RetrieveSecret(ctx, secretKey, svc)
	if err != nil {
		logrus.WithError(err).WithField("secret_key", secretKey).Error("Failed to retrieve email service credentials")
		return models.EmailCreds{}, fmt.Errorf("failed to retrieve email service credentials: %w", err)
	}

	var emailCreds models.EmailCreds
	if err := json.Unmarshal([]byte(secretValue), &emailCreds); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal email credentials")
		return models.EmailCreds{}, fmt.Errorf("failed to unmarshal email credentials: %w", err)
	}

	return emailCreds, nil
}

func ValidateEnvironment() error {
	requiredEnvVars := []string{"EMAIL_SERVICE_KEY"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("environment variable %s is not set", envVar)
		}
	}
	return nil
}
