// Package integration provides migration integration tests.
package integration

import "os"

const (
	defaultNodeServerURL = "http://localhost:4000/graphql"
	defaultGoServerURL   = "http://localhost:4001/graphql"
)

// getNodeServerURL returns the Node server URL from environment or default
func getNodeServerURL() string {
	url := os.Getenv("NODE_SERVER_URL")
	if url == "" {
		url = defaultNodeServerURL
	}
	return url
}

// getGoServerURL returns the Go server URL from environment or default
func getGoServerURL() string {
	url := os.Getenv("GO_SERVER_URL")
	if url == "" {
		url = defaultGoServerURL
	}
	return url
}
