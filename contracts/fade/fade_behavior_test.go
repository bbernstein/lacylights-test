// Package fade provides comprehensive fade behavior contract tests.
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

// TestFadeBehaviorEnum tests that the FadeBehavior enum values are accepted.
func TestFadeBehaviorEnum(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a fixture definition with channels that have different fade behaviors
	var createResp struct {
		CreateFixtureDefinition struct {
			ID       string `json:"id"`
			Channels []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				FadeBehavior string `json:"fadeBehavior"`
				IsDiscrete   bool   `json:"isDiscrete"`
			} `json:"channels"`
		} `json:"createFixtureDefinition"`
	}

	err := client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) {
				id
				channels {
					id
					name
					fadeBehavior
					isDiscrete
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test FadeBehavior",
			"model":        "Enum Test",
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{
					"name":         "Dimmer",
					"type":         "INTENSITY",
					"offset":       0,
					"fadeBehavior": "FADE",
					"isDiscrete":   false,
				},
				{
					"name":         "Color Macro",
					"type":         "OTHER",
					"offset":       1,
					"fadeBehavior": "SNAP",
					"isDiscrete":   true,
				},
				{
					"name":         "Gobo",
					"type":         "OTHER",
					"offset":       2,
					"fadeBehavior": "SNAP_END",
					"isDiscrete":   true,
				},
			},
		},
	}, &createResp)

	require.NoError(t, err)
	require.NotEmpty(t, createResp.CreateFixtureDefinition.ID)
	require.Len(t, createResp.CreateFixtureDefinition.Channels, 3)

	// Verify fade behaviors were set correctly
	channelMap := make(map[string]struct {
		FadeBehavior string
		IsDiscrete   bool
	})
	for _, ch := range createResp.CreateFixtureDefinition.Channels {
		channelMap[ch.Name] = struct {
			FadeBehavior string
			IsDiscrete   bool
		}{ch.FadeBehavior, ch.IsDiscrete}
	}

	assert.Equal(t, "FADE", channelMap["Dimmer"].FadeBehavior)
	assert.False(t, channelMap["Dimmer"].IsDiscrete)
	assert.Equal(t, "SNAP", channelMap["Color Macro"].FadeBehavior)
	assert.True(t, channelMap["Color Macro"].IsDiscrete)
	assert.Equal(t, "SNAP_END", channelMap["Gobo"].FadeBehavior)
	assert.True(t, channelMap["Gobo"].IsDiscrete)

	// Cleanup
	_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
		map[string]interface{}{"id": createResp.CreateFixtureDefinition.ID}, nil)
}

