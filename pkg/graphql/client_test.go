// Package graphql provides a GraphQL HTTP client for testing.
package graphql

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("default endpoint", func(t *testing.T) {
		client := NewClient("")
		assert.Equal(t, "http://localhost:4001/graphql", client.Endpoint())
		assert.Empty(t, client.Fingerprint())
	})

	t.Run("custom endpoint", func(t *testing.T) {
		client := NewClient("http://example.com/graphql")
		assert.Equal(t, "http://example.com/graphql", client.Endpoint())
	})

	t.Run("with fingerprint option", func(t *testing.T) {
		client := NewClient("", WithFingerprint("my-device-id"))
		assert.Equal(t, "my-device-id", client.Fingerprint())
	})

	t.Run("with test device option", func(t *testing.T) {
		client := NewClient("", WithTestDevice())
		assert.Equal(t, TestDeviceFingerprint, client.Fingerprint())
	})
}

func TestNewTestClient(t *testing.T) {
	client := NewTestClient("")
	assert.Equal(t, TestDeviceFingerprint, client.Fingerprint())
	assert.Equal(t, "http://localhost:4001/graphql", client.Endpoint())
}

func TestClientSetFingerprint(t *testing.T) {
	client := NewClient("")
	assert.Empty(t, client.Fingerprint())

	client.SetFingerprint("new-fingerprint")
	assert.Equal(t, "new-fingerprint", client.Fingerprint())

	client.SetFingerprint("")
	assert.Empty(t, client.Fingerprint())
}

func TestClientFingerprintHeader(t *testing.T) {
	var receivedFingerprint string
	var receivedContentType string

	// Create a test server that captures headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFingerprint = r.Header.Get(DeviceFingerprintHeader)
		receivedContentType = r.Header.Get("Content-Type")

		// Return a valid GraphQL response
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Data: json.RawMessage(`{"test": "value"}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Run("sends fingerprint header when configured", func(t *testing.T) {
		receivedFingerprint = ""

		client := NewClient(server.URL, WithFingerprint("test-fingerprint-123"))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)

		assert.Equal(t, "test-fingerprint-123", receivedFingerprint)
		assert.Equal(t, "application/json", receivedContentType)
	})

	t.Run("does not send fingerprint header when not configured", func(t *testing.T) {
		receivedFingerprint = ""

		client := NewClient(server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)

		assert.Empty(t, receivedFingerprint)
	})

	t.Run("test device sends correct fingerprint", func(t *testing.T) {
		receivedFingerprint = ""

		client := NewTestClient(server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)

		assert.Equal(t, TestDeviceFingerprint, receivedFingerprint)
	})

	t.Run("fingerprint can be changed after creation", func(t *testing.T) {
		receivedFingerprint = ""

		client := NewClient(server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// First request without fingerprint
		_, err := client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)
		assert.Empty(t, receivedFingerprint)

		// Set fingerprint and make another request
		client.SetFingerprint("dynamic-fingerprint")
		_, err = client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)
		assert.Equal(t, "dynamic-fingerprint", receivedFingerprint)

		// Clear fingerprint
		client.SetFingerprint("")
		_, err = client.Execute(ctx, "query { test }", nil)
		require.NoError(t, err)
		assert.Empty(t, receivedFingerprint)
	})
}

func TestClientWithCustomHTTPClient(t *testing.T) {
	var requestReceived bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Data: json.RawMessage(`{"test": "value"}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	customClient := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	client := NewClient(server.URL, WithHTTPClient(customClient))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Execute(ctx, "query { test }", nil)
	require.NoError(t, err)
	assert.True(t, requestReceived)
}

func TestTestDeviceFingerprintConstant(t *testing.T) {
	// Ensure the test device fingerprint has a recognizable format
	assert.Equal(t, "test-device-fingerprint-12345", TestDeviceFingerprint)
	assert.NotEmpty(t, TestDeviceFingerprint)
}

func TestDeviceFingerprintHeaderConstant(t *testing.T) {
	assert.Equal(t, "X-Device-Fingerprint", DeviceFingerprintHeader)
}

func TestQueryWithVariablesAndFingerprint(t *testing.T) {
	var receivedBody []byte
	var receivedFingerprint string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFingerprint = r.Header.Get(DeviceFingerprintHeader)
		receivedBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Data: json.RawMessage(`{"project": {"id": "123", "name": "Test"}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithTestDevice())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	}

	err := client.Query(ctx, `query GetProject($id: ID!) { project(id: $id) { id name } }`,
		map[string]interface{}{"id": "123"}, &result)

	require.NoError(t, err)
	assert.Equal(t, "123", result.Project.ID)
	assert.Equal(t, "Test", result.Project.Name)
	assert.Equal(t, TestDeviceFingerprint, receivedFingerprint)

	// Verify the request body contains our query and variables
	var req Request
	err = json.Unmarshal(receivedBody, &req)
	require.NoError(t, err)
	assert.Contains(t, req.Query, "GetProject")
	assert.Equal(t, "123", req.Variables["id"])
}
