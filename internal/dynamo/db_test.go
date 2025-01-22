package dynamo

import (
	"context"
	"errors"
	"github.com/Atlas-Mesh/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, input *dynamodb.PutItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func TestStoreUser(t *testing.T) {
	tests := []struct {
		name          string
		user          models.CreateUserResponse
		mockOutput    *dynamodb.PutItemOutput
		mockError     error
		expectedError string
		timeZone      string
		mockExpected  bool
	}{
		{
			name: "Successful Store",
			user: models.CreateUserResponse{
				DID:    "did:example:123",
				Email:  "user@example.com",
				Handle: "user123",
			},
			mockOutput:    &dynamodb.PutItemOutput{},
			mockError:     nil,
			expectedError: "",
			timeZone:      "UTC",
			mockExpected:  true,
		},
		{
			name: "Missing User DID",
			user: models.CreateUserResponse{
				DID:    "",
				Email:  "user@example.com",
				Handle: "user123",
			},
			mockOutput:    nil,
			mockError:     nil,
			expectedError: "user DID and handle are required",
			timeZone:      "UTC",
			mockExpected:  false,
		},
		{
			name: "Invalid Time Zone",
			user: models.CreateUserResponse{
				DID:    "did:example:123",
				Email:  "user@example.com",
				Handle: "user123",
			},
			mockOutput:    nil,
			mockError:     nil,
			expectedError: "failed to load time zone: unknown time zone Foo/Bar",
			timeZone:      "Foo/Bar",
			mockExpected:  false,
		},
		{
			name: "DynamoDB Error",
			user: models.CreateUserResponse{
				DID:    "did:example:123",
				Email:  "user@example.com",
				Handle: "user123",
			},
			mockOutput:    nil,
			mockError:     errors.New("DynamoDB error"),
			expectedError: "failed to store user in DynamoDB: DynamoDB error",
			timeZone:      "UTC",
			mockExpected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockDynamoDBClient)

			if tt.mockExpected {
				mockClient.On("PutItem", mock.Anything, mock.MatchedBy(
					func(input *dynamodb.PutItemInput) bool {
						return input.TableName != nil && *input.TableName == "UsersTable"
					})).Return(tt.mockOutput, tt.mockError)
			}

			client := NewDynamoClient(mockClient, "UsersTable", tt.timeZone)

			err := client.StoreUser(tt.user)

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
			}

			if tt.mockExpected {
				mockClient.AssertExpectations(t)
			}
		})
	}
}
