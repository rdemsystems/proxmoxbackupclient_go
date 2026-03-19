package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultServiceURL = "http://127.0.0.1:18765"
	ConnectionTimeout = 5 * time.Second  // Increased for service startup
	RequestTimeout    = 30 * time.Second // Backup returns immediately, safe
)

// Client handles communication with the local service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		baseURL: DefaultServiceURL,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// IsServiceAvailable checks if the local service is running
func (c *Client) IsServiceAvailable() bool {
	client := &http.Client{
		Timeout: ConnectionTimeout,
	}

	resp, err := client.Get(c.baseURL + "/status")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetStatus retrieves the service status
func (c *Client) GetStatus() (*StatusResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned error: %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// StartBackup sends a backup request to the service
func (c *Client) StartBackup(req *BackupRequest) (*BackupResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/backup",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send backup request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, fmt.Errorf("backup failed: %s", errResp.Error)
		}
		return nil, fmt.Errorf("backup failed with status %d", resp.StatusCode)
	}

	var backupResp BackupResponse
	if err := json.Unmarshal(respBody, &backupResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &backupResp, nil
}

// GetJobs retrieves the list of configured jobs
func (c *Client) GetJobs() (*JobsResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/jobs")
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned error: %d", resp.StatusCode)
	}

	var jobs JobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &jobs, nil
}
