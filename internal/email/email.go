package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ShareFrame/user-management/config"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/resend/resend-go/v2"
)

func SendEmail(ctx context.Context, recipient string, svc config.SecretsManagerAPI) (string, error) {
	emailCredsStr, err := config.RetrieveSecret(ctx, os.Getenv("EMAIL_SERVICE_KEY"), svc)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	var emailCreds models.EmailCreds
	err = json.Unmarshal([]byte(emailCredsStr), &emailCreds)
	if err != nil {
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
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return sent.Id, nil
}
