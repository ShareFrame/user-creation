package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/ShareFrame/user-management/internal/helper/mocks"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndFormatUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabaseService(ctrl)

	tests := []struct {
		name        string
		input       models.UserRequest
		mockSetup   func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid User Input",
			input: models.UserRequest{
				Handle:   "validuser",
				Email:    "valid@example.com",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), "valid@example.com").Return(false, nil).Times(1)
			},
			expectError: false,
		},
		{
			name: "Database Error",
			input: models.UserRequest{
				Handle:   "validuser",
				Email:    "valid@example.com",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), "valid@example.com").Return(false, errors.New("DB connection failed")).Times(1)
			},
			expectError: true,
			errorMsg:    "internal error: failed to check email",
		},
		{
			name: "Blocked Username",
			input: models.UserRequest{
				Handle:   "admin",
				Email:    "valid@example.com",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			expectError: true,
			errorMsg:    BlockedHandle,
		},
		{
			name: "Handle Too Short",
			input: models.UserRequest{
				Handle:   "a",
				Email:    "valid@example.com",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			expectError: true,
			errorMsg:    HandleTooShort,
		},
		{
			name: "Empty Email But Valid Handle & Password",
			input: models.UserRequest{
				Handle:   "validuser",
				Email:    "",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), gomock.Any()).Times(0)
			},
			expectError: true,
			errorMsg:    "handle, email, and password are required fields",
		},
		{
			name: "Database Unexpected Error",
			input: models.UserRequest{
				Handle:   "validuser",
				Email:    "unexpected@example.com",
				Password: "Valid@123",
			},
			mockSetup: func() {
				mockDB.EXPECT().CheckEmailExists(gomock.Any(), "unexpected@example.com").Return(false, errors.New("some unexpected DB error")).Times(1)
			},
			expectError: true,
			errorMsg:    "internal error: failed to check email",
		},
		
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			_, err := ValidateAndFormatUser(context.TODO(), tt.input, mockDB)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHandle(t *testing.T) {
	tests := []struct {
		name        string
		handle      string
		expectError bool
		errorMsg    string
	}{
		{"Valid Handle", "validuser", false, ""},
		{"Too Short", "a", true, HandleTooShort},
		{"Too Long", "averylongusernamethatexceeds18", true, HandleTooLong},
		{"Invalid Characters", "invalid user!", true, InvalidHandle},
		{"Blocked Handle", "admin", true, BlockedHandle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHandle(tt.handle)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnsureHandleSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Already has suffix", "user" + PDS_Suffix, "user" + PDS_Suffix},
		{"Missing suffix", "user", "user" + PDS_Suffix},
		{"Handle has trailing space", "user ", "user" + PDS_Suffix},
		{"Handle has uppercase", "UserName", "UserName" + PDS_Suffix},
		{"Handle has both leading and trailing space", " user ", "user" + PDS_Suffix},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureHandleSuffix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{"Valid Email", "test@example.com", false},
		{"Invalid Email", "invalid-email", true},
		{"Missing Domain", "user@", true},
		{"No Username", "@example.com", true},
		{"Email with subdomain", "user@mail.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{"Valid Password", "Password1@", false},
		{"Too Short", "Pw1!", true},
		{"No Uppercase", "password1@", true},
		{"No Lowercase", "PASSWORD1@", true},
		{"No Digit", "Password!@", true},
		{"No Special Char", "Password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