// TestFixtureInstanceInheritsFadeBehavior tests that fixture instances inherit FadeBehavior from definitions.
func TestFixtureInstanceInheritsFadeBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

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
		"input": map[string]interface{}{"name": "FadeBehavior Inheritance Test"},
	}, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture definition with mixed fade behaviors
	var defResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test FadeBehavior",
			"model":        "Inheritance Test",
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "fadeBehavior": "FADE"},
				{"name": "Red", "type": "RED", "offset": 1, "fadeBehavior": "FADE"},
				{"name": "Strobe", "type": "OTHER", "offset": 2, "fadeBehavior": "SNAP", "isDiscrete": true},
				{"name": "Color Macro", "type": "OTHER", "offset": 3, "fadeBehavior": "SNAP", "isDiscrete": true},
			},
		},
	}, &defResp)
	require.NoError(t, err)
	defID := defResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": defID}, nil)
	}()

	// Create fixture instance
	var instanceResp struct {
		CreateFixtureInstance struct {
			ID       string `json:"id"`
			Channels []struct {
				Name         string `json:"name"`
				Type         string `json:"type"`
				FadeBehavior string `json:"fadeBehavior"`
				IsDiscrete   bool   `json:"isDiscrete"`
			} `json:"channels"`
		} `json:"createFixtureInstance"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) {
				id
				channels {
					name
					type
					fadeBehavior
					isDiscrete
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"definitionId": defID,
			"name":         "Test Instance",
			"universe":     1,
			"startChannel": 1,
		},
	}, &instanceResp)
	require.NoError(t, err)
	require.Len(t, instanceResp.CreateFixtureInstance.Channels, 4)

	// Verify instance channels inherited FadeBehavior from definition
	for _, ch := range instanceResp.CreateFixtureInstance.Channels {
		switch ch.Name {
		case "Dimmer", "Red":
			assert.Equal(t, "FADE", ch.FadeBehavior, "Channel %s should have FADE behavior", ch.Name)
			assert.False(t, ch.IsDiscrete, "Channel %s should not be discrete", ch.Name)
		case "Strobe", "Color Macro":
			assert.Equal(t, "SNAP", ch.FadeBehavior, "Channel %s should have SNAP behavior", ch.Name)
			assert.True(t, ch.IsDiscrete, "Channel %s should be discrete", ch.Name)
		}
	}
}

// TestBulkUpdateInstanceChannelsFadeBehavior tests bulk updating instance channel fade behaviors.
func TestBulkUpdateInstanceChannelsFadeBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

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
		"input": map[string]interface{}{"name": "Bulk Update FadeBehavior Test"},
	}, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture definition
	var defResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test Bulk Update",
			"model":        "FadeBehavior",
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "fadeBehavior": "FADE"},
				{"name": "Effect", "type": "OTHER", "offset": 1, "fadeBehavior": "FADE"},
			},
		},
	}, &defResp)
	require.NoError(t, err)
	defID := defResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": defID}, nil)
	}()

	// Create fixture instance
	var instanceResp struct {
		CreateFixtureInstance struct {
			ID       string `json:"id"`
			Channels []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				FadeBehavior string `json:"fadeBehavior"`
			} `json:"channels"`
		} `json:"createFixtureInstance"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) {
				id
				channels {
					id
					name
					fadeBehavior
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"definitionId": defID,
			"name":         "Test Instance",
			"universe":     1,
			"startChannel": 1,
		},
	}, &instanceResp)
	require.NoError(t, err)

	// Find the Effect channel ID
	var effectChannelID string
	for _, ch := range instanceResp.CreateFixtureInstance.Channels {
		if ch.Name == "Effect" {
			effectChannelID = ch.ID
			assert.Equal(t, "FADE", ch.FadeBehavior, "Effect channel should start with FADE behavior")
			break
		}
	}
	require.NotEmpty(t, effectChannelID, "Should find Effect channel")

	// Bulk update the Effect channel to SNAP
	var updateResp struct {
		BulkUpdateInstanceChannelsFadeBehavior []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			FadeBehavior string `json:"fadeBehavior"`
		} `json:"bulkUpdateInstanceChannelsFadeBehavior"`
	}
	err = client.Mutate(ctx, `
		mutation BulkUpdateFadeBehavior($channelIds: [ID!]!, $fadeBehavior: FadeBehavior!) {
			bulkUpdateInstanceChannelsFadeBehavior(channelIds: $channelIds, fadeBehavior: $fadeBehavior) {
				id
				name
				fadeBehavior
			}
		}
	`, map[string]interface{}{
		"channelIds":   []string{effectChannelID},
		"fadeBehavior": "SNAP",
	}, &updateResp)

	require.NoError(t, err)
	require.Len(t, updateResp.BulkUpdateInstanceChannelsFadeBehavior, 1)
	assert.Equal(t, "Effect", updateResp.BulkUpdateInstanceChannelsFadeBehavior[0].Name)
	assert.Equal(t, "SNAP", updateResp.BulkUpdateInstanceChannelsFadeBehavior[0].FadeBehavior)
}

// fadeBehaviorTestSetup contains resources for fade behavior DMX tests.
type fadeBehaviorTestSetup struct {
	client       *graphql.Client
	projectID    string
	definitionID string
	fixtureID    string
	sceneBoardID string
	sceneIDs     map[string]string // name -> ID
}

// newFadeBehaviorTestSetup creates test fixtures with mixed fade behaviors.
func newFadeBehaviorTestSetup(t *testing.T) *fadeBehaviorTestSetup {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	resetDMXState(t, client)

	setup := &fadeBehaviorTestSetup{
		client:   client,
		sceneIDs: make(map[string]string),
	}

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
		"input": map[string]interface{}{"name": "FadeBehavior DMX Test"},
	}, &projectResp)
	require.NoError(t, err)
	setup.projectID = projectResp.CreateProject.ID

	// Create fixture definition with FADE and SNAP channels
	var defResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "FadeBehavior DMX Test",
			"model":        "Mixed Channels",
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "fadeBehavior": "FADE"},
				{"name": "Red", "type": "RED", "offset": 1, "fadeBehavior": "FADE"},
				{"name": "Green", "type": "GREEN", "offset": 2, "fadeBehavior": "FADE"},
				{"name": "Blue", "type": "BLUE", "offset": 3, "fadeBehavior": "FADE"},
				{"name": "Color Macro", "type": "OTHER", "offset": 4, "fadeBehavior": "SNAP", "isDiscrete": true},
				{"name": "Strobe", "type": "OTHER", "offset": 5, "fadeBehavior": "SNAP", "isDiscrete": true},
			},
		},
	}, &defResp)
	require.NoError(t, err)
	setup.definitionID = defResp.CreateFixtureDefinition.ID

	// Create fixture instance
	var instanceResp struct {
		CreateFixtureInstance struct {
			ID string `json:"id"`
		} `json:"createFixtureInstance"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    setup.projectID,
			"definitionId": setup.definitionID,
			"name":         "Test Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &instanceResp)
	require.NoError(t, err)
	setup.fixtureID = instanceResp.CreateFixtureInstance.ID

	// Create scene board
	var boardResp struct {
		CreateSceneBoard struct {
			ID string `json:"id"`
		} `json:"createSceneBoard"`
	}
	err = client.Mutate(ctx, `
		mutation CreateSceneBoard($input: CreateSceneBoardInput!) {
			createSceneBoard(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":       setup.projectID,
			"name":            "FadeBehavior Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &boardResp)
	require.NoError(t, err)
	setup.sceneBoardID = boardResp.CreateSceneBoard.ID

	return setup
}

func (s *fadeBehaviorTestSetup) cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resetDMXState(t, s.client)

	_ = s.client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
		map[string]interface{}{"id": s.projectID}, nil)
	_ = s.client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
		map[string]interface{}{"id": s.definitionID}, nil)
}

