// Package api provides API contract tests for the LacyLights GraphQL servers.
package api

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemInfoQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		SystemInfo struct {
			ArtnetEnabled          bool    `json:"artnetEnabled"`
			ArtnetBroadcastAddress *string `json:"artnetBroadcastAddress"`
		} `json:"systemInfo"`
	}

	err := client.Query(ctx, `
		query {
			systemInfo {
				artnetEnabled
				artnetBroadcastAddress
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	// Just verify the query works - actual values may differ
	assert.NotNil(t, resp.SystemInfo)
}

func TestProjectsListQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		Projects []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"projects"`
	}

	err := client.Query(ctx, `
		query {
			projects {
				id
				name
				description
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotNil(t, resp.Projects)
}

func TestNetworkInterfaceOptionsQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		NetworkInterfaceOptions []struct {
			Name          string `json:"name"`
			Address       string `json:"address"`
			Broadcast     string `json:"broadcast"`
			Description   string `json:"description"`
			InterfaceType string `json:"interfaceType"`
		} `json:"networkInterfaceOptions"`
	}

	err := client.Query(ctx, `
		query {
			networkInterfaceOptions {
				name
				address
				broadcast
				description
				interfaceType
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotNil(t, resp.NetworkInterfaceOptions)

	// Verify we have at least localhost and global broadcast
	hasLocalhost := false
	hasGlobal := false
	for _, opt := range resp.NetworkInterfaceOptions {
		if opt.InterfaceType == "localhost" {
			hasLocalhost = true
		}
		if opt.InterfaceType == "global" {
			hasGlobal = true
		}
	}
	assert.True(t, hasLocalhost, "should have localhost option")
	assert.True(t, hasGlobal, "should have global broadcast option")
}

func TestDMXOutputQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// dmxOutput returns [Int!]! - a flat array of 512 channel values
	var resp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err := client.Query(ctx, `
		query {
			dmxOutput(universe: 0)
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.Len(t, resp.DMXOutput, 512, "DMX universe should have 512 channels")
}

func TestCreateAndDeleteProject(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a project
	var createResp struct {
		CreateProject struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"createProject"`
	}

	err := client.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Contract Test Project",
			"description": "Created by contract tests",
		},
	}, &createResp)

	require.NoError(t, err)
	assert.NotEmpty(t, createResp.CreateProject.ID)
	assert.Equal(t, "Contract Test Project", createResp.CreateProject.Name)

	projectID := createResp.CreateProject.ID

	// Clean up - delete the project
	var deleteResp struct {
		DeleteProject bool `json:"deleteProject"`
	}

	err = client.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &deleteResp)

	require.NoError(t, err)
	assert.True(t, deleteResp.DeleteProject)
}

func TestFixtureDefinitionsQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
			Type         string `json:"type"`
		} `json:"fixtureDefinitions"`
	}

	err := client.Query(ctx, `
		query {
			fixtureDefinitions {
				id
				manufacturer
				model
				type
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotNil(t, resp.FixtureDefinitions)
}
