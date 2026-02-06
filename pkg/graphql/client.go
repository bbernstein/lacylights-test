// Package graphql provides a GraphQL HTTP client for testing.
package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Note: Node server comparison functions have been removed as lacylights-node is deprecated.

const (
	// TestDeviceFingerprint is a well-known fingerprint for integration tests.
	// When running integration tests, the backend should either:
	// 1. Have AUTH_ENABLED=false (no auth required)
	// 2. Have this device pre-approved in the database
	// 3. Auto-approve devices with this fingerprint in test mode
	TestDeviceFingerprint = "test-device-fingerprint-12345"

	// DeviceFingerprintHeader is the HTTP header name for device authentication.
	DeviceFingerprintHeader = "X-Device-Fingerprint"
)

// Client is a GraphQL HTTP client for testing.
type Client struct {
	endpoint    string
	fingerprint string
	httpClient  *http.Client
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithFingerprint sets the device fingerprint for authentication.
// The fingerprint is sent as the X-Device-Fingerprint header with each request.
func WithFingerprint(fingerprint string) ClientOption {
	return func(c *Client) {
		c.fingerprint = fingerprint
	}
}

// WithTestDevice configures the client to use the standard test device fingerprint.
// This is a convenience wrapper around WithFingerprint(TestDeviceFingerprint).
//
// When running tests, ensure the backend is configured to accept this device:
// - Set AUTH_ENABLED=false (simplest, no auth required)
// - Or pre-approve the test device in the database
// - Or configure the backend to auto-approve test devices
func WithTestDevice() ClientOption {
	return WithFingerprint(TestDeviceFingerprint)
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new GraphQL client with optional configuration.
// If no endpoint is provided, it uses GRAPHQL_ENDPOINT env var or defaults
// to http://localhost:4001/graphql.
//
// Example usage:
//
//	// Basic client (no auth)
//	client := graphql.NewClient("")
//
//	// Client with device fingerprint
//	client := graphql.NewClient("", graphql.WithFingerprint("my-device-id"))
//
//	// Client with test device (for integration tests)
//	client := graphql.NewClient("", graphql.WithTestDevice())
func NewClient(endpoint string, opts ...ClientOption) *Client {
	if endpoint == "" {
		endpoint = os.Getenv("GRAPHQL_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "http://localhost:4001/graphql"
	}

	c := &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewTestClient creates a GraphQL client configured for integration tests.
// It uses the test device fingerprint for authentication.
//
// This is equivalent to:
//
//	NewClient(endpoint, WithTestDevice())
func NewTestClient(endpoint string) *Client {
	return NewClient(endpoint, WithTestDevice())
}


// Request represents a GraphQL request.
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// Response represents a GraphQL response.
type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Query executes a GraphQL query and unmarshals the response.
func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	resp, err := c.Execute(ctx, query, variables)
	if err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		return fmt.Errorf("graphql errors: %v", resp.Errors)
	}

	if result != nil {
		if err := json.Unmarshal(resp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Mutate executes a GraphQL mutation and unmarshals the response.
func (c *Client) Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error {
	return c.Query(ctx, mutation, variables, result)
}

// Execute executes a GraphQL request and returns the raw response.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]interface{}) (*Response, error) {
	req := Request{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add device fingerprint header if configured
	if c.fingerprint != "" {
		httpReq.Header.Set(DeviceFingerprintHeader, c.fingerprint)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", httpResp.StatusCode, string(respBody))
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ExecuteRaw executes a GraphQL request and returns the raw JSON response.
func (c *Client) ExecuteRaw(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	resp, err := c.Execute(ctx, query, variables)
	if err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql errors: %v", resp.Errors)
	}

	return resp.Data, nil
}

// Endpoint returns the client's endpoint URL.
func (c *Client) Endpoint() string {
	return c.endpoint
}

// Fingerprint returns the client's device fingerprint.
// Returns empty string if no fingerprint is configured.
//
// Note: Client is not safe for concurrent use. If a Client instance is shared
// between multiple goroutines, callers must ensure that all accesses to it,
// including calls to Fingerprint and SetFingerprint, are properly synchronized.
func (c *Client) Fingerprint() string {
	return c.fingerprint
}

// SetFingerprint sets the device fingerprint for subsequent requests.
// Pass empty string to clear the fingerprint.
//
// This method is not safe for concurrent use without external synchronization.
// If a Client is shared between goroutines, protect calls to this method with
// appropriate locking.
func (c *Client) SetFingerprint(fingerprint string) {
	c.fingerprint = fingerprint
}

// CompareResponses compares two JSON responses for equality.
// Returns true if equal, false otherwise with a description of differences.
func CompareResponses(a, b json.RawMessage) (bool, string) {
	var aData, bData interface{}

	if err := json.Unmarshal(a, &aData); err != nil {
		return false, fmt.Sprintf("failed to unmarshal response A: %v", err)
	}

	if err := json.Unmarshal(b, &bData); err != nil {
		return false, fmt.Sprintf("failed to unmarshal response B: %v", err)
	}

	return compareValues(aData, bData, "")
}

func compareValues(a, b interface{}, path string) (bool, string) {
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok {
			return false, fmt.Sprintf("type mismatch at %s: map vs %T", path, b)
		}
		return compareMaps(aVal, bVal, path)

	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return false, fmt.Sprintf("type mismatch at %s: array vs %T", path, b)
		}
		return compareArrays(aVal, bVal, path)

	default:
		if a != b {
			return false, fmt.Sprintf("value mismatch at %s: %v vs %v", path, a, b)
		}
		return true, ""
	}
}

func compareMaps(a, b map[string]interface{}, path string) (bool, string) {
	// Check all keys in a
	for key, aVal := range a {
		bVal, ok := b[key]
		if !ok {
			return false, fmt.Sprintf("missing key at %s.%s", path, key)
		}
		newPath := path + "." + key
		if path == "" {
			newPath = key
		}
		if equal, diff := compareValues(aVal, bVal, newPath); !equal {
			return false, diff
		}
	}

	// Check for extra keys in b
	for key := range b {
		if _, ok := a[key]; !ok {
			return false, fmt.Sprintf("extra key at %s.%s", path, key)
		}
	}

	return true, ""
}

func compareArrays(a, b []interface{}, path string) (bool, string) {
	if len(a) != len(b) {
		return false, fmt.Sprintf("array length mismatch at %s: %d vs %d", path, len(a), len(b))
	}

	for i := range a {
		newPath := fmt.Sprintf("%s[%d]", path, i)
		if equal, diff := compareValues(a[i], b[i], newPath); !equal {
			return false, diff
		}
	}

	return true, ""
}

