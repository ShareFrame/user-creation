package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/mock"
)

type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.
	GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig *Config
		expectedError  string
	}{
		{
			name: "Successful Load",
			envVars: map[string]string{
				"DYNAMO_TABLE_NAME": "UsersTable",
				"ATPROTO_BASE_URL":  "https://example.com",
			},
			expectedConfig: &Config{
				DynamoTableName: "UsersTable",
				AtProtoBaseURL:  "https://example.com",
			},
			expectedError: "",
		},
		{
			name: "Missing DynamoTableName",
			envVars: map[string]string{
				"ATPROTO_BASE_URL": "https://example.com",
			},
			expectedConfig: nil,
			expectedError:  "DYNAMO_TABLE_NAME environment variable is required",
		},
		{
			name: "Missing AtProtoBaseURL",
			envVars: map[string]string{
				"DYNAMO_TABLE_NAME": "UsersTable",
			},
			expectedConfig: nil,
			expectedError:  "ATPROTO_BASE_URL environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			ctx := context.Background()
			cfg, _, err := LoadConfig(ctx)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("Expected error %q, got none", tt.expectedError)
				}
				if err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if cfg.DynamoTableName != tt.expectedConfig.DynamoTableName {
					t.Errorf("Expected DynamoTableName %q, got %q",
						tt.expectedConfig.DynamoTableName, cfg.DynamoTableName)
				}
				if cfg.AtProtoBaseURL != tt.expectedConfig.AtProtoBaseURL {
					t.Errorf("Expected AtProtoBaseURL %q, got %q",
						tt.expectedConfig.AtProtoBaseURL, cfg.AtProtoBaseURL)
				}
			}
		})
	}
}

func TestRetrieveAdminCreds(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		mockOutput    *secretsmanager.GetSecretValueOutput
		mockError     error
		expectedCreds string
		expectedError string
	}{
		{
			name: "Successful Secret Retrieval",
			envVars: map[string]string{
				"PDS_ADMIN_SECRET_NAME": "test-secret",
				"AWS_REGION":            "us-east-1",
			},
			mockOutput: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("admin-creds"),
			},
			mockError:     nil,
			expectedCreds: "admin-creds",
			expectedError: "",
		},
		{
			name: "Missing Secret Name Environment Variable",
			envVars: map[string]string{
				"AWS_REGION": "us-east-1",
			},
			mockOutput:    nil,
			mockError:     nil,
			expectedCreds: "",
			expectedError: "secret name is required",
		},
		{
			name: "AWS Secrets Manager Error",
			envVars: map[string]string{
				"PDS_ADMIN_SECRET_NAME": "test-secret",
				"AWS_REGION":            "us-east-1",
			},
			mockOutput:    nil,
			mockError:     errors.New("AWS error"),
			expectedCreds: "",
			expectedError: "failed to retrieve secret: AWS error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			// Unset environment variables after the test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			mockClient := new(mockSecretsManagerClient)
			ctx := context.Background()

			if tt.mockOutput != nil || tt.mockError != nil {
				mockClient.On("GetSecretValue", ctx, mock.Anything).Return(tt.mockOutput, tt.mockError)
			}

			creds, err := RetrieveSecret(ctx, tt.envVars["PDS_ADMIN_SECRET_NAME"], mockClient)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("Expected error %q, got none", tt.expectedError)
				}
				if err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if creds != tt.expectedCreds {
					t.Errorf("Expected creds %q, got %q", tt.expectedCreds, creds)
				}
			}

			// Assert mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}
