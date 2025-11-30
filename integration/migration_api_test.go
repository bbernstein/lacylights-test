// Package integration provides GraphQL API comparison tests for migration.
package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGraphQLAPIComparison verifies both servers return identical responses
func TestGraphQLAPIComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
	}{
		{
			name: "SystemInfo Query",
			query: `
				query {
					systemInfo {
						artnetEnabled
						artnetBroadcastAddress
					}
				}
			`,
			variables: nil,
		},
		{
			name: "Projects List",
			query: `
				query {
					projects {
						id
						name
						description
					}
				}
			`,
			variables: nil,
		},
		{
			name: "Fixture Definitions",
			query: `
				query {
					fixtureDefinitions {
						id
						manufacturer
						model
						type
					}
				}
			`,
			variables: nil,
		},
		{
			name: "DMX Output",
			query: `
				query {
					dmxOutput(universe: 1)
				}
			`,
			variables: nil,
		},
		{
			name: "Network Interfaces",
			query: `
				query {
					networkInterfaceOptions {
						name
						address
						broadcast
						interfaceType
					}
				}
			`,
			variables: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute query on both servers
			nodeResp, err := nodeClient.ExecuteRaw(ctx, tt.query, tt.variables)
			require.NoError(t, err, "Node server query failed")

			goResp, err := goClient.ExecuteRaw(ctx, tt.query, tt.variables)
			require.NoError(t, err, "Go server query failed")

			// Compare responses
			equal, diff := graphql.CompareResponses(nodeResp, goResp)
			if !equal {
				t.Logf("Node response: %s", string(nodeResp))
				t.Logf("Go response: %s", string(goResp))
			}
			assert.True(t, equal, "Responses should be identical: %s", diff)
		})
	}
}

// TestMutationAPIComparison verifies mutations work identically
func TestMutationAPIComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Test project creation with Node
	var nodeCreateResp struct {
		CreateProject struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"createProject"`
	}

	testDesc := "API comparison test"
	err := nodeClient.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Node API Test",
			"description": testDesc,
		},
	}, &nodeCreateResp)

	require.NoError(t, err)
	nodeProjectID := nodeCreateResp.CreateProject.ID

	defer func() {
		var deleteResp struct {
			DeleteProject bool `json:"deleteProject"`
		}
		_ = nodeClient.Mutate(context.Background(), `
			mutation DeleteProject($id: ID!) {
				deleteProject(id: $id)
			}
		`, map[string]interface{}{
			"id": nodeProjectID,
		}, &deleteResp)
	}()

	// Test project creation with Go
	var goCreateResp struct {
		CreateProject struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"createProject"`
	}

	err = goClient.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Go API Test",
			"description": testDesc,
		},
	}, &goCreateResp)

	require.NoError(t, err)
	goProjectID := goCreateResp.CreateProject.ID

	defer func() {
		var deleteResp struct {
			DeleteProject bool `json:"deleteProject"`
		}
		_ = goClient.Mutate(context.Background(), `
			mutation DeleteProject($id: ID!) {
				deleteProject(id: $id)
			}
		`, map[string]interface{}{
			"id": goProjectID,
		}, &deleteResp)
	}()

	// Compare structure (IDs will differ, but structure should match)
	assert.NotEmpty(t, nodeCreateResp.CreateProject.ID)
	assert.NotEmpty(t, goCreateResp.CreateProject.ID)
	assert.Equal(t, "Node API Test", nodeCreateResp.CreateProject.Name)
	assert.Equal(t, "Go API Test", goCreateResp.CreateProject.Name)
	assert.NotNil(t, nodeCreateResp.CreateProject.Description)
	assert.NotNil(t, goCreateResp.CreateProject.Description)

	if nodeCreateResp.CreateProject.Description != nil && goCreateResp.CreateProject.Description != nil {
		assert.Equal(t, *nodeCreateResp.CreateProject.Description, *goCreateResp.CreateProject.Description)
	}
}

// TestErrorHandlingComparison verifies error responses are consistent
func TestErrorHandlingComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	tests := []struct {
		name      string
		query     string
		variables map[string]interface{}
	}{
		{
			name: "Query non-existent project",
			query: `
				query GetProject($id: ID!) {
					project(id: $id) {
						id
						name
					}
				}
			`,
			variables: map[string]interface{}{
				"id": "non-existent-project-id",
			},
		},
		{
			name: "Invalid universe number",
			query: `
				query {
					dmxOutput(universe: 999)
				}
			`,
			variables: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute query on both servers (expecting errors)
			nodeResp, nodeErr := nodeClient.Execute(ctx, tt.query, tt.variables)
			goResp, goErr := goClient.Execute(ctx, tt.query, tt.variables)

			// Both should handle errors similarly
			// Either both succeed with null data, or both return errors
			if nodeErr != nil && goErr != nil {
				// Both returned errors - this is acceptable
				t.Logf("Both servers returned errors (expected): Node=%v, Go=%v", nodeErr, goErr)
				return
			}

			if nodeErr == nil && goErr == nil {
				// Both succeeded - check if they have errors in GraphQL response
				nodeHasErrors := len(nodeResp.Errors) > 0
				goHasErrors := len(goResp.Errors) > 0

				assert.Equal(t, nodeHasErrors, goHasErrors,
					"Both servers should handle errors consistently")

				if nodeHasErrors && goHasErrors {
					t.Logf("Both servers returned GraphQL errors (expected)")
				}
				return
			}

			// One succeeded and one failed - this is inconsistent
			t.Errorf("Inconsistent error handling: Node error=%v, Go error=%v", nodeErr, goErr)
		})
	}
}

