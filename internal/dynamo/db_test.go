package dynamo

import (
	"context"
	"errors"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"
)

type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, input *dynamodb.PutItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) Query(ctx context.Context, input *dynamodb.QueryInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func TestStoreUser(t *testing.T) {
	tests := []struct {
		name          string
		user          models.CreateUserResponse
		event         models.UserRequest
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
				Handle: "user123",
			},
			event: models.UserRequest{
				Handle: "user123",
				Email:  "user@example.com",
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
				Handle: "user123",
			},
			event: models.UserRequest{
				Handle: "user123",
				Email:  "user@example.com",
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
				Handle: "user123",
			},
			event: models.UserRequest{
				Handle: "user123",
				Email:  "user@example.com",
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

			err := client.StoreUser(tt.user, tt.event)

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

func TestCheckEmailExists(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		mockOutput    *dynamodb.QueryOutput
		mockError     error
		expectedExist bool
		expectedError string
	}{
		{
			name:  "Email Exists",
			email: "existing@example.com",
			mockOutput: &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"Email": &types.AttributeValueMemberS{Value: "existing@example.com"}},
				},
			},
			mockError:     nil,
			expectedExist: true,
			expectedError: "",
		},
		{
			name:          "Email Does Not Exist",
			email:         "new@example.com",
			mockOutput:    &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}},
			mockError:     nil,
			expectedExist: false,
			expectedError: "",
		},
		{
			name:          "DynamoDB Query Error",
			email:         "error@example.com",
			mockOutput:    nil,
			mockError:     errors.New("DynamoDB query error"),
			expectedExist: false,
			expectedError: "failed to check email existence: DynamoDB query error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockDynamoDBClient)
			mockClient.On("Query", mock.Anything, mock.MatchedBy(
				func(input *dynamodb.QueryInput) bool {
					return input.TableName != nil && *input.TableName == "UsersTable"
				})).Return(tt.mockOutput, tt.mockError)

			client := NewDynamoClient(mockClient, "UsersTable", "UTC")

			exists, err := client.CheckEmailExists(context.TODO(), tt.email)

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

			if exists != tt.expectedExist {
				t.Errorf("Expected existence %v, got %v", tt.expectedExist, exists)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
