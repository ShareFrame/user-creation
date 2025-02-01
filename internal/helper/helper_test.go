package helper

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDynamoClient struct {
	mock.Mock
}

func (m *mockDynamoClient) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestValidateAndFormatUser(t *testing.T) {
	mockDynamo := new(mockDynamoClient)

	tests := []struct {
		name        string
		input       models.UserRequest
		emailExists bool
		emailErr    error
		expectedErr string
		expectedOut models.UserRequest
	}{
		{
			name:        "Valid User",
			input:       models.UserRequest{Handle: "validuser", Email: "user@example.com"},
			emailExists: false,
			expectedErr: "",
			expectedOut: models.UserRequest{Handle: "validuser.shareframe.social", Email: "user@example.com"},
		},
		{
			name:        "Email Already Taken",
			input:       models.UserRequest{Handle: "validuser", Email: "user@example.com"},
			emailExists: true,
			expectedErr: "email is already registered",
		},
		{
			name:        "DynamoDB Email Check Error",
			input:       models.UserRequest{Handle: "validuser", Email: "user@example.com"},
			emailExists: false,
			emailErr:    errors.New("DynamoDB error"),
			expectedErr: "internal error: failed to check email",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockDynamo.On("CheckEmailExists", mock.Anything, test.input.Email).
				Return(test.emailExists, test.emailErr).Once()

			output, err := ValidateAndFormatUser(context.TODO(), test.input, mockDynamo)

			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedOut, output)
			}

			mockDynamo.AssertExpectations(t)
		})
	}
}

func TestRetrieveAdminCredentials(t *testing.T) {
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
			secretValue: `{"PDS_ADMIN_USERNAME":"admin","PDS_ADMIN_PASSWORD":"secret123","PDS_JWT_SECRET":"jwtsecret"}`,
			expectedErr: "",
			expectedCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "secret123",
				PDSJWTSecret:     "jwtsecret",
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

func TestRetrieveUtilAccountCreds(t *testing.T) {
	mockSM := new(mockSecretsManagerClient)

	tests := []struct {
		name          string
		envVar        string
		secretValue   string
		secretErr     error
		expectedErr   string
		expectedCreds models.UtilACcountCreds
	}{
		{
			name:        "Valid Util Account Credentials",
			envVar:      "PDS_UTIL_ACCOUNT_CREDS",
			secretValue: `{"username":"utiluser","password":"utilpass", "did": "did:3:12345"}`,
			expectedErr: "",
			expectedCreds: models.UtilACcountCreds{
				Username: "utiluser",
				Password: "utilpass",
				DID:      "did:3:12345",
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

			creds, err := RetrieveUtilAccountCreds(context.Background(), mockSM)

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
