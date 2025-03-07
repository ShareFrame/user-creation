package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRDSClient struct {
	mock.Mock
}

func (m *mockRDSClient) ExecuteStatement(ctx context.Context, input *rdsdata.ExecuteStatementInput, opts ...func(*rdsdata.Options)) (*rdsdata.ExecuteStatementOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) != nil {
		return args.Get(0).(*rdsdata.ExecuteStatementOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestStoreUser(t *testing.T) {
	mockClient := new(mockRDSClient)
	ctx := context.Background()

	user := models.CreateUserResponse{
		DID:    "did:example:123",
		Handle: "testuser",
	}

	event := models.UserRequest{
		Email: "test@example.com",
	}

	tests := []struct {
		name        string
		mockOutput  *rdsdata.ExecuteStatementOutput
		mockError   error
		expectedErr string
	}{
		{
			name:        "Successful User Storage",
			mockOutput:  &rdsdata.ExecuteStatementOutput{},
			mockError:   nil,
			expectedErr: "",
		},
		{
			name:        "Database Error",
			mockOutput:  nil,
			mockError:   errors.New("DB connection failed"),
			expectedErr: "failed to store user in PostgreSQL: DB connection failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockClient.ExpectedCalls = nil

			db := &PostgresDB{
				Client:       mockClient,
				DBClusterARN: "test-cluster",
				SecretARN:    "test-secret",
				DatabaseName: "test-db",
			}

			if test.mockError != nil {
				mockClient.On("ExecuteStatement", mock.Anything, mock.Anything).
					Return((*rdsdata.ExecuteStatementOutput)(nil), test.mockError)
			} else {
				mockClient.On("ExecuteStatement", mock.Anything, mock.Anything).
					Return(test.mockOutput, nil)
			}

			err := db.StoreUser(ctx, user, event)

			if test.expectedErr != "" {
				assert.Error(t, err, "Expected an error but got nil")
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}


func TestCheckEmailExists(t *testing.T) {
	mockClient := new(mockRDSClient)
	ctx := context.Background()

	tests := []struct {
		name         string
		mockOutput   *rdsdata.ExecuteStatementOutput
		mockError    error
		expectedBool bool
		expectedErr  string
	}{
		{
			name: "Email Exists",
			mockOutput: &rdsdata.ExecuteStatementOutput{
				Records: [][]types.Field{
					{&types.FieldMemberStringValue{Value: "1"}},
				},
			},
			mockError:    nil,
			expectedBool: true,
			expectedErr:  "",
		},
		{
			name: "Email Does Not Exist",
			mockOutput: &rdsdata.ExecuteStatementOutput{
				Records: [][]types.Field{},
			},
			mockError:    nil,
			expectedBool: false,
			expectedErr:  "",
		},
		{
			name:         "Database Connection Failure",
			mockOutput:   nil,
			mockError:    errors.New("DB connection failed"),
			expectedBool: false,
			expectedErr:  "failed to check email existence: DB connection failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockClient.ExpectedCalls = nil

			db := &PostgresDB{
				Client:       mockClient,
				DBClusterARN: "test-cluster",
				SecretARN:    "test-secret",
				DatabaseName: "test-db",
			}

			mockClient.On("ExecuteStatement", mock.Anything, mock.Anything).
				Return(test.mockOutput, test.mockError)

			result, err := db.CheckEmailExists(ctx, "test@example.com")

			assert.Equal(t, test.expectedBool, result)

			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

