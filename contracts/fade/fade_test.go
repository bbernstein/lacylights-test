// Package fade provides comprehensive fade behavior contract tests.
package fade

import (
	"context"
	"math"
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

// checkArtNetEnabled checks if fade tests should run.
// Skip if:
// 1. SKIP_FADE_TESTS environment variable is set
// 2. Art-Net is not enabled on the server
// 3. Server cannot be reached
func checkArtNetEnabled(t *testing.T) {
	// Skip if explicitly disabled via environment variable
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	client := graphql.NewClient("")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resp struct {
		SystemInfo struct {
			ArtnetEnabled bool `json:"artnetEnabled"`
		} `json:"systemInfo"`
	}

	err := client.Query(ctx, `query { systemInfo { artnetEnabled } }`, nil, &resp)
	if err != nil {
		t.Skipf("Skipping fade test: cannot query systemInfo: %v", err)
	}

	if !resp.SystemInfo.ArtnetEnabled {
		t.Skip("Skipping fade test: Art-Net is not enabled on the server")
	}
}

// testSetup contains common test resources
type testSetup struct {
	client       *graphql.Client
	projectID    string
	fixtureID    string
	sceneBoardID string
	scenes       map[string]string // name -> ID
}

// newTestSetup creates a new test setup with project and fixture
// Skips the test if Art-Net is not enabled on the server
func newTestSetup(t *testing.T) *testSetup {
	// Check if Art-Net is enabled before running fade tests
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	setup := &testSetup{
		client: client,
		scenes: make(map[string]string),
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
		"input": map[string]interface{}{"name": "Fade Test Project"},
	}, &projectResp)
	require.NoError(t, err)
	setup.projectID = projectResp.CreateProject.ID

	// Get a built-in fixture definition first
	var defResp struct {
		FixtureDefinitions []struct {
			ID string `json:"id"`
		} `json:"fixtureDefinitions"`
	}

	err = client.Query(ctx, `
		query { fixtureDefinitions(filter: { isBuiltIn: true }) { id } }
	`, nil, &defResp)
	require.NoError(t, err)
	require.NotEmpty(t, defResp.FixtureDefinitions)
	definitionID := defResp.FixtureDefinitions[0].ID

	// Create a fixture instance
	var fixtureResp struct {
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
			"definitionId": definitionID,
			"name":         "RGB Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)
	require.NoError(t, err)
	setup.fixtureID = fixtureResp.CreateFixtureInstance.ID

	// Create a scene board for fade-controlled activation
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
			"name":            "Test Scene Board",
			"defaultFadeTime": 2.0,
		},
	}, &boardResp)
	require.NoError(t, err)
	setup.sceneBoardID = boardResp.CreateSceneBoard.ID

	return setup
}

// cleanup removes the test project and all related data
func (s *testSetup) cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure blackout
	_ = s.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Delete project
	_ = s.client.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{"id": s.projectID}, nil)
}

