package email

import (
	"context"
	"errors"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/resend/resend-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockEmailClient struct {
	mock.Mock
}

func (m *MockEmailClient) Send(params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*resend.SendEmailResponse), args.Error(1)
}

type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.
	GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestSendEmail(t *testing.T) {
	mockClient := new(MockEmailClient)
	ctx := context.Background()

	tests := []struct {
		name          string
		recipient     string
		mockResponse  *resend.SendEmailResponse
		mockError     error
		expectedID    string
		expectedError string
	}{
		{
			name:          "Success case",
			recipient:     "test@example.com",
			mockResponse:  &resend.SendEmailResponse{Id: "email123"},
			mockError:     nil,
			expectedID:    "email123",
			expectedError: "",
		},
		{
			name:          "Empty recipient",
			recipient:     "",
			mockResponse:  nil,
			mockError:     nil,
			expectedID:    "",
			expectedError: "recipient email address is required",
		},
		{
			name:          "Send email failure",
			recipient:     "test@example.com",
			mockResponse:  nil,
			mockError:     errors.New("network error"),
			expectedID:    "",
			expectedError: "failed to send email to test@example.com: network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ExpectedCalls = nil

			if tt.recipient != "" {
				params := &resend.SendEmailRequest{
					From:    "Admin <admin@shareframe.social>",
					To:      []string{tt.recipient},
					Html:    "<strong>hello world</strong>",
					Subject: "Welcome to ShareFrame",
					ReplyTo: "replyto@example.com",
				}

				response := tt.mockResponse
				if response == nil {
					response = (*resend.SendEmailResponse)(nil)
				}
				mockClient.On("Send", params).Return(response, tt.mockError)
			}

			id, err := SendEmail(ctx, tt.recipient, mockClient)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedID, id)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetEmailCreds(t *testing.T) {
	mockSvc := new(mockSecretsManagerClient)
	ctx := context.Background()

	tests := []struct {
		name          string
		secretKey     string
		mockOutput    *secretsmanager.GetSecretValueOutput
		mockError     error
		expectedCreds models.EmailCreds
		expectedError string
	}{
		{
			name:      "Success case",
			secretKey: "EMAIL_SERVICE_KEY",
			mockOutput: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"RESEND_APIKEY": "test-key"}`),
			},
			mockError:     nil,
			expectedCreds: models.EmailCreds{APIKey: "test-key"},
			expectedError: "",
		},
		{
			name:          "Missing secret key",
			secretKey:     "",
			mockOutput:    nil,
			mockError:     nil,
			expectedCreds: models.EmailCreds{},
			expectedError: "email service secret key is required",
		},
		{
			name:          "Secrets Manager error",
			secretKey:     "EMAIL_SERVICE_KEY",
			mockOutput:    nil,
			mockError:     errors.New("access denied"),
			expectedCreds: models.EmailCreds{},
			expectedError: "failed to retrieve email service credentials: failed to retrieve secret: access denied",
		},
		{
			name:      "Unmarshal error",
			secretKey: "EMAIL_SERVICE_KEY",
			mockOutput: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`invalid: json`),
			},
			mockError:     nil,
			expectedCreds: models.EmailCreds{},
			expectedError: "failed to unmarshal email credentials: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.ExpectedCalls = nil

			input := &secretsmanager.GetSecretValueInput{
				SecretId:     aws.String(tt.secretKey),
				VersionStage: aws.String("AWSCURRENT"),
			}

			if tt.secretKey != "" {
				output := tt.mockOutput
				if output == nil {
					output = (*secretsmanager.GetSecretValueOutput)(nil)
				}
				mockSvc.On("GetSecretValue", ctx, input).Return(output, tt.mockError)
			}

			creds, err := GetEmailCreds(ctx, mockSvc, tt.secretKey)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedCreds, creds)
			mockSvc.AssertExpectations(t)
		})
	}
}
