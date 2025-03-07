package config

import (
	"context"
	"encoding/json"
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

	validSecret := PostgresSecret{
		Username:     "user",
		Password:     "pass",
		Database:     "testdb",
		Host:         "localhost",
		Port:         "5432",
		DBClusterARN: "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
		SecretARN:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
	}

	validSecretJSON, _ := json.Marshal(validSecret)

	tests := []struct {
		name           string
		envVars        map[string]string
		mockSecret     *secretsmanager.GetSecretValueOutput
		mockSecretErr  error
		expectedErrMsg string
	}{
		{
			name: "Successful Config Load",
			envVars: map[string]string{
				"POSTGRES_CONN_STR": "test-secret",
				"ATPROTO_BASE_URL":  "https://example.com",
			},
			mockSecret: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(string(validSecretJSON)),
			},
			mockSecretErr:  nil,
			expectedErrMsg: "",
		},
		{
			name: "Corrupt JSON in Secret",
			envVars: map[string]string{
				"POSTGRES_CONN_STR": "test-secret",
				"ATPROTO_BASE_URL":  "https://example.com",
			},
			mockSecret: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"invalid": "json"`),
			},
			mockSecretErr:  nil,
			expectedErrMsg: "failed to parse PostgreSQL secret JSON",
		},
		{
			name: "Missing Required Fields in Secret JSON",
			envVars: map[string]string{
				"POSTGRES_CONN_STR": "test-secret",
				"ATPROTO_BASE_URL":  "https://example.com",
			},
			mockSecret: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"username": "", "password": "", "database": "", "host": ""}`),
			},
			mockSecretErr:  nil,
			expectedErrMsg: "parsed PostgreSQL secret is missing required fields",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Clearenv()
			for key, value := range test.envVars {
				os.Setenv(key, value)
			}

			mockSecretsClient.ExpectedCalls = nil
			mockSecretsClient.On("GetSecretValue", mock.Anything, mock.Anything).
				Return(test.mockSecret, test.mockSecretErr)

			_, _, err := LoadConfig(ctx, mockSecretsClient)

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