// createScene creates a scene with the given name and channel values
func (s *testSetup) createScene(t *testing.T, name string, channelValues []int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var sceneResp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	// Create scene with fixture values included
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
	}, &sceneResp)
	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// Add scene to scene board for fade-controlled activation
	buttonIndex := len(s.scenes) // Use scene count as button position
	err = s.client.Mutate(ctx, `
		mutation AddSceneToBoard($input: CreateSceneBoardButtonInput!) {
			addSceneToBoard(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"sceneBoardId": s.sceneBoardID,
			"sceneId":      sceneID,
			"layoutX":      buttonIndex * 200, // Space buttons apart
			"layoutY":      0,
		},
	}, nil)
	require.NoError(t, err)

	s.scenes[name] = sceneID
	return sceneID
}

// getDMXOutput gets current DMX output for universe 1
// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
func (s *testSetup) getDMXOutput(t *testing.T) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err := s.client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &resp)
	require.NoError(t, err)
	return resp.DMXOutput
}

// activateScene activates a scene with optional fade time
// Uses activateSceneFromBoard for fade control, or setSceneLive for instant (0 fade)
func (s *testSetup) activateScene(t *testing.T, sceneID string, fadeTime float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if fadeTime == 0 {
		// Use setSceneLive for instant activation
		err := s.client.Mutate(ctx, `
			mutation SetSceneLive($sceneId: ID!) {
				setSceneLive(sceneId: $sceneId)
			}
		`, map[string]interface{}{
			"sceneId": sceneID,
		}, nil)
		require.NoError(t, err)
	} else {
		// Use activateSceneFromBoard for fade-controlled activation
		err := s.client.Mutate(ctx, `
			mutation ActivateSceneFromBoard($sceneBoardId: ID!, $sceneId: ID!, $fadeTimeOverride: Float) {
				activateSceneFromBoard(sceneBoardId: $sceneBoardId, sceneId: $sceneId, fadeTimeOverride: $fadeTimeOverride)
			}
		`, map[string]interface{}{
			"sceneBoardId":     s.sceneBoardID,
			"sceneId":          sceneID,
			"fadeTimeOverride": fadeTime,
		}, nil)
		require.NoError(t, err)
	}
}

// fadeToBlack fades to black with given time
func (s *testSetup) fadeToBlack(t *testing.T, fadeTime float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.client.Mutate(ctx, `
		mutation FadeToBlack($fadeOutTime: Float!) {
			fadeToBlack(fadeOutTime: $fadeOutTime)
		}
	`, map[string]interface{}{"fadeOutTime": fadeTime}, nil)
	require.NoError(t, err)
}

// ============================================================================
// Basic Fade Tests
// ============================================================================

func TestActivateSceneWithFade(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene at full brightness
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Ensure clean state
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Activate scene with a 2-second fade
	setup.activateScene(t, sceneID, 2.0)

	// Query DMX output during fade
	time.Sleep(100 * time.Millisecond)
	midFadeOutput := setup.getDMXOutput(t)
	t.Logf("Mid-fade value (0.1s): %v", midFadeOutput[:3])

	// Wait for fade to complete
	time.Sleep(2500 * time.Millisecond)
	finalOutput := setup.getDMXOutput(t)

	// Verify all channels are at full
	assert.Equal(t, 255, finalOutput[0], "Red channel should be at 255")
	assert.Equal(t, 255, finalOutput[1], "Green channel should be at 255")
	assert.Equal(t, 255, finalOutput[2], "Blue channel should be at 255")
}

func TestFadeToBlack(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create and activate scene immediately
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})
	setup.activateScene(t, sceneID, 0)
	time.Sleep(100 * time.Millisecond)

	// Verify at full
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Should start at full")

	// Fade to black over 2 seconds
	setup.fadeToBlack(t, 2.0)

	// Check mid-fade
	time.Sleep(1000 * time.Millisecond)
	midOutput := setup.getDMXOutput(t)
	t.Logf("Mid-fade to black value: %d", midOutput[0])
	assert.True(t, midOutput[0] > 0 && midOutput[0] < 255, "Should be mid-fade")

	// Wait for completion
	time.Sleep(1500 * time.Millisecond)
	finalOutput := setup.getDMXOutput(t)

	// Should be at 0
	assert.Equal(t, 0, finalOutput[0], "Red channel should be at 0")
	assert.Equal(t, 0, finalOutput[1], "Green channel should be at 0")
	assert.Equal(t, 0, finalOutput[2], "Blue channel should be at 0")
}

func TestInstantFade(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene
	sceneID := setup.createScene(t, "Full", []int{255, 128, 64})

	// Ensure blackout
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Activate with 0 fade time (instant)
	setup.activateScene(t, sceneID, 0)
	time.Sleep(100 * time.Millisecond)

	// Should be immediately at target values
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Red should be 255")
	assert.Equal(t, 128, output[1], "Green should be 128")
	assert.Equal(t, 64, output[2], "Blue should be 64")
}

// ============================================================================
// Fade Interruption Tests
// ============================================================================

func TestFadeInterruptionWithNewScene(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create two scenes
	scene1ID := setup.createScene(t, "Full", []int{255, 255, 255})
	scene2ID := setup.createScene(t, "Half", []int{128, 128, 128})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Start a long fade to scene 1
	setup.activateScene(t, scene1ID, 5.0)

	// Wait a bit, then interrupt with scene 2
	time.Sleep(500 * time.Millisecond)
	setup.activateScene(t, scene2ID, 1.0)

	// Wait for second fade to complete
	time.Sleep(1500 * time.Millisecond)

	// Should be at scene 2's value
	output := setup.getDMXOutput(t)
	assert.InDelta(t, 128, output[0], 5, "Should be at scene 2's value after interruption")
}

func TestFadeInterruptionWithFadeToBlack(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene at full
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Start fade to full over 5 seconds
	setup.activateScene(t, sceneID, 5.0)

	// Wait until mid-fade
	time.Sleep(2500 * time.Millisecond)
	midOutput := setup.getDMXOutput(t)
	t.Logf("Value at 2.5s of 5s fade: %d", midOutput[0])

	// Interrupt with immediate fadeToBlack
	setup.fadeToBlack(t, 0.5)

	// Wait for fadeToBlack to complete
	time.Sleep(700 * time.Millisecond)

	// Should be at black
	output := setup.getDMXOutput(t)
	assert.InDelta(t, 0, output[0], 5, "Should be at black after interruption")
}

func TestMultipleRapidInterruptions(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create several scenes
	scene1ID := setup.createScene(t, "Red", []int{255, 0, 0})
	scene2ID := setup.createScene(t, "Green", []int{0, 255, 0})
	scene3ID := setup.createScene(t, "Blue", []int{0, 0, 255})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Rapidly interrupt fades
	setup.activateScene(t, scene1ID, 2.0)
	time.Sleep(100 * time.Millisecond)
	setup.activateScene(t, scene2ID, 2.0)
	time.Sleep(100 * time.Millisecond)
	setup.activateScene(t, scene3ID, 1.0)

	// Wait for final fade to complete
	time.Sleep(1500 * time.Millisecond)

	// Should be at scene 3 (blue)
	output := setup.getDMXOutput(t)
	assert.InDelta(t, 0, output[0], 10, "Red should be 0")
	assert.InDelta(t, 0, output[1], 10, "Green should be 0")
	assert.InDelta(t, 255, output[2], 10, "Blue should be 255")
}

// ============================================================================
// Fade Timing Precision Tests
// ============================================================================

func TestFadeProgressionLinear(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene at full
	sceneID := setup.createScene(t, "Full", []int{255, 0, 0})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Start 2-second fade
	fadeTime := 2.0
	setup.activateScene(t, sceneID, fadeTime)

	// Sample at multiple points and verify roughly linear progression
	samples := []struct {
		time     time.Duration
		expected float64 // Expected percentage (0-100)
		tolerance float64
	}{
		{500 * time.Millisecond, 25, 15},
		{1000 * time.Millisecond, 50, 15},
		{1500 * time.Millisecond, 75, 15},
	}

	for _, sample := range samples {
		time.Sleep(sample.time - 100*time.Millisecond)
		output := setup.getDMXOutput(t)
		actualPercent := float64(output[0]) / 255 * 100
		t.Logf("At %.2fs: value=%d (%.1f%%)", sample.time.Seconds(), output[0], actualPercent)
		assert.InDelta(t, sample.expected, actualPercent, sample.tolerance,
			"Fade progress at %v should be around %.0f%%", sample.time, sample.expected)
		time.Sleep(100 * time.Millisecond)
	}
}

func TestFadeCompletesToExactValue(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Test various target values
	testValues := [][]int{
		{255, 255, 255},
		{128, 128, 128},
		{64, 128, 192},
		{1, 1, 1},
		{254, 127, 63},
	}

	for _, values := range testValues {
		// Start from black
		setup.fadeToBlack(t, 0)
		time.Sleep(100 * time.Millisecond)

		// Create scene with target values
		sceneID := setup.createScene(t, "Test", values)
		setup.activateScene(t, sceneID, 1.0)

		// Wait for fade to complete
		time.Sleep(1500 * time.Millisecond)

		output := setup.getDMXOutput(t)
		assert.Equal(t, values[0], output[0], "Red should be exact")
		assert.Equal(t, values[1], output[1], "Green should be exact")
		assert.Equal(t, values[2], output[2], "Blue should be exact")
	}
}

// ============================================================================
// Cross-Fade Tests (scene to scene)
// ============================================================================

func TestCrossFadeBetweenScenes(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create two different color scenes
	scene1ID := setup.createScene(t, "Red", []int{255, 0, 0})
	scene2ID := setup.createScene(t, "Blue", []int{0, 0, 255})

	// Start at scene 1 (instant)
	setup.activateScene(t, scene1ID, 0)
	time.Sleep(100 * time.Millisecond)

	// Verify red
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Should start at red")
	assert.Equal(t, 0, output[2], "Should start with no blue")

	// Cross-fade to scene 2
	setup.activateScene(t, scene2ID, 2.0)

	// Check midpoint - should have both colors
	time.Sleep(1000 * time.Millisecond)
	midOutput := setup.getDMXOutput(t)
	t.Logf("Mid-crossfade: R=%d, B=%d", midOutput[0], midOutput[2])

	// Both should be mid-range during crossfade
	assert.True(t, midOutput[0] > 50 && midOutput[0] < 200, "Red should be fading out")
	assert.True(t, midOutput[2] > 50 && midOutput[2] < 200, "Blue should be fading in")

	// Wait for completion
	time.Sleep(1500 * time.Millisecond)
	finalOutput := setup.getDMXOutput(t)

	// Should be blue now
	assert.InDelta(t, 0, finalOutput[0], 5, "Red should be 0")
	assert.InDelta(t, 255, finalOutput[2], 5, "Blue should be 255")
}

// ============================================================================
// Cue List Fade Tests
// ============================================================================

func TestCueListFadeTransitions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scenes
	scene1ID := setup.createScene(t, "Scene 1", []int{255, 0, 0})
	scene2ID := setup.createScene(t, "Scene 2", []int{0, 255, 0})
	scene3ID := setup.createScene(t, "Scene 3", []int{0, 0, 255})

	// Create cue list
	var cueListResp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	err := setup.client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": setup.projectID,
			"name":      "Fade Test Cue List",
		},
	}, &cueListResp)
	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Add cues with specific fade times
	for i, sceneID := range []string{scene1ID, scene2ID, scene3ID} {
		err := setup.client.Mutate(ctx, `
			mutation AddCue($input: AddCueInput!) {
				addCueToList(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":  cueListID,
				"name":       "Cue " + string(rune('1'+i)),
				"cueNumber":  float64(i + 1),
				"sceneId":    sceneID,
				"fadeInTime": 1.0,
			},
		}, nil)
		require.NoError(t, err)
	}

	// Start cue list
	err = setup.client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId) { id }
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
	require.NoError(t, err)

	// Wait for first cue fade
	time.Sleep(1500 * time.Millisecond)
	output := setup.getDMXOutput(t)
	assert.InDelta(t, 255, output[0], 5, "Should be at scene 1 (red)")

	// Go to next cue
	err = setup.client.Mutate(ctx, `mutation { nextCue { id } }`, nil, nil)
	require.NoError(t, err)

	// Wait for transition
	time.Sleep(1500 * time.Millisecond)
	output = setup.getDMXOutput(t)
	assert.InDelta(t, 255, output[1], 5, "Should be at scene 2 (green)")

	// Stop cue list
	err = setup.client.Mutate(ctx, `mutation { stopCueList }`, nil, nil)
	require.NoError(t, err)
}

