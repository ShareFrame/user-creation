package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) != nil {
		return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestLoadConfig(t *testing.T) {
	mockSecretsClient := new(mockSecretsManagerClient)
	ctx := context.Background()

	tests := []struct {
		name           string
		envVars        map[string]string
		expectedErrMsg string
	}{
		{
			name: "Successful Config Load",
			envVars: map[string]string{
				"ATPROTO_BASE_URL": "https://example.com",
			},
			expectedErrMsg: "",
		},
		{
			name: "Missing ATPROTO_BASE_URL",
			envVars: map[string]string{},
			expectedErrMsg: "ATPROTO_BASE_URL environment variable is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Clearenv()
			for key, value := range test.envVars {
				os.Setenv(key, value)
			}

			mockSecretsClient.ExpectedCalls = nil

			_, _, err := LoadConfig(ctx, mockSecretsClient)

			if test.expectedErrMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErrMsg)
			}
		})
	}
}

func TestRetrieveSecret(t *testing.T) {
	mockSecretsClient := new(mockSecretsManagerClient)
	ctx := context.Background()

	tests := []struct {
		name           string
		mockSecret     *secretsmanager.GetSecretValueOutput
		mockSecretErr  error
		expectedResult string
		expectedErrMsg string
	}{
		{
			name: "Successful Secret Retrieval",
			mockSecret: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("super-secret-value"),
			},
			mockSecretErr:  nil,
			expectedResult: "super-secret-value",
			expectedErrMsg: "",
		},
		{
			name:           "Secrets Manager Returns Error",
			mockSecret:     nil,
			mockSecretErr:  errors.New("failed to retrieve secret"),
			expectedResult: "",
			expectedErrMsg: "failed to retrieve secret",
		},
		{
			name: "Secret String is Nil",
			mockSecret: &secretsmanager.GetSecretValueOutput{
				SecretString: nil,
			},
			mockSecretErr:  nil,
			expectedResult: "",
			expectedErrMsg: "secret string is nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSecretsClient.ExpectedCalls = nil
			mockSecretsClient.On("GetSecretValue", mock.Anything, mock.Anything).
				Return(test.mockSecret, test.mockSecretErr)

			result, err := RetrieveSecret(ctx, "test-secret", mockSecretsClient)

			assert.Equal(t, test.expectedResult, result)

			if test.expectedErrMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErrMsg)
			}

			mockSecretsClient.AssertExpectations(t)
		})
	}
}
