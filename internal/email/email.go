package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/resend/resend-go/v2"
	"github.com/sirupsen/logrus"
)

func SendEmail(ctx context.Context, recipient string, svc config.SecretsManagerAPI) (string, error) {
	emailCredsStr, err := config.RetrieveSecret(ctx, os.Getenv("EMAIL_SERVICE_KEY"), svc)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve email service credentials")
		return "", fmt.Errorf("failed to retrieve email service credentials: %w", err)
	}

	var emailCreds models.EmailCreds
	if err := json.Unmarshal([]byte(emailCredsStr), &emailCreds); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal email credentials")
		return "", fmt.Errorf("failed to unmarshal email credentials: %w", err)
	}

	client := resend.NewClient(emailCreds.APIKey)

	params := &resend.SendEmailRequest{
		From:    "Admin <admin@shareframe.social>",
		To:      []string{recipient},
		Html:    "<strong>hello world</strong>",
		Subject: "Welcome to ShareFrame",
		ReplyTo: "replyto@example.com",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		logrus.WithError(err).Error("Failed to send email")
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	logrus.WithField("email_id", sent.Id).Info("Email sent successfully")

	return sent.Id, nil
}
