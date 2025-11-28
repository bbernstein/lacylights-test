// Package fade provides fade behavior contract tests.
package fade

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/artnet"
	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getArtNetPort() string {
	port := os.Getenv("ARTNET_LISTEN_PORT")
	if port == "" {
		port = "6454"
	}
	return ":" + port
}

// createTestProjectWithScene creates a project with a scene for testing fades.
func createTestProjectWithScene(t *testing.T, client *graphql.Client) (projectID, sceneID, fixtureID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create project
	var projectResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}

	err := client.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"name": "Fade Test Project"},
	}, &projectResp)
	require.NoError(t, err)
	projectID = projectResp.CreateProject.ID

	// Create fixture
	var fixtureResp struct {
		CreateFixture struct {
			ID string `json:"id"`
		} `json:"createFixture"`
	}

	err = client.Mutate(ctx, `
		mutation CreateFixture($input: CreateFixtureInput!) {
			createFixture(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"name":         "Fade Test Fixture",
			"manufacturer": "Generic",
			"model":        "Dimmer",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)
	require.NoError(t, err)
	fixtureID = fixtureResp.CreateFixture.ID

	// Create scene with fixture value at full
	var sceneResp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err = client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Full Scene",
			"description": "All fixtures at full",
		},
	}, &sceneResp)
	require.NoError(t, err)
	sceneID = sceneResp.CreateScene.ID

	// Add fixture values to scene
	err = client.Mutate(ctx, `
		mutation AddFixtureToScene($sceneId: ID!, $fixtureValues: [FixtureValueInput!]!) {
			addFixturesToScene(sceneId: $sceneId, fixtureValues: $fixtureValues) { id }
		}
	`, map[string]interface{}{
		"sceneId": sceneID,
		"fixtureValues": []map[string]interface{}{
			{
				"fixtureId":     fixtureID,
				"channelValues": []int{255}, // Full brightness
			},
		},
	}, nil)
	require.NoError(t, err)

	return projectID, sceneID, fixtureID
}

func deleteTestProject(t *testing.T, client *graphql.Client, projectID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = client.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{"id": projectID}, nil)
}

func TestActivateSceneWithFade(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create test data
	projectID, sceneID, _ := createTestProjectWithScene(t, client)
	defer deleteTestProject(t, client, projectID)

	// First blackout to ensure clean state
	err := client.Mutate(ctx, `mutation { blackout }`, nil, nil)
	require.NoError(t, err)

	// Activate scene with a 2-second fade
	var activateResp struct {
		ActivateScene struct {
			ID string `json:"id"`
		} `json:"activateScene"`
	}

	err = client.Mutate(ctx, `
		mutation ActivateScene($sceneId: ID!, $fadeTime: Float) {
			activateScene(sceneId: $sceneId, fadeTime: $fadeTime) {
				id
			}
		}
	`, map[string]interface{}{
		"sceneId":  sceneID,
		"fadeTime": 2.0,
	}, &activateResp)

	require.NoError(t, err)
	assert.Equal(t, sceneID, activateResp.ActivateScene.ID)

	// Query DMX output immediately - should be in the process of fading
	time.Sleep(100 * time.Millisecond)

	var dmxResp1 struct {
		DMXOutput struct {
			Channels []int `json:"channels"`
		} `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 0) { channels } }
	`, nil, &dmxResp1)
	require.NoError(t, err)

	// Channel should be between 0 and 255 during fade
	midFadeValue := dmxResp1.DMXOutput.Channels[0]
	t.Logf("Mid-fade value: %d", midFadeValue)

	// Wait for fade to complete
	time.Sleep(2500 * time.Millisecond)

	// Query again - should be at full
	var dmxResp2 struct {
		DMXOutput struct {
			Channels []int `json:"channels"`
		} `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 0) { channels } }
	`, nil, &dmxResp2)
	require.NoError(t, err)

	// Channel should be at full brightness
	assert.Equal(t, 255, dmxResp2.DMXOutput.Channels[0], "Channel should be at 255 after fade completes")

	// Clean up - blackout
	_ = client.Mutate(ctx, `mutation { blackout }`, nil, nil)
}

func TestFadeInterruptionWithNewScene(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create project with fixture
	projectID, scene1ID, fixtureID := createTestProjectWithScene(t, client)
	defer deleteTestProject(t, client, projectID)

	// Create a second scene at half brightness
	var scene2Resp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err := client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Half Scene",
			"description": "Fixtures at half",
		},
	}, &scene2Resp)
	require.NoError(t, err)
	scene2ID := scene2Resp.CreateScene.ID

	// Add fixture values at half
	err = client.Mutate(ctx, `
		mutation AddFixtureToScene($sceneId: ID!, $fixtureValues: [FixtureValueInput!]!) {
			addFixturesToScene(sceneId: $sceneId, fixtureValues: $fixtureValues) { id }
		}
	`, map[string]interface{}{
		"sceneId": scene2ID,
		"fixtureValues": []map[string]interface{}{
			{
				"fixtureId":     fixtureID,
				"channelValues": []int{128},
			},
		},
	}, nil)
	require.NoError(t, err)

	// Blackout first
	_ = client.Mutate(ctx, `mutation { blackout }`, nil, nil)

	// Start a long fade to scene 1
	err = client.Mutate(ctx, `
		mutation ActivateScene($sceneId: ID!, $fadeTime: Float) {
			activateScene(sceneId: $sceneId, fadeTime: $fadeTime) { id }
		}
	`, map[string]interface{}{
		"sceneId":  scene1ID,
		"fadeTime": 5.0,
	}, nil)
	require.NoError(t, err)

	// Wait a moment then interrupt with scene 2
	time.Sleep(500 * time.Millisecond)

	err = client.Mutate(ctx, `
		mutation ActivateScene($sceneId: ID!, $fadeTime: Float) {
			activateScene(sceneId: $sceneId, fadeTime: $fadeTime) { id }
		}
	`, map[string]interface{}{
		"sceneId":  scene2ID,
		"fadeTime": 1.0,
	}, nil)
	require.NoError(t, err)

	// Wait for second fade to complete
	time.Sleep(1500 * time.Millisecond)

	// Should be at scene 2's value (128)
	var dmxResp struct {
		DMXOutput struct {
			Channels []int `json:"channels"`
		} `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 0) { channels } }
	`, nil, &dmxResp)
	require.NoError(t, err)

	// Should be near 128 (allow small tolerance)
	finalValue := dmxResp.DMXOutput.Channels[0]
	assert.InDelta(t, 128, finalValue, 5, "Channel should be around 128 after interruption")

	// Clean up
	_ = client.Mutate(ctx, `mutation { blackout }`, nil, nil)
}

func TestFadeCapturedViaArtNet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	client := graphql.NewClient("")

	// Create test data
	projectID, sceneID, _ := createTestProjectWithScene(t, client)
	defer deleteTestProject(t, client, projectID)

	// Blackout and clear frames
	_ = client.Mutate(ctx, `mutation { blackout }`, nil, nil)
	time.Sleep(100 * time.Millisecond)
	receiver.ClearFrames()

	// Activate scene with fade
	err = client.Mutate(ctx, `
		mutation ActivateScene($sceneId: ID!, $fadeTime: Float) {
			activateScene(sceneId: $sceneId, fadeTime: $fadeTime) { id }
		}
	`, map[string]interface{}{
		"sceneId":  sceneID,
		"fadeTime": 1.0,
	}, nil)
	require.NoError(t, err)

	// Wait for fade to complete plus a bit
	time.Sleep(1500 * time.Millisecond)

	// Get captured frames
	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured - Art-Net may not be enabled")
	}

	t.Logf("Captured %d Art-Net frames during 1s fade", len(frames))

	// Verify we captured frames showing the fade progression
	var values []int
	for _, frame := range frames {
		if frame.Universe == 0 {
			values = append(values, int(frame.Channels[0]))
		}
	}

	if len(values) > 1 {
		// Values should generally increase during fade from 0 to 255
		// Check that we saw intermediate values
		hasIntermediate := false
		for _, v := range values {
			if v > 10 && v < 245 {
				hasIntermediate = true
				break
			}
		}
		assert.True(t, hasIntermediate, "Should capture intermediate fade values via Art-Net")
	}

	// Clean up
	_ = client.Mutate(ctx, `mutation { blackout }`, nil, nil)
}