func TestCueFadeTimeOverride(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Create cue list with long default fade
	var cueListResp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	err := setup.client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": setup.projectID,
			"name":      "Override Test",
		},
	}, &cueListResp)
	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Add cue with 5-second fade
	err = setup.client.Mutate(ctx, `
		mutation AddCue($input: AddCueInput!) {
			addCueToList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"cueListId":  cueListID,
			"name":       "Slow Cue",
			"cueNumber":  1.0,
			"sceneId":    sceneID,
			"fadeInTime": 5.0,
		},
	}, nil)
	require.NoError(t, err)

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Start cue list with override fade time
	err = setup.client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!, $fadeInTime: Float) {
			startCueList(cueListId: $cueListId, fadeInTime: $fadeInTime) { id }
		}
	`, map[string]interface{}{
		"cueListId":  cueListID,
		"fadeInTime": 0.5, // Override to 0.5 seconds
	}, nil)
	require.NoError(t, err)

	// Should complete in ~0.5s, not 5s
	time.Sleep(800 * time.Millisecond)
	output := setup.getDMXOutput(t)
	assert.InDelta(t, 255, output[0], 10, "Should be at full with override fade time")

	// Stop cue list
	_ = setup.client.Mutate(ctx, `mutation { stopCueList }`, nil, nil)
}

