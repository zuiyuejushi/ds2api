package poolclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ds2api/internal/account"
	"ds2api/internal/config"
)

// Client implements account.PoolInterface by calling a remote pool server over HTTP.
type Client struct {
	serverURL string
	authKey   string
	http      *http.Client
}

// New creates a new pool client targeting the given server URL.
func New(serverURL, authKey string) *Client {
	return &Client{
		serverURL: serverURL,
		authKey:   authKey,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// AcquireWait acquires an account from the remote pool, blocking until one is available.
func (c *Client) AcquireWait(ctx context.Context, target string, exclude map[string]bool) (*account.PoolAccount, error) {
	excludeList := make([]string, 0, len(exclude))
	for k := range exclude {
		excludeList = append(excludeList, k)
	}
	body := map[string]any{
		"target":  target,
		"exclude": excludeList,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("poolclient: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/api/pool/acquire", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("poolclient: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poolclient: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poolclient: server returned %d: %s", resp.StatusCode, string(errBody))
	}
	var result account.PoolAccount
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("poolclient: decode response: %w", err)
	}
	return &result, nil
}

// Release releases an account back to the remote pool.
func (c *Client) Release(accountID string) {
	body, _ := json.Marshal(map[string]string{"accountID": accountID})
	req, _ := http.NewRequest(http.MethodPost, c.serverURL+"/api/pool/release", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Reset resets the remote pool.
func (c *Client) Reset() {
	req, _ := http.NewRequest(http.MethodPost, c.serverURL+"/api/pool/reset", nil)
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Stats returns usage statistics from the remote pool.
func (c *Client) Stats() account.PoolStats {
	req, _ := http.NewRequest(http.MethodGet, c.serverURL+"/api/pool/stats", nil)
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return account.PoolStats{}
	}
	defer resp.Body.Close()
	var stats account.PoolStats
	json.NewDecoder(resp.Body).Decode(&stats)
	return stats
}

// FetchConfig fetches the full server configuration.
func (c *Client) FetchConfig() (config.Config, error) {
	req, _ := http.NewRequest(http.MethodGet, c.serverURL+"/api/pool/config", nil)
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return config.Config{}, fmt.Errorf("poolclient fetch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return config.Config{}, fmt.Errorf("poolclient fetch config: server returned %d: %s", resp.StatusCode, string(errBody))
	}
	var result struct {
		Config config.Config `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return config.Config{}, fmt.Errorf("poolclient fetch config: decode error: %w", err)
	}
	return result.Config, nil
}
