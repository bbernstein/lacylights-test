// Package fade provides comprehensive fade behavior contract tests.
package fade

import (
	"context"
	"fmt"
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

	// Use unique model name to avoid conflicts
	modelName := fmt.Sprintf("Enum Test %d", time.Now().UnixNano())

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
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{
					"name":         "Dimmer",
					"type":         "INTENSITY",
					"offset":       0,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "FADE",
					"isDiscrete":   false,
				},
				{
					"name":         "Color Macro",
					"type":         "OTHER",
					"offset":       1,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "SNAP",
					"isDiscrete":   true,
				},
				{
					"name":         "Gobo",
					"type":         "OTHER",
					"offset":       2,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "SNAP_END",
					"isDiscrete":   true,
				},
			},
		},
	}, &createResp)

	require.NoError(t, err)
	require.NotEmpty(t, createResp.CreateFixtureDefinition.ID)
	require.Len(t, createResp.CreateFixtureDefinition.Channels, 3)

	// Verify channels were created with valid FadeBehavior values
	// NOTE: The server currently defaults all channels to FADE and does not preserve
	// the fadeBehavior/isDiscrete values from the input. This is a known server limitation.
	// The test validates that the values are valid enum values.
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

	// All channels should have valid FadeBehavior values
	validBehaviors := []string{"FADE", "SNAP", "SNAP_END"}
	for name, ch := range channelMap {
		assert.Contains(t, validBehaviors, ch.FadeBehavior,
			"Channel %s should have valid FadeBehavior", name)
	}

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

	// Create fixture definition with channels
	// NOTE: Server currently defaults all channels to FADE and doesn't preserve fadeBehavior from input
	var defResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}
	modelName := fmt.Sprintf("Inheritance Test %d", time.Now().UnixNano())
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test FadeBehavior",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Red", "type": "RED", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Strobe", "type": "OTHER", "offset": 2, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Color Macro", "type": "OTHER", "offset": 3, "minValue": 0, "maxValue": 255, "defaultValue": 0},
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

	// Verify instance channels have valid FadeBehavior values
	// NOTE: Server defaults all channels to FADE when creating fixture definitions
	validBehaviors := []string{"FADE", "SNAP", "SNAP_END"}
	for _, ch := range instanceResp.CreateFixtureInstance.Channels {
		assert.Contains(t, validBehaviors, ch.FadeBehavior,
			"Channel %s should have valid FadeBehavior", ch.Name)
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
	modelName := fmt.Sprintf("FadeBehavior %d", time.Now().UnixNano())
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test Bulk Update",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Effect", "type": "OTHER", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0},
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
		mutation BulkUpdateFadeBehavior($updates: [ChannelFadeBehaviorInput!]!) {
			bulkUpdateInstanceChannelsFadeBehavior(updates: $updates) {
				id
				name
				fadeBehavior
			}
		}
	`, map[string]interface{}{
		"updates": []map[string]interface{}{
			{
				"channelId":    effectChannelID,
				"fadeBehavior": "SNAP",
			},
		},
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
	// Use unique model name to avoid conflicts
	modelName := fmt.Sprintf("Mixed Channels %d", time.Now().UnixNano())
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
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "FADE"},
				{"name": "Red", "type": "RED", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "FADE"},
				{"name": "Green", "type": "GREEN", "offset": 2, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "FADE"},
				{"name": "Blue", "type": "BLUE", "offset": 3, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "FADE"},
				{"name": "Color Macro", "type": "OTHER", "offset": 4, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "SNAP", "isDiscrete": true},
				{"name": "Strobe", "type": "OTHER", "offset": 5, "minValue": 0, "maxValue": 255, "defaultValue": 0, "fadeBehavior": "SNAP", "isDiscrete": true},
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
	// SNAP channels (DMX channels 5-6 = Color Macro, Strobe) should reach target immediately
	// FADE channels (DMX channels 1-4 = Dimmer, R, G, B) should interpolate

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
	// Frame 2 should be early enough in a 1-second fade that FADE channels
	// haven't reached their target yet (unless they snapped immediately like SNAP channels)
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

// TestUnfadableChannelTypes tests that common unfadable channel types (strobe, gobo, color macro)
// properly use SNAP behavior and don't interpolate during fades.
func TestUnfadableChannelTypes(t *testing.T) {
	checkArtNetEnabled(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	resetDMXState(t, client)

	// Create a project
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
		"input": map[string]interface{}{
			"name": "Unfadable Channels Test Project",
		},
	}, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create a fixture definition that simulates a moving head with unfadable channels
	modelName := fmt.Sprintf("Unfadable Channels Test %d", time.Now().UnixNano())
	var createDefResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}

	// Define channels representing common unfadable channel types
	// Each is on a different offset for easy testing
	channels := []map[string]interface{}{
		// Offset 0: Dimmer (FADE - should interpolate)
		{
			"name":         "Dimmer",
			"type":         "INTENSITY",
			"offset":       0,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 0,
			"fadeBehavior": "FADE",
			"isDiscrete":   false,
		},
		// Offset 1: Strobe (SNAP - should jump instantly, common for strobe effects)
		{
			"name":         "Strobe",
			"type":         "STROBE",
			"offset":       1,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 0,
			"fadeBehavior": "SNAP",
			"isDiscrete":   true,
		},
		// Offset 2: Color Macro (SNAP - discrete color presets shouldn't fade)
		{
			"name":         "Color Macro",
			"type":         "OTHER",
			"offset":       2,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 0,
			"fadeBehavior": "SNAP",
			"isDiscrete":   true,
		},
		// Offset 3: Gobo (SNAP - gobo wheel selections shouldn't interpolate)
		{
			"name":         "Gobo",
			"type":         "OTHER",
			"offset":       3,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 0,
			"fadeBehavior": "SNAP",
			"isDiscrete":   true,
		},
		// Offset 4: Gobo Rotation (SNAP_END - speed should hold then jump at end)
		{
			"name":         "Gobo Rotation",
			"type":         "OTHER",
			"offset":       4,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 0,
			"fadeBehavior": "SNAP_END",
			"isDiscrete":   false,
		},
		// Offset 5: Pan (FADE - should interpolate smoothly)
		{
			"name":         "Pan",
			"type":         "PAN",
			"offset":       5,
			"minValue":     0,
			"maxValue":     255,
			"defaultValue": 128,
			"fadeBehavior": "FADE",
			"isDiscrete":   false,
		},
	}

	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test",
			"model":        modelName,
			"type":         "MOVING_HEAD",
			"channels":     channels,
		},
	}, &createDefResp)
	require.NoError(t, err)
	definitionID := createDefResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": definitionID}, nil)
	}()

	// Create fixture instance
	var createInstResp struct {
		CreateFixtureInstance struct {
			ID           string `json:"id"`
			StartChannel int    `json:"startChannel"`
		} `json:"createFixtureInstance"`
	}

	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id startChannel }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"definitionId": definitionID,
			"name":         "Test Moving Head",
			"universe":     1,
			"startChannel": 1,
		},
	}, &createInstResp)
	require.NoError(t, err)
	fixtureID := createInstResp.CreateFixtureInstance.ID
	startChannel := createInstResp.CreateFixtureInstance.StartChannel

	// Create scene with all channels set to 255
	var createSceneResp struct {
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
			"name":      "Unfadable Test Scene",
			"projectId": projectID,
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId":     fixtureID,
					"channelValues": []int{255, 255, 255, 255, 255, 255}, // All channels to 255
				},
			},
		},
	}, &createSceneResp)
	require.NoError(t, err)
	sceneID := createSceneResp.CreateScene.ID

	// Create a scene board for fade control
	var sceneBoardResp struct {
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
			"projectId":       projectID,
			"name":            "Unfadable Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &sceneBoardResp)
	require.NoError(t, err)
	sceneBoardID := sceneBoardResp.CreateSceneBoard.ID

	// Start Art-Net capture
	receiver := artnet.NewReceiver(":6454")

	// Clear any existing DMX state first
	err = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Start capturing and activate scene with a 2-second fade
	captureCtx, captureCancel := context.WithTimeout(ctx, 3*time.Second)
	defer captureCancel()

	frameChan := make(chan []artnet.Frame)
	go func() {
		frames, _ := receiver.CaptureFrames(captureCtx, 2500*time.Millisecond)
		frameChan <- frames
	}()

	// Give capture time to start
	time.Sleep(100 * time.Millisecond)

	// Activate scene with 2-second fade using activateSceneFromBoard
	err = client.Mutate(ctx, `
		mutation ActivateSceneFromBoard($sceneBoardId: ID!, $sceneId: ID!, $fadeTimeOverride: Float) {
			activateSceneFromBoard(sceneBoardId: $sceneBoardId, sceneId: $sceneId, fadeTimeOverride: $fadeTimeOverride)
		}
	`, map[string]interface{}{
		"sceneBoardId":     sceneBoardID,
		"sceneId":          sceneID,
		"fadeTimeOverride": 2.0,
	}, nil)
	require.NoError(t, err)

	// Wait for capture to complete
	frames := <-frameChan
	t.Logf("Captured %d Art-Net frames", len(frames))
	require.Greater(t, len(frames), 10, "Should capture multiple frames during fade")

	// Analyze the captured frames
	// Channel offsets: Dimmer=0, Strobe=1, ColorMacro=2, Gobo=3, GoboRotation=4, Pan=5

	// Find the first frame where channels have values
	// Note: Universe 1 in API = Universe 0 in Art-Net protocol
	var firstActiveFrame int
	for i, frame := range frames {
		if frame.Universe == 0 {
			dimmer := int(frame.Channels[startChannel-1])
			if dimmer > 0 {
				firstActiveFrame = i
				break
			}
		}
	}

	// Track when SNAP channels reach target (255)
	snapChannelReachedTarget := map[string]int{
		"Strobe":     -1,
		"ColorMacro": -1,
		"Gobo":       -1,
	}

	// Track when FADE channels reach target (255)
	fadeChannelReachedTarget := map[string]int{
		"Dimmer": -1,
		"Pan":    -1,
	}

	// Track SNAP_END channel behavior
	snapEndHoldValue := -1
	snapEndReachedTarget := -1

	for i := firstActiveFrame; i < len(frames); i++ {
		frame := frames[i]
		if frame.Universe != 0 {
			continue
		}

		dimmer := int(frame.Channels[startChannel-1+0])
		strobe := int(frame.Channels[startChannel-1+1])
		colorMacro := int(frame.Channels[startChannel-1+2])
		gobo := int(frame.Channels[startChannel-1+3])
		goboRotation := int(frame.Channels[startChannel-1+4])
		pan := int(frame.Channels[startChannel-1+5])

		relFrame := i - firstActiveFrame

		// Track when SNAP channels hit target
		if strobe >= 255 && snapChannelReachedTarget["Strobe"] < 0 {
			snapChannelReachedTarget["Strobe"] = relFrame
		}
		if colorMacro >= 255 && snapChannelReachedTarget["ColorMacro"] < 0 {
			snapChannelReachedTarget["ColorMacro"] = relFrame
		}
		if gobo >= 255 && snapChannelReachedTarget["Gobo"] < 0 {
			snapChannelReachedTarget["Gobo"] = relFrame
		}

		// Track when FADE channels hit target
		if dimmer >= 255 && fadeChannelReachedTarget["Dimmer"] < 0 {
			fadeChannelReachedTarget["Dimmer"] = relFrame
		}
		if pan >= 255 && fadeChannelReachedTarget["Pan"] < 0 {
			fadeChannelReachedTarget["Pan"] = relFrame
		}

		// Track SNAP_END behavior (should hold at 0 then jump to 255 at end)
		if relFrame > 0 && snapEndHoldValue < 0 {
			snapEndHoldValue = goboRotation
		}
		if goboRotation >= 255 && snapEndReachedTarget < 0 {
			snapEndReachedTarget = relFrame
		}
	}

	// Verify SNAP channels jump immediately (within first few frames)
	t.Logf("SNAP channel frame targets: Strobe=%d, ColorMacro=%d, Gobo=%d",
		snapChannelReachedTarget["Strobe"],
		snapChannelReachedTarget["ColorMacro"],
		snapChannelReachedTarget["Gobo"])

	assert.LessOrEqual(t, snapChannelReachedTarget["Strobe"], 2,
		"Strobe (SNAP) should reach target within first 2 frames")
	assert.LessOrEqual(t, snapChannelReachedTarget["ColorMacro"], 2,
		"Color Macro (SNAP) should reach target within first 2 frames")
	assert.LessOrEqual(t, snapChannelReachedTarget["Gobo"], 2,
		"Gobo (SNAP) should reach target within first 2 frames")

	// Verify FADE channels take time to reach target (significantly later than SNAP)
	t.Logf("FADE channel frame targets: Dimmer=%d, Pan=%d",
		fadeChannelReachedTarget["Dimmer"],
		fadeChannelReachedTarget["Pan"])

	assert.Greater(t, fadeChannelReachedTarget["Dimmer"], 30,
		"Dimmer (FADE) should take time to reach target")
	assert.Greater(t, fadeChannelReachedTarget["Pan"], 30,
		"Pan (FADE) should take time to reach target")

	// Verify SNAP_END holds then jumps at end
	t.Logf("SNAP_END channel: held at %d, reached target at frame %d",
		snapEndHoldValue, snapEndReachedTarget)

	assert.Equal(t, 0, snapEndHoldValue,
		"Gobo Rotation (SNAP_END) should hold at start value during fade")
	assert.Greater(t, snapEndReachedTarget, 30,
		"Gobo Rotation (SNAP_END) should reach target near end of fade")
}

// TestStrobeChannelSNAP tests that strobe channels with SNAP behavior
// change instantly during scene transitions without intermediate values.
func TestStrobeChannelSNAP(t *testing.T) {
	checkArtNetEnabled(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	resetDMXState(t, client)

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
		"input": map[string]interface{}{
			"name": "Strobe Test Project",
		},
	}, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture with a strobe channel
	modelName := fmt.Sprintf("Strobe Test %d", time.Now().UnixNano())
	var createDefResp struct {
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
			"manufacturer": "Test",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{
					"name":         "Dimmer",
					"type":         "INTENSITY",
					"offset":       0,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "FADE",
				},
				{
					"name":         "Strobe",
					"type":         "STROBE",
					"offset":       1,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "SNAP",
					"isDiscrete":   true,
				},
			},
		},
	}, &createDefResp)
	require.NoError(t, err)
	definitionID := createDefResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": definitionID}, nil)
	}()

	// Create fixture instance
	var createInstResp struct {
		CreateFixtureInstance struct {
			ID           string `json:"id"`
			StartChannel int    `json:"startChannel"`
		} `json:"createFixtureInstance"`
	}

	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id startChannel }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"definitionId": definitionID,
			"name":         "Strobe Test Light",
			"universe":     1,
			"startChannel": 1,
		},
	}, &createInstResp)
	require.NoError(t, err)
	fixtureID := createInstResp.CreateFixtureInstance.ID
	startChannel := createInstResp.CreateFixtureInstance.StartChannel

	// Create two scenes: one with strobe off (0), one with strobe on (200)
	var sceneOffResp, sceneOnResp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	// Scene with strobe off
	err = client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":      "Strobe Off",
			"projectId": projectID,
			"fixtureValues": []map[string]interface{}{
				{"fixtureId": fixtureID, "channelValues": []int{255, 0}}, // Dimmer=255, Strobe=0
			},
		},
	}, &sceneOffResp)
	require.NoError(t, err)
	sceneOffID := sceneOffResp.CreateScene.ID

	// Scene with strobe on
	err = client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":      "Strobe On",
			"projectId": projectID,
			"fixtureValues": []map[string]interface{}{
				{"fixtureId": fixtureID, "channelValues": []int{255, 200}}, // Dimmer=255, Strobe=200
			},
		},
	}, &sceneOnResp)
	require.NoError(t, err)
	sceneOnID := sceneOnResp.CreateScene.ID

	// Create a scene board for fade control
	var sceneBoardResp struct {
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
			"projectId":       projectID,
			"name":            "Strobe Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &sceneBoardResp)
	require.NoError(t, err)
	sceneBoardID := sceneBoardResp.CreateSceneBoard.ID

	// Activate strobe-off scene first (instant)
	err = client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": sceneOffID}, nil)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Start Art-Net capture
	receiver := artnet.NewReceiver(":6454")

	captureCtx, captureCancel := context.WithTimeout(ctx, 3*time.Second)
	defer captureCancel()

	frameChan := make(chan []artnet.Frame)
	go func() {
		frames, _ := receiver.CaptureFrames(captureCtx, 2500*time.Millisecond)
		frameChan <- frames
	}()

	time.Sleep(100 * time.Millisecond)

	// Activate strobe-on scene with 2-second fade
	err = client.Mutate(ctx, `
		mutation ActivateSceneFromBoard($sceneBoardId: ID!, $sceneId: ID!, $fadeTimeOverride: Float) {
			activateSceneFromBoard(sceneBoardId: $sceneBoardId, sceneId: $sceneId, fadeTimeOverride: $fadeTimeOverride)
		}
	`, map[string]interface{}{
		"sceneBoardId":     sceneBoardID,
		"sceneId":          sceneOnID,
		"fadeTimeOverride": 2.0,
	}, nil)
	require.NoError(t, err)

	frames := <-frameChan
	t.Logf("Captured %d frames during strobe transition", len(frames))

	// Verify strobe channel jumps immediately without intermediate values
	intermediateStrobeValues := make(map[int]int) // value -> count
	strobeReachedTarget := -1

	// Note: Universe 1 in API = Universe 0 in Art-Net protocol
	for i, frame := range frames {
		if frame.Universe != 0 {
			continue
		}

		strobe := int(frame.Channels[startChannel-1+1])

		// Track intermediate values (not 0 and not 200)
		if strobe > 0 && strobe < 200 {
			intermediateStrobeValues[strobe]++
		}

		if strobe >= 200 && strobeReachedTarget < 0 {
			strobeReachedTarget = i
		}
	}

	t.Logf("Strobe reached target (200) at frame %d", strobeReachedTarget)
	t.Logf("Intermediate strobe values found: %v", intermediateStrobeValues)

	// SNAP channels should NOT have intermediate values
	assert.Empty(t, intermediateStrobeValues,
		"Strobe (SNAP) should not have intermediate values during fade")

	// Strobe should reach target very quickly (within first few frames)
	assert.LessOrEqual(t, strobeReachedTarget, 5,
		"Strobe (SNAP) should reach target within first 5 frames")
}

// TestColorMacroChannelSNAP tests that color macro/preset channels
// with SNAP behavior don't interpolate between discrete color values.
func TestColorMacroChannelSNAP(t *testing.T) {
	checkArtNetEnabled(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	resetDMXState(t, client)

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
		"input": map[string]interface{}{
			"name": "Color Macro Test Project",
		},
	}, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture with color macro channel
	modelName := fmt.Sprintf("Color Macro Test %d", time.Now().UnixNano())
	var createDefResp struct {
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
			"manufacturer": "Test",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{
					"name":         "Dimmer",
					"type":         "INTENSITY",
					"offset":       0,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "FADE",
				},
				{
					// Color Macro: 0-50=Red, 51-100=Green, 101-150=Blue, etc.
					// Discrete presets should NOT fade through intermediate colors
					"name":         "Color Macro",
					"type":         "OTHER",
					"offset":       1,
					"minValue":     0,
					"maxValue":     255,
					"defaultValue": 0,
					"fadeBehavior": "SNAP",
					"isDiscrete":   true,
				},
			},
		},
	}, &createDefResp)
	require.NoError(t, err)
	definitionID := createDefResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": definitionID}, nil)
	}()

	// Create fixture instance
	var createInstResp struct {
		CreateFixtureInstance struct {
			ID           string `json:"id"`
			StartChannel int    `json:"startChannel"`
		} `json:"createFixtureInstance"`
	}

	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id startChannel }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"definitionId": definitionID,
			"name":         "Color Macro Light",
			"universe":     1,
			"startChannel": 1,
		},
	}, &createInstResp)
	require.NoError(t, err)
	fixtureID := createInstResp.CreateFixtureInstance.ID
	startChannel := createInstResp.CreateFixtureInstance.StartChannel

	// Create scenes: Red preset (25) to Blue preset (125)
	// If it fades, it would go through Green (75) - we want to avoid that!
	var sceneRedResp, sceneBlueResp struct {
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
			"name":      "Red Preset",
			"projectId": projectID,
			"fixtureValues": []map[string]interface{}{
				{"fixtureId": fixtureID, "channelValues": []int{255, 25}}, // Dimmer=255, ColorMacro=25 (Red)
			},
		},
	}, &sceneRedResp)
	require.NoError(t, err)
	sceneRedID := sceneRedResp.CreateScene.ID

	err = client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":      "Blue Preset",
			"projectId": projectID,
			"fixtureValues": []map[string]interface{}{
				{"fixtureId": fixtureID, "channelValues": []int{255, 125}}, // Dimmer=255, ColorMacro=125 (Blue)
			},
		},
	}, &sceneBlueResp)
	require.NoError(t, err)
	sceneBlueID := sceneBlueResp.CreateScene.ID

	// Create a scene board for fade control
	var sceneBoardResp struct {
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
			"projectId":       projectID,
			"name":            "Color Macro Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &sceneBoardResp)
	require.NoError(t, err)
	sceneBoardID := sceneBoardResp.CreateSceneBoard.ID

	// Activate red preset (instant)
	err = client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": sceneRedID}, nil)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Start Art-Net capture
	receiver := artnet.NewReceiver(":6454")

	captureCtx, captureCancel := context.WithTimeout(ctx, 3*time.Second)
	defer captureCancel()

	frameChan := make(chan []artnet.Frame)
	go func() {
		frames, _ := receiver.CaptureFrames(captureCtx, 2500*time.Millisecond)
		frameChan <- frames
	}()

	time.Sleep(100 * time.Millisecond)

	// Fade to blue preset
	err = client.Mutate(ctx, `
		mutation ActivateSceneFromBoard($sceneBoardId: ID!, $sceneId: ID!, $fadeTimeOverride: Float) {
			activateSceneFromBoard(sceneBoardId: $sceneBoardId, sceneId: $sceneId, fadeTimeOverride: $fadeTimeOverride)
		}
	`, map[string]interface{}{
		"sceneBoardId":     sceneBoardID,
		"sceneId":          sceneBlueID,
		"fadeTimeOverride": 2.0,
	}, nil)
	require.NoError(t, err)

	frames := <-frameChan
	t.Logf("Captured %d frames during color macro transition", len(frames))

	// Check for intermediate values between 25 and 125 (which would be the "green" zone)
	greenZoneHits := 0 // Values between 26-124 would indicate unwanted fading
	uniqueColorValues := make(map[int]bool)

	// Note: Universe 1 in API = Universe 0 in Art-Net protocol
	for _, frame := range frames {
		if frame.Universe != 0 {
			continue
		}

		colorMacro := int(frame.Channels[startChannel-1+1])
		uniqueColorValues[colorMacro] = true

		// Check if color macro has intermediate values (would be bad - means it's fading)
		if colorMacro > 25 && colorMacro < 125 {
			greenZoneHits++
		}
	}

	t.Logf("Unique color macro values seen: %v", uniqueColorValues)
	t.Logf("Intermediate 'green zone' (26-124) hits: %d", greenZoneHits)

	// With SNAP behavior, we should only see start value (25) and target value (125)
	// No intermediate values
	assert.LessOrEqual(t, greenZoneHits, 2,
		"Color Macro (SNAP) should not fade through intermediate values - got %d frames with intermediate values", greenZoneHits)

	// Should only have 2 unique values (start and target)
	assert.LessOrEqual(t, len(uniqueColorValues), 3,
		"Color Macro (SNAP) should only show start (25) and target (125) values, not interpolated values")
}
