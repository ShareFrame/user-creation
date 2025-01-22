package atproto

import (
	"bytes"
	"errors"
	"github.com/Atlas-Mesh/user-management/internal/models"
	"io/ioutil"
	"net/http"
	"testing"
)

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestCreateInviteCode(t *testing.T) {
	tests := []struct {
		name           string
		httpResponse   *http.Response
		httpError      error
		expectedOutput *models.InviteCodeResponse
		expectedError  string
		adminCreds     models.AdminCreds
	}{
		{
			name: "Successful Invite Code Creation",
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"Code": "invite123"}`))),
			},
			httpError: nil,
			expectedOutput: &models.InviteCodeResponse{
				Code: "invite123",
			},
			expectedError: "",
			adminCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "password",
			},
		},
		{
			name:           "HTTP Error",
			httpResponse:   nil,
			httpError:      errors.New("HTTP request failed"),
			expectedOutput: nil,
			expectedError:  "request failed: HTTP request failed",
			adminCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "password",
			},
		},
		{
			name: "Non-200 Status Code",
			httpResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`Internal Server Error`))),
			},
			httpError:      nil,
			expectedOutput: nil,
			expectedError:  "unexpected status code: 500",
			adminCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "password",
			},
		},
		{
			name: "Invalid JSON Response",
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`invalid json`))),
			},
			httpError:      nil,
			expectedOutput: nil,
			expectedError:  "failed to decode response: invalid character 'i' looking for beginning of value",
			adminCreds: models.AdminCreds{
				PDSAdminUsername: "admin",
				PDSAdminPassword: "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return tt.httpResponse, tt.httpError
				},
			}

			client := &ATProtocolClient{
				BaseURL:    "https://example.com",
				HTTPClient: mockClient,
			}

			result, err := client.CreateInviteCode(tt.adminCreds)

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
				if result.Code != tt.expectedOutput.Code {
					t.Errorf("Expected invite code %q, got %q", tt.expectedOutput.Code, result.Code)
				}
			}
		})
	}
}

func TestRegisterUser(t *testing.T) {
	tests := []struct {
		name           string
		httpResponse   *http.Response
		httpError      error
		expectedOutput models.CreateUserResponse
		expectedError  string
		handle         string
		email          string
		inviteCode     string
	}{
		{
			name: "Successful User Registration",
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"DID": "did:example:123", "Handle": "user123", "Email": "user@example.com"}`))),
			},
			httpError: nil,
			expectedOutput: models.CreateUserResponse{
				DID:    "did:example:123",
				Handle: "user123",
				Email:  "user@example.com",
			},
			expectedError: "",
			handle:        "user123",
			email:         "user@example.com",
			inviteCode:    "invite123",
		},
		{
			name:           "HTTP Error",
			httpResponse:   nil,
			httpError:      errors.New("HTTP request failed"),
			expectedOutput: models.CreateUserResponse{},
			expectedError:  "request failed: HTTP request failed",
			handle:         "user123",
			email:          "user@example.com",
			inviteCode:     "invite123",
		},
		{
			name: "Invalid JSON Response",
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`invalid json`))),
			},
			httpError:      nil,
			expectedOutput: models.CreateUserResponse{},
			expectedError:  "failed to decode response: invalid character 'i' looking for beginning of value",
			handle:         "user123",
			email:          "user@example.com",
			inviteCode:     "invite123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return tt.httpResponse, tt.httpError
				},
			}

			client := &ATProtocolClient{
				BaseURL:    "https://example.com",
				HTTPClient: mockClient,
			}

			result, err := client.RegisterUser(tt.handle, tt.email, tt.inviteCode)

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
				if result.DID != tt.expectedOutput.DID {
					t.Errorf("Expected DID %q, got %q", tt.expectedOutput.DID, result.DID)
				}
				if result.Handle != tt.expectedOutput.Handle {
					t.Errorf("Expected Handle %q, got %q", tt.expectedOutput.Handle, result.Handle)
				}
				if result.Email != tt.expectedOutput.Email {
					t.Errorf("Expected Email %q, got %q", tt.expectedOutput.Email, result.Email)
				}
			}
		})
	}
}