func (s *fadeBehaviorTestSetup) createScene(t *testing.T, name string, channelValues []int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var resp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err := s.client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": s.projectID,
			"name":      name,
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId":     s.fixtureID,
					"channelValues": channelValues,
				},
			},
		},
	}, &resp)
	require.NoError(t, err)
	s.sceneIDs[name] = resp.CreateScene.ID
	return resp.CreateScene.ID
}

// TestFadeBehaviorDMXOutput tests that SNAP channels jump immediately while FADE channels interpolate.
// This is an Art-Net capture test that verifies actual DMX output behavior.
func TestFadeBehaviorDMXOutput(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	setup := newFadeBehaviorTestSetup(t)
	defer setup.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create two scenes with different values
	// Scene 1: All channels at 0
	setup.createScene(t, "Scene Off", []int{0, 0, 0, 0, 0, 0})

	// Scene 2: All channels at high values
	// [Dimmer=200, R=150, G=100, B=50, ColorMacro=180, Strobe=255]
	sceneOnID := setup.createScene(t, "Scene On", []int{200, 150, 100, 50, 180, 255})

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	// Ensure we start from black
	var fadeResp struct {
		FadeToBlack bool `json:"fadeToBlack"`
	}
	err = setup.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, &fadeResp)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Clear any previous frames before starting capture
	receiver.ClearFrames()

	// Activate scene with 1 second fade
	var activateResp struct {
		ActivateSceneFromBoard bool `json:"activateSceneFromBoard"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  sceneOnID,
		"fadeTime": 1.0,
	}, &activateResp)
	require.NoError(t, err)
	assert.True(t, activateResp.ActivateSceneFromBoard)

	// Wait for fade to complete plus buffer
	time.Sleep(1500 * time.Millisecond)

	// Get captured frames
	frames := receiver.GetFrames()

	if len(frames) < 10 {
		t.Skipf("Not enough Art-Net frames captured (%d), skipping DMX verification", len(frames))
	}

	t.Logf("Captured %d Art-Net frames", len(frames))

	// Analyze frames to verify SNAP vs FADE behavior
	// SNAP channels (5, 6 = Color Macro, Strobe) should reach target immediately
	// FADE channels (1-4 = Dimmer, R, G, B) should interpolate

	// Find first frame index where SNAP channels are at target
	snapTargetFrame := -1
	fadeTargetFrame := -1

	for i, frame := range frames {
		if frame.Universe != 0 { // Universe 1 = index 0
			continue
		}

		// Check SNAP channels (indexes 4, 5 = channels 5, 6)
		// frame.Channels is a fixed-size [512]byte array, so indexes are always valid
		colorMacro := frame.Channels[4]
		strobe := frame.Channels[5]

		if colorMacro == 180 && strobe == 255 && snapTargetFrame == -1 {
			snapTargetFrame = i
			t.Logf("SNAP channels reached target at frame %d", i)
		}

		// Check FADE channels (indexes 0-3 = channels 1-4)
		dimmer := frame.Channels[0]
		red := frame.Channels[1]
		green := frame.Channels[2]
		blue := frame.Channels[3]

		if dimmer == 200 && red == 150 && green == 100 && blue == 50 && fadeTargetFrame == -1 {
			fadeTargetFrame = i
			t.Logf("FADE channels reached target at frame %d", i)
		}
	}

	// SNAP channels should reach target much sooner than FADE channels
	if snapTargetFrame >= 0 && fadeTargetFrame >= 0 {
		assert.Less(t, snapTargetFrame, fadeTargetFrame,
			"SNAP channels should reach target before FADE channels (snap=%d, fade=%d)",
			snapTargetFrame, fadeTargetFrame)
	} else if snapTargetFrame == -1 {
		t.Log("Warning: SNAP channels did not reach target in captured frames")
	}

	// Verify FADE channels were interpolating (not jumping immediately)
	// At ~44Hz Art-Net rate, frame 2 should be ~45ms into a 1-second fade,
	// so FADE channels should be around 4.5% of target value, not at 100%
	if len(frames) > 5 {
		earlyFrame := frames[2]
		if earlyFrame.Universe == 0 {
			dimmer := earlyFrame.Channels[0]
			// Dimmer should not have snapped immediately to target
			// Assert that FADE channels are actually fading, not snapping
			assert.True(t, dimmer < 200,
				"FADE channel (Dimmer) should not immediately reach target at frame 2 (got %d)", dimmer)
			if dimmer > 0 && dimmer < 200 {
				t.Logf("FADE channel (Dimmer) interpolating correctly: value=%d at frame 2", dimmer)
			}
		}
	}
}
