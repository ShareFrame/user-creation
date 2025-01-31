package helper

import (
	"context"
	"os"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock SecretsManagerAPI
type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.
	GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestValidateAndFormatUser(t *testing.T) {
	tests := []struct {
		name        string
		input       models.UserRequest
		expectedErr string
		expectedOut models.UserRequest
	}{
		{
			name:        "Valid User",
			input:       models.UserRequest{Handle: "validuser", Email: "user@example.com"},
			expectedErr: "",
			expectedOut: models.UserRequest{Handle: "validuser.shareframe.social", Email: "user@example.com"},
		},
		{
			name:        "Blocked Username",
			input:       models.UserRequest{Handle: "admin", Email: "admin@example.com"},
			expectedErr: "handle is not allowed",
		},
		{
			name:        "Handle missing",
			input:       models.UserRequest{Handle: "", Email: "user@example.com"},
			expectedErr: "handle and email are required fields",
		},
		{
			name:        "Invalid Email",
			input:       models.UserRequest{Handle: "user", Email: "invalid-email"},
			expectedErr: "invalid email format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := ValidateAndFormatUser(test.input)
			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedOut, output)
			}
		})
	}
}

func TestRetrieveCredentials(t *testing.T) {
	mockSM := new(mockSecretsManagerClient)

	tests := []struct {
		name          string
		envVar        string
		secretValue   string
		secretErr     error
		expectedErr   string
		expectedCreds models.AdminCreds
	}{
		{
			name:        "Valid Admin Credentials",
			envVar:      "PDS_ADMIN_SECRET_NAME",
			secretValue: `{"PDS_ADMIN_USERNAME":"admin","PDS_ADMIN_PASSWORD":"secret123", "PDS_JWT_SECRET":"secret123"}`,
			expectedErr: "",
			expectedCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "secret123",
				PDSJWTSecret:     "secret123",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv(test.envVar, test.envVar)

			inputMatcher := mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
				return input.SecretId != nil && *input.SecretId == test.envVar
			})

			if test.secretErr != nil {
				mockSM.On("GetSecretValue", mock.Anything, inputMatcher).
					Return(nil, test.secretErr)
			} else {
				mockSM.On("GetSecretValue", mock.Anything, inputMatcher).
					Return(&secretsmanager.GetSecretValueOutput{
						SecretString: &test.secretValue,
					}, nil)
			}

			creds, err := RetrieveAdminCredentials(context.Background(), mockSM)

			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedCreds, creds)
			}

			mockSM.AssertExpectations(t)
		})
	}
}
