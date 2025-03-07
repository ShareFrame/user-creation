package helper

import (
	"context"
	"testing"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/ShareFrame/user-management/internal/postgres"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockPostgresClient struct {
	mock.Mock
}

var _ postgres.PostgresDBService = (*mockPostgresClient)(nil)

func (m *mockPostgresClient) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *mockPostgresClient) StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error {
	args := m.Called(ctx, user, event)
	return args.Error(0)
}

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

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectedErr string
	}{
		{"Valid Password", "Strong@123", ""},
		{"Too Short", "Short1!", PasswordError},
		{"No Uppercase", "weakpassword1!", PasswordError},
		{"No Lowercase", "WEAKPASSWORD1!", PasswordError},
		{"No Digit", "NoDigits!!", PasswordError},
		{"No Special Character", "NoSpecial1", PasswordError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePassword(test.password)
			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Equal(t, test.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnsureHandleSuffix(t *testing.T) {
	tests := []struct {
		name     string
		handle   string
		expected string
	}{
		{"Already Has Suffix", "username.shareframe.social", "username.shareframe.social"},
		{"Missing Suffix", "username", "username.shareframe.social"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := EnsureHandleSuffix(test.handle)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestValidateHandle(t *testing.T) {
	tests := []struct {
		name        string
		handle      string
		expectedErr string
	}{
		{"Valid Handle", "validuser", ""},
		{"Too Short", "ab", HandleTooShort},
		{"Too Long", "thisisaverylonghandle", HandleTooLong},
		{"Contains Special Characters", "invalid@handle", InvalidHandle},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateHandle(test.handle)
			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectedErr string
	}{
		{"Valid Email", "user@example.com", ""},
		{"Missing @", "userexample.com", "invalid email format"},
		{"Missing domain", "user@", "invalid email format"},
		{"Invalid TLD", "user@example.c", "invalid email format"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateEmail(test.email)
			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAndFormatUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		user             models.UserRequest
		mockEmailExists  bool
		mockEmailErr     error
		expectCheckEmail bool
		expectedErr      string
	}{
		{"Valid User", models.UserRequest{Handle: "validuser", Email: "user@example.com", Password: "Valid@123"}, false, nil, true, ""},
		{"Missing Handle", models.UserRequest{Handle: "", Email: "user@example.com", Password: "Valid@123"}, false, nil, false, MissingFields},
		{"Missing Email", models.UserRequest{Handle: "validuser", Email: "", Password: "Valid@123"}, false, nil, false, MissingFields},
		{"Missing Password", models.UserRequest{Handle: "validuser", Email: "user@example.com", Password: ""}, false, nil, false, MissingFields},
		{"Invalid Handle", models.UserRequest{Handle: "inv@lid", Email: "user@example.com", Password: "Valid@123"}, false, nil, false, InvalidHandle},
		{"Invalid Email", models.UserRequest{Handle: "validuser", Email: "invalid-email", Password: "Valid@123"}, false, nil, false, "invalid email format"},
		{"Invalid Password", models.UserRequest{Handle: "validuser", Email: "user@example.com", Password: "weak"}, false, nil, false, PasswordError},
		{"Email Already Exists", models.UserRequest{Handle: "validuser", Email: "user@example.com", Password: "Valid@123"}, true, nil, true, EmailTaken},
		{"DB Check Failure", models.UserRequest{Handle: "validuser", Email: "user@example.com", Password: "Valid@123"}, false, assert.AnError, true, "internal error: failed to check email"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockDB := new(mockPostgresClient)

			if test.expectCheckEmail {
				mockDB.On("CheckEmailExists", ctx, test.user.Email).Return(test.mockEmailExists, test.mockEmailErr)
			}

			_, err := ValidateAndFormatUser(ctx, test.user, mockDB)

			if test.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if test.expectCheckEmail {
				mockDB.AssertCalled(t, "CheckEmailExists", ctx, test.user.Email)
			} else {
				mockDB.AssertNotCalled(t, "CheckEmailExists", ctx, test.user.Email)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestRetrieveAdminCredentials(t *testing.T) {
	mockSecretsManager := new(mockSecretsManagerClient)
	ctx := context.Background()

	mockSecretValue := `{"PDS_JWT_SECRET":"jwtsecret","PDS_ADMIN_USERNAME":"admin","PDS_ADMIN_PASSWORD":"securepass"}`
	mockSecretsManager.On("GetSecretValue", ctx, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: &mockSecretValue,
	}, nil)

	creds, err := RetrieveAdminCredentials(ctx, mockSecretsManager)
	assert.NoError(t, err)
	assert.Equal(t, "jwtsecret", creds.PDSJWTSecret)
	assert.Equal(t, "admin", creds.PDSAdminUsername)
	assert.Equal(t, "securepass", creds.PDSAdminPassword)
}

func TestRetrieveUtilAccountCreds(t *testing.T) {
	mockSecretsManager := new(mockSecretsManagerClient)
	ctx := context.Background()

	mockSecretValue := `{"username":"util-user","password":"util-pass","did":"did:example:123"}`
	mockSecretsManager.On("GetSecretValue", ctx, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: &mockSecretValue,
	}, nil)

	creds, err := RetrieveUtilAccountCreds(ctx, mockSecretsManager)
	assert.NoError(t, err)
	assert.Equal(t, "util-user", creds.Username)
	assert.Equal(t, "util-pass", creds.Password)
	assert.Equal(t, "did:example:123", creds.DID)
}