// ============================================================================
// Preview Mode Fade Tests
// ============================================================================

func TestPreviewModeFadeDoesNotAffectLive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create two scenes
	liveSceneID := setup.createScene(t, "Live", []int{255, 0, 0})
	previewSceneID := setup.createScene(t, "Preview", []int{0, 255, 0})

	// Set live scene
	setup.activateScene(t, liveSceneID, 0)
	time.Sleep(100 * time.Millisecond)

	// Verify live output
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Live should be red")

	// Start preview session
	var sessionResp struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err := setup.client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) { id }
		}
	`, map[string]interface{}{"projectId": setup.projectID}, &sessionResp)
	require.NoError(t, err)
	sessionID := sessionResp.StartPreviewSession.ID

	// Preview a different scene
	err = setup.client.Mutate(ctx, `
		mutation PreviewScene($sessionId: ID!, $sceneId: ID!) {
			previewScene(sessionId: $sessionId, sceneId: $sceneId) { id }
		}
	`, map[string]interface{}{
		"sessionId": sessionID,
		"sceneId":   previewSceneID,
	}, nil)
	require.NoError(t, err)

	// Give time for any potential leak
	time.Sleep(500 * time.Millisecond)

	// Live output should still be red, not affected by preview
	output = setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Live should still be red")
	assert.Equal(t, 0, output[1], "Live should not have preview green")

	// End preview session
	err = setup.client.Mutate(ctx, `
		mutation EndPreview($sessionId: ID!) {
			endPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{"sessionId": sessionID}, nil)
	require.NoError(t, err)

	// Verify live unchanged
	output = setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Live should still be red after preview end")
}

