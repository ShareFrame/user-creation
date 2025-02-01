package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
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
