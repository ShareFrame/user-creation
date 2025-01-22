package atproto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Atlas-Mesh/user-management/internal/models"
	"net/http"
	"time"
)

const (
	CreateInviteCodeEndpoint = "/xrpc/com.atproto.server.createInviteCode"
	RegisterUserEndpoint     = "/xrpc/com.atproto.server.createAccount"
	useCount                 = 1
	timeout                  = 15 * time.Second
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ATProtocolClient struct {
	BaseURL    string
	HTTPClient HTTPClient
}

func NewATProtocolClient(baseURL string, client HTTPClient) *ATProtocolClient {
	return &ATProtocolClient{
		BaseURL:    baseURL,
		HTTPClient: client,
	}
}

func (c *ATProtocolClient) CreateInviteCode(adminCreds models.AdminCreds) (*models.InviteCodeResponse, error) {
	data := map[string]int{"useCount": useCount}
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(
		adminCreds.PDSAdminUsername + ":" + adminCreds.PDSAdminPassword))
	headers := map[string]string{
		"Authorization": "Basic " + auth,
		"Content-Type":  "application/json",
	}

	resp, err := c.doPost(CreateInviteCodeEndpoint, body, headers)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var inviteCodeResp models.InviteCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&inviteCodeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &inviteCodeResp, nil
}

func (c *ATProtocolClient) RegisterUser(handle, email, inviteCode string) (models.CreateUserResponse, error) {
	if handle == "" || email == "" || inviteCode == "" {
		return models.CreateUserResponse{}, fmt.Errorf("handle, email, and inviteCode are required")
	}

	data := map[string]string{
		"handle":     handle,
		"email":      email,
		"inviteCode": inviteCode,
	}
	body, err := json.Marshal(data)
	if err != nil {
		return models.CreateUserResponse{}, fmt.Errorf("failed to marshal body: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := c.doPost(RegisterUserEndpoint, body, headers)
	if err != nil {
		return models.CreateUserResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.CreateUserResponse{}, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	var registerResp models.CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return models.CreateUserResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return registerResp, nil
}

func (c *ATProtocolClient) doPost(endpoint string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.HTTPClient.Do(req)
}
