// Package preview provides preview session contract tests.
package preview

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProjectID is created and deleted for each test that needs it
func createTestProject(t *testing.T, client *graphql.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var resp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}

	err := client.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name": "Preview Test Project",
		},
	}, &resp)

	require.NoError(t, err)
	return resp.CreateProject.ID
}

func deleteTestProject(t *testing.T, client *graphql.Client, projectID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = client.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{
		"id": projectID,
	}, nil)
}

func TestStartPreviewSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a test project
	projectID := createTestProject(t, client)
	defer deleteTestProject(t, client, projectID)

	// Start preview session
	var startResp struct {
		StartPreviewSession struct {
			ID        string `json:"id"`
			ProjectID string `json:"projectId"`
			IsActive  bool   `json:"isActive"`
		} `json:"startPreviewSession"`
	}

	err := client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) {
				id
				projectId
				isActive
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &startResp)

	require.NoError(t, err)
	assert.NotEmpty(t, startResp.StartPreviewSession.ID)
	assert.Equal(t, projectID, startResp.StartPreviewSession.ProjectID)
	assert.True(t, startResp.StartPreviewSession.IsActive)

	// Clean up - cancel the session
	sessionID := startResp.StartPreviewSession.ID
	var cancelResp struct {
		CancelPreviewSession bool `json:"cancelPreviewSession"`
	}

	err = client.Mutate(ctx, `
		mutation CancelPreview($sessionId: ID!) {
			cancelPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{
		"sessionId": sessionID,
	}, &cancelResp)

	require.NoError(t, err)
	assert.True(t, cancelResp.CancelPreviewSession)
}

func TestPreviewChannelOverride(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a test project with a fixture
	projectID := createTestProject(t, client)
	defer deleteTestProject(t, client, projectID)

	// Create a fixture
	var fixtureResp struct {
		CreateFixture struct {
			ID           string `json:"id"`
			StartChannel int    `json:"startChannel"`
		} `json:"createFixture"`
	}

	err := client.Mutate(ctx, `
		mutation CreateFixture($input: CreateFixtureInput!) {
			createFixture(input: $input) {
				id
				startChannel
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"name":         "Test Fixture",
			"manufacturer": "Generic",
			"model":        "RGB Par",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixture.ID

	// Start preview session
	var startResp struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err = client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) {
				id
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &startResp)

	require.NoError(t, err)
	sessionID := startResp.StartPreviewSession.ID

	// Update channel in preview
	var updateResp struct {
		UpdatePreviewChannel bool `json:"updatePreviewChannel"`
	}

	err = client.Mutate(ctx, `
		mutation UpdatePreview($sessionId: ID!, $fixtureId: ID!, $channelIndex: Int!, $value: Int!) {
			updatePreviewChannel(sessionId: $sessionId, fixtureId: $fixtureId, channelIndex: $channelIndex, value: $value)
		}
	`, map[string]interface{}{
		"sessionId":    sessionID,
		"fixtureId":    fixtureID,
		"channelIndex": 0,
		"value":        200,
	}, &updateResp)

	require.NoError(t, err)
	assert.True(t, updateResp.UpdatePreviewChannel)

	// Clean up
	_ = client.Mutate(ctx, `
		mutation CancelPreview($sessionId: ID!) {
			cancelPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{
		"sessionId": sessionID,
	}, nil)
}

func TestPreviewSessionCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a test project
	projectID := createTestProject(t, client)
	defer deleteTestProject(t, client, projectID)

	// Start preview session
	var startResp struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err := client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) {
				id
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &startResp)

	require.NoError(t, err)
	sessionID := startResp.StartPreviewSession.ID

	// Commit the session
	var commitResp struct {
		CommitPreviewSession bool `json:"commitPreviewSession"`
	}

	err = client.Mutate(ctx, `
		mutation CommitPreview($sessionId: ID!) {
			commitPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{
		"sessionId": sessionID,
	}, &commitResp)

	require.NoError(t, err)
	assert.True(t, commitResp.CommitPreviewSession)
}

func TestStartingNewSessionCancelsPrevious(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a test project
	projectID := createTestProject(t, client)
	defer deleteTestProject(t, client, projectID)

	// Start first session
	var startResp1 struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err := client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) {
				id
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &startResp1)

	require.NoError(t, err)
	session1ID := startResp1.StartPreviewSession.ID

	// Start second session (should cancel first)
	var startResp2 struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err = client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) {
				id
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &startResp2)

	require.NoError(t, err)
	session2ID := startResp2.StartPreviewSession.ID

	// Sessions should be different
	assert.NotEqual(t, session1ID, session2ID)

	// Clean up
	_ = client.Mutate(ctx, `
		mutation CancelPreview($sessionId: ID!) {
			cancelPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{
		"sessionId": session2ID,
	}, nil)
}