// TestConcurrentRequestsComparison verifies both servers handle concurrent requests
func TestConcurrentRequestsComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Test concurrent queries
	numRequests := 10
	query := `
		query {
			projects {
				id
				name
			}
		}
	`

	// Run concurrent requests on Node server
	nodeResults := make([]json.RawMessage, numRequests)
	nodeErrors := make([]error, numRequests)
	nodeDone := make(chan bool)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			nodeResults[idx], nodeErrors[idx] = nodeClient.ExecuteRaw(ctx, query, nil)
			nodeDone <- true
		}(i)
	}

	// Run concurrent requests on Go server
	goResults := make([]json.RawMessage, numRequests)
	goErrors := make([]error, numRequests)
	goDone := make(chan bool)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			goResults[idx], goErrors[idx] = goClient.ExecuteRaw(ctx, query, nil)
			goDone <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-nodeDone
		<-goDone
	}

	// Verify all requests succeeded
	nodeSuccesses := 0
	goSuccesses := 0

	for i := 0; i < numRequests; i++ {
		if nodeErrors[i] == nil {
			nodeSuccesses++
		}
		if goErrors[i] == nil {
			goSuccesses++
		}
	}

	assert.Equal(t, numRequests, nodeSuccesses, "All Node requests should succeed")
	assert.Equal(t, numRequests, goSuccesses, "All Go requests should succeed")

	// Verify responses are consistent
	if nodeSuccesses > 0 && goSuccesses > 0 {
		equal, diff := graphql.CompareResponses(nodeResults[0], goResults[0])
		assert.True(t, equal, "Concurrent responses should be identical: %s", diff)
	}
}

// TestSubscriptionAPIComparison verifies WebSocket subscription endpoints
func TestSubscriptionAPIComparison(t *testing.T) {
	// This test would require WebSocket client implementation
	// For now, we'll test that both servers expose the subscription endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Test introspection to verify subscription types exist
	query := `
		query {
			__schema {
				subscriptionType {
					name
					fields {
						name
					}
				}
			}
		}
	`

	var nodeResp struct {
		Schema struct {
			SubscriptionType *struct {
				Name   string `json:"name"`
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"subscriptionType"`
		} `json:"__schema"`
	}

	var goResp struct {
		Schema struct {
			SubscriptionType *struct {
				Name   string `json:"name"`
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"subscriptionType"`
		} `json:"__schema"`
	}

	err := nodeClient.Query(ctx, query, nil, &nodeResp)
	require.NoError(t, err)

	err = goClient.Query(ctx, query, nil, &goResp)
	require.NoError(t, err)

	// Verify both have subscription types
	assert.NotNil(t, nodeResp.Schema.SubscriptionType, "Node should have subscription type")
	assert.NotNil(t, goResp.Schema.SubscriptionType, "Go should have subscription type")

	if nodeResp.Schema.SubscriptionType != nil && goResp.Schema.SubscriptionType != nil {
		// Compare subscription fields
		assert.Equal(t, len(nodeResp.Schema.SubscriptionType.Fields),
			len(goResp.Schema.SubscriptionType.Fields),
			"Both servers should have same number of subscription fields")
	}
}

// TestSchemaIntrospectionComparison verifies GraphQL schemas are identical
func TestSchemaIntrospectionComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Full introspection query
	query := `
		query IntrospectionQuery {
			__schema {
				queryType { name }
				mutationType { name }
				subscriptionType { name }
				types {
					kind
					name
					description
				}
			}
		}
	`

	nodeResp, err := nodeClient.ExecuteRaw(ctx, query, nil)
	require.NoError(t, err)

	goResp, err := goClient.ExecuteRaw(ctx, query, nil)
	require.NoError(t, err)

	// Compare schema introspection results
	equal, diff := graphql.CompareResponses(nodeResp, goResp)
	if !equal {
		t.Logf("Schema differences found: %s", diff)
		// Note: Some differences in built-in types or ordering might be acceptable
		// Log the difference but don't fail the test if it's just ordering
		t.Logf("Node schema: %s", string(nodeResp))
		t.Logf("Go schema: %s", string(goResp))
	}
	assert.True(t, equal, "GraphQL schemas should be identical: %s", diff)
}