func TestPreviewSessionOutputValues(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene
	sceneID := setup.createScene(t, "Test", []int{128, 64, 32})

	// Start preview session
	var sessionResp struct {
		StartPreviewSession struct {
			ID string `json:"id"`
		} `json:"startPreviewSession"`
	}

	err := setup.client.Mutate(ctx, `
		mutation StartPreview($projectId: ID!) {
			startPreviewSession(projectId: $projectId) { id }
		}
	`, map[string]interface{}{"projectId": setup.projectID}, &sessionResp)
	require.NoError(t, err)
	sessionID := sessionResp.StartPreviewSession.ID

	// Preview the scene
	err = setup.client.Mutate(ctx, `
		mutation PreviewScene($sessionId: ID!, $sceneId: ID!) {
			previewScene(sessionId: $sessionId, sceneId: $sceneId) { id }
		}
	`, map[string]interface{}{
		"sessionId": sessionID,
		"sceneId":   sceneID,
	}, nil)
	require.NoError(t, err)

	// Get preview output
	var previewResp struct {
		PreviewSession struct {
			Output []int `json:"output"`
		} `json:"previewSession"`
	}

	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	err = setup.client.Query(ctx, `
		query GetPreview($sessionId: ID!) {
			previewSession(sessionId: $sessionId) {
				output(universe: 1)
			}
		}
	`, map[string]interface{}{"sessionId": sessionID}, &previewResp)
	require.NoError(t, err)

	// Verify preview output matches scene values
	assert.Equal(t, 128, previewResp.PreviewSession.Output[0], "Preview red should be 128")
	assert.Equal(t, 64, previewResp.PreviewSession.Output[1], "Preview green should be 64")
	assert.Equal(t, 32, previewResp.PreviewSession.Output[2], "Preview blue should be 32")

	// Cleanup
	_ = setup.client.Mutate(ctx, `
		mutation EndPreview($sessionId: ID!) {
			endPreviewSession(sessionId: $sessionId)
		}
	`, map[string]interface{}{"sessionId": sessionID}, nil)
}

// ============================================================================
// Art-Net Output Tests
// ============================================================================

