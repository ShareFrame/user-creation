package atproto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ATProtocolClient struct {
	BaseURL string
}

func NewATProtocolClient(baseURL string) *ATProtocolClient {
	return &ATProtocolClient{BaseURL: baseURL}
}

func (c *ATProtocolClient) RegisterUser(handle, email string) error {
	data := map[string]string{"handle": handle, "email": email}
	body, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", c.BaseURL+"/xrpc/com.atproto.server.createAccount",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return fmt.Errorf("failed to create account: %s", resp.Status)
	}
	return nil
}
