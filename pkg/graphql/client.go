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

// Client is a GraphQL HTTP client for testing.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new GraphQL client.
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = os.Getenv("GRAPHQL_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "http://localhost:4001/graphql"
	}

	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