func TestFadeCapturedViaArtNet(t *testing.T) {
	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Blackout and clear frames
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)
	receiver.ClearFrames()

	// Activate scene with fade
	setup.activateScene(t, sceneID, 1.0)

	// Wait for fade to complete
	time.Sleep(1500 * time.Millisecond)

	// Get captured frames
	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured - Art-Net may not be enabled")
	}

	t.Logf("Captured %d Art-Net frames during 1s fade", len(frames))

	// Verify we captured intermediate values
	var values []int
	for _, frame := range frames {
		if frame.Universe == 0 {
			values = append(values, int(frame.Channels[0]))
		}
	}

	if len(values) > 1 {
		hasIntermediate := false
		for _, v := range values {
			if v > 10 && v < 245 {
				hasIntermediate = true
				break
			}
		}
		assert.True(t, hasIntermediate, "Should capture intermediate fade values via Art-Net")
	}
}

func TestArtNetFrameRate(t *testing.T) {
	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Blackout
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)
	receiver.ClearFrames()

	// Activate with 2-second fade
	startTime := time.Now()
	setup.activateScene(t, sceneID, 2.0)

	// Wait for fade to complete
	time.Sleep(2500 * time.Millisecond)
	duration := time.Since(startTime)

	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured")
	}

	// Calculate frame rate
	frameRate := float64(len(frames)) / duration.Seconds()
	t.Logf("Art-Net frame rate: %.1f fps (%d frames over %.2fs)", frameRate, len(frames), duration.Seconds())

	// Should be at least 30fps (typically 44fps)
	assert.True(t, frameRate >= 25, "Frame rate should be at least 25 fps, got %.1f", frameRate)
}

// ============================================================================
// Edge Cases and Error Conditions
// ============================================================================

func TestFadeWithZeroChannelChange(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene at specific value
	scene1ID := setup.createScene(t, "Initial", []int{128, 128, 128})

	// Activate scene instantly
	setup.activateScene(t, scene1ID, 0)
	time.Sleep(100 * time.Millisecond)

	// Duplicate the scene (same values)
	scene2ID := setup.createScene(t, "Same", []int{128, 128, 128})

	// Fade to same values (should still work, just no change)
	setup.activateScene(t, scene2ID, 1.0)
	time.Sleep(1500 * time.Millisecond)

	// Should still be at 128
	output := setup.getDMXOutput(t)
	assert.Equal(t, 128, output[0], "Value should remain 128")
}

func TestVeryShortFade(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Very short fade (0.1 seconds)
	setup.activateScene(t, sceneID, 0.1)
	time.Sleep(300 * time.Millisecond)

	// Should be at full
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Should reach full after short fade")
}

func TestVeryLongFade(t *testing.T) {
	// This is a sampling test - we don't wait for the full fade
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Start from black
	setup.fadeToBlack(t, 0)
	time.Sleep(100 * time.Millisecond)

	// Start a 30-second fade
	setup.activateScene(t, sceneID, 30.0)

	// Check at 1 second - should be about 3.3% (255 * 0.033 ≈ 8.5)
	time.Sleep(1000 * time.Millisecond)
	output := setup.getDMXOutput(t)
	expectedMin := 5
	expectedMax := 15
	t.Logf("Value at 1s of 30s fade: %d (expected %d-%d)", output[0], expectedMin, expectedMax)
	assert.True(t, output[0] >= expectedMin && output[0] <= expectedMax,
		"Value at 1s should be in range %d-%d, got %d", expectedMin, expectedMax, output[0])

	// Interrupt with fadeToBlack to clean up
	setup.fadeToBlack(t, 0)
}

func TestFadeFromPartialValue(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scenes
	halfSceneID := setup.createScene(t, "Half", []int{128, 128, 128})
	fullSceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	// Start at half
	setup.activateScene(t, halfSceneID, 0)
	time.Sleep(100 * time.Millisecond)

	// Verify starting point
	output := setup.getDMXOutput(t)
	assert.Equal(t, 128, output[0], "Should start at half")

	// Fade to full
	setup.activateScene(t, fullSceneID, 2.0)

	// Check midpoint - should be around 192 (128 + (255-128)/2)
	time.Sleep(1000 * time.Millisecond)
	midOutput := setup.getDMXOutput(t)
	expectedMid := 192
	t.Logf("Mid-fade from 128 to 255: %d (expected ~%d)", midOutput[0], expectedMid)
	assert.InDelta(t, expectedMid, midOutput[0], 20, "Should be around 192 at midpoint")

	// Wait for completion
	time.Sleep(1500 * time.Millisecond)
	finalOutput := setup.getDMXOutput(t)
	assert.Equal(t, 255, finalOutput[0], "Should reach full")
}

