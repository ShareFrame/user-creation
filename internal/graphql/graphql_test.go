package graphql

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSecretsManagerClient struct {
	mock.Mock
}

func (m *MockSecretsManagerClient) GetSecretValue(ctx context.Context,
	input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.
	GetSecretValueOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestHandler(t *testing.T) {
	// Mock environment variables
	os.Setenv("DYNAMO_TABLE_NAME", "test_table")
	os.Setenv("PDS_ADMIN_SECRET_NAME", "test_secret_name")
	os.Setenv("EMAIL_SERVICE_KEY", "test_email_key")
	os.Setenv("ATPROTO_BASE_URL", "https://example.com")
	defer func() {
		os.Unsetenv("DYNAMO_TABLE_NAME")
		os.Unsetenv("PDS_ADMIN_SECRET_NAME")
		os.Unsetenv("EMAIL_SERVICE_KEY")
		os.Unsetenv("ATPROTO_BASE_URL")
	}()

	// Mock Secrets Manager
	mockSecretsManager := &mock.MockSecretsManagerClient{
		GetSecretValueFunc: func(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
			if aws.ToString(input.SecretId) == "test_secret_name" {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{"username": "admin", "password": "admin123"}`),
				}, nil
			}
			return nil, fmt.Errorf("secret not found")
		},
	}

	tests := []struct {
		name           string
		inputBody      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid CreateUser Mutation",
			inputBody: `{
				"query": "mutation CreateUser($input: CreateUserInput!) { CreateUser(input: $input) { message } }",
				"operationName": "CreateUser",
				"variables": {
					"input": {
						"handle": "testuser",
						"email": "testuser@example.com"
					}
				}
			}`,
			expectedStatus: 201,
			expectedBody:   "User registered successfully",
		},
		// Additional test cases...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock context and inject the mock client
			ctx := context.WithValue(context.TODO(), config.SecretsManagerClientKey, mockSecretsManager)

			// Create a new APIGatewayProxyRequest with the input body
			request := events.APIGatewayProxyRequest{
				Body: tt.inputBody,
			}

			// Call the GraphQL Handler
			response, err := Handler(ctx, request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, response.StatusCode)
			assert.Equal(t, tt.expectedBody, response.Body)
		})
	}
}