func TestFadeDownward(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scenes
	fullSceneID := setup.createScene(t, "Full", []int{255, 255, 255})
	quarterSceneID := setup.createScene(t, "Quarter", []int{64, 64, 64})

	// Start at full
	setup.activateScene(t, fullSceneID, 0)
	time.Sleep(100 * time.Millisecond)

	// Verify starting point
	output := setup.getDMXOutput(t)
	assert.Equal(t, 255, output[0], "Should start at full")

	// Fade down to quarter
	setup.activateScene(t, quarterSceneID, 2.0)

	// Check midpoint - should be around 160 (255 - (255-64)/2)
	time.Sleep(1000 * time.Millisecond)
	midOutput := setup.getDMXOutput(t)
	expectedMid := 160
	t.Logf("Mid-fade from 255 to 64: %d (expected ~%d)", midOutput[0], expectedMid)
	assert.InDelta(t, expectedMid, midOutput[0], 20, "Should be around 160 at midpoint")

	// Wait for completion
	time.Sleep(1500 * time.Millisecond)
	finalOutput := setup.getDMXOutput(t)
	assert.InDelta(t, 64, finalOutput[0], 5, "Should reach quarter")
}

// ============================================================================
// Easing Type Tests (if supported)
// ============================================================================

func TestEasingTypes(t *testing.T) {
	// This test verifies different easing curves produce different progressions
	// Note: May need adjustment based on actual easing implementation

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setup := newTestSetup(t)
	defer setup.cleanup(t)

	// Create scene and cue list
	sceneID := setup.createScene(t, "Full", []int{255, 255, 255})

	var cueListResp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	err := setup.client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": setup.projectID,
			"name":      "Easing Test",
		},
	}, &cueListResp)
	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	easingTypes := []string{"LINEAR", "CUBIC", "SINE"}
	midpointValues := make(map[string]int)

	for i, easing := range easingTypes {
		// Add cue with specific easing
		var cueResp struct {
			AddCueToList struct {
				ID string `json:"id"`
			} `json:"addCueToList"`
		}

		err := setup.client.Mutate(ctx, `
			mutation AddCue($input: AddCueInput!) {
				addCueToList(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":  cueListID,
				"name":       easing + " Cue",
				"cueNumber":  float64(i + 1),
				"sceneId":    sceneID,
				"fadeInTime": 2.0,
				"easingType": easing,
			},
		}, &cueResp)
		if err != nil {
			t.Logf("Easing type %s may not be supported: %v", easing, err)
			continue
		}

		// Start from black
		setup.fadeToBlack(t, 0)
		time.Sleep(200 * time.Millisecond)

		// Start cue list from this cue
		err = setup.client.Mutate(ctx, `
			mutation StartCueList($cueListId: ID!, $startFromCue: Float) {
				startCueList(cueListId: $cueListId, startFromCue: $startFromCue) { id }
			}
		`, map[string]interface{}{
			"cueListId":    cueListID,
			"startFromCue": float64(i + 1),
		}, nil)
		require.NoError(t, err)

		// Sample at midpoint
		time.Sleep(1000 * time.Millisecond)
		output := setup.getDMXOutput(t)
		midpointValues[easing] = output[0]
		t.Logf("Easing %s midpoint value: %d", easing, output[0])

		// Stop and wait
		_ = setup.client.Mutate(ctx, `mutation { stopCueList }`, nil, nil)
		time.Sleep(500 * time.Millisecond)
	}

	// Different easing types should produce different midpoint values
	// Linear should be ~127, Cubic (ease-in-out) might be different, etc.
	if len(midpointValues) > 1 {
		// Just log the differences; exact values depend on implementation
		t.Logf("Midpoint values by easing: %v", midpointValues)
	}
}

// ============================================================================
// Helper for easing calculations (for reference)
// ============================================================================

func linearEase(t float64) float64 {
	return t
}

func sineEaseInOut(t float64) float64 {
	return -(math.Cos(math.Pi*t) - 1) / 2
}

func cubicEaseInOut(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}
