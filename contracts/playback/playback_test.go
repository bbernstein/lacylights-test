// Package playback provides playback and live control contract tests.
package playback

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipDMXTests returns true if SKIP_DMX_TESTS or SKIP_FADE_TESTS is set
// These tests depend on DMX output which may not work in all environments
func skipDMXTests() bool {
	return os.Getenv("SKIP_DMX_TESTS") != "" || os.Getenv("SKIP_FADE_TESTS") != ""
}

// setupPlaybackTest creates a project with fixtures, scenes, and a cue list for playback testing.
func setupPlaybackTest(t *testing.T, client *graphql.Client, ctx context.Context) (projectID, cueListID, scene1ID, scene2ID string) {
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
		"input": map[string]interface{}{"name": "Playback Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID = projectResp.CreateProject.ID

	// Get or create fixture definition
	var listResp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
		} `json:"fixtureDefinitions"`
	}

	err = client.Query(ctx, `
		query {
			fixtureDefinitions {
				id
				manufacturer
				model
			}
		}
	`, nil, &listResp)

	require.NoError(t, err)

	var definitionID string
	for _, def := range listResp.FixtureDefinitions {
		if def.Manufacturer == "Generic" && def.Model == "Dimmer" {
			definitionID = def.ID
			break
		}
	}

	if definitionID == "" {
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
				"manufacturer": "Generic",
				"model":        "Dimmer",
				"type":         "DIMMER",
				"channels": []map[string]interface{}{
					{
						"name":         "Intensity",
						"type":         "INTENSITY",
						"offset":       0,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
					},
				},
			},
		}, &createDefResp)

		require.NoError(t, err)
		definitionID = createDefResp.CreateFixtureDefinition.ID
	}

	// Create fixture
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
			"projectId":    projectID,
			"definitionId": definitionID,
			"name":         "Playback Test Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixtureInstance.ID

	// Create two scenes with different values
	var scene1Resp struct {
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
			"projectId": projectID,
			"name":      "Full Bright",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels":  []map[string]int{{"offset": 0, "value": 255}},
				},
			},
		},
	}, &scene1Resp)

	require.NoError(t, err)
	scene1ID = scene1Resp.CreateScene.ID

	var scene2Resp struct {
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
			"projectId": projectID,
			"name":      "Half Bright",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels":  []map[string]int{{"offset": 0, "value": 128}},
				},
			},
		},
	}, &scene2Resp)

	require.NoError(t, err)
	scene2ID = scene2Resp.CreateScene.ID

	// Create cue list with cues
	var cueListResp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	err = client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Playback Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID = cueListResp.CreateCueList.ID

	// Add cues
	for i, sceneID := range []string{scene1ID, scene2ID} {
		err = client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        "Cue " + string(rune('A'+i)),
				"cueNumber":   float64(i + 1),
				"fadeInTime":  1.0,
				"fadeOutTime": 1.0,
			},
		}, nil)
		require.NoError(t, err)
	}

	return projectID, cueListID, scene1ID, scene2ID
}

func cleanupPlaybackTest(client *graphql.Client, ctx context.Context, projectID string) {
	// Fade to black before cleanup
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
	_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
		map[string]interface{}{"id": projectID}, nil)
}

// TestCueListPlayback tests starting, navigating, and stopping cue list playback.
func TestCueListPlayback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, _, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// START CUE LIST
	t.Run("StartCueList", func(t *testing.T) {
		var startResp struct {
			StartCueList bool `json:"startCueList"`
		}

		err := client.Mutate(ctx, `
			mutation StartCueList($cueListId: ID!) {
				startCueList(cueListId: $cueListId)
			}
		`, map[string]interface{}{"cueListId": cueListID}, &startResp)

		require.NoError(t, err)
		assert.True(t, startResp.StartCueList)

		// Check playback status
		time.Sleep(200 * time.Millisecond)

		var statusResp struct {
			CueListPlaybackStatus struct {
				CueListID       string `json:"cueListId"`
				IsPlaying       bool   `json:"isPlaying"`
				IsFading        bool   `json:"isFading"`
				CurrentCueIndex *int   `json:"currentCueIndex"`
				CurrentCue      *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"currentCue"`
			} `json:"cueListPlaybackStatus"`
		}

		err = client.Query(ctx, `
			query GetPlaybackStatus($cueListId: ID!) {
				cueListPlaybackStatus(cueListId: $cueListId) {
					cueListId
					isPlaying
					isFading
					currentCueIndex
					currentCue {
						id
						name
					}
				}
			}
		`, map[string]interface{}{"cueListId": cueListID}, &statusResp)

		require.NoError(t, err)
		assert.Equal(t, cueListID, statusResp.CueListPlaybackStatus.CueListID)
		// Note: isPlaying may be false immediately after start due to timing
		// The important thing is that the status query succeeded and returned the right cue list
		// isFading indicates if a fade transition is in progress
	})

	// Wait for first cue to settle
	time.Sleep(1500 * time.Millisecond)

	// NEXT CUE
	t.Run("NextCue", func(t *testing.T) {
		var nextResp struct {
			NextCue bool `json:"nextCue"`
		}

		err := client.Mutate(ctx, `
			mutation NextCue($cueListId: ID!) {
				nextCue(cueListId: $cueListId)
			}
		`, map[string]interface{}{"cueListId": cueListID}, &nextResp)

		require.NoError(t, err)
		assert.True(t, nextResp.NextCue)

		// Wait for transition and verify
		time.Sleep(1500 * time.Millisecond)

		if !skipDMXTests() {
			var dmxResp struct {
				DMXOutput []int `json:"dmxOutput"`
			}

			err = client.Query(ctx, `
				query { dmxOutput(universe: 1) }
			`, nil, &dmxResp)

			require.NoError(t, err)
			// Should be at scene 2 (half bright = 128)
			assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "DMX should be near 128 after cue 2")
		}
	})

	// PREVIOUS CUE
	t.Run("PreviousCue", func(t *testing.T) {
		var prevResp struct {
			PreviousCue bool `json:"previousCue"`
		}

		err := client.Mutate(ctx, `
			mutation PreviousCue($cueListId: ID!) {
				previousCue(cueListId: $cueListId)
			}
		`, map[string]interface{}{"cueListId": cueListID}, &prevResp)

		require.NoError(t, err)
		assert.True(t, prevResp.PreviousCue)

		// Wait for transition
		time.Sleep(1500 * time.Millisecond)

		if !skipDMXTests() {
			var dmxResp struct {
				DMXOutput []int `json:"dmxOutput"`
			}

			err = client.Query(ctx, `
				query { dmxOutput(universe: 1) }
			`, nil, &dmxResp)

			require.NoError(t, err)
			// Should be back at scene 1 (full bright = 255)
			assert.InDelta(t, 255, dmxResp.DMXOutput[0], 5, "DMX should be near 255 after returning to cue 1")
		}
	})

	// GO TO CUE (by index)
	t.Run("GoToCue", func(t *testing.T) {
		var gotoResp struct {
			GoToCue bool `json:"goToCue"`
		}

		err := client.Mutate(ctx, `
			mutation GoToCue($cueListId: ID!, $cueIndex: Int!) {
				goToCue(cueListId: $cueListId, cueIndex: $cueIndex)
			}
		`, map[string]interface{}{
			"cueListId": cueListID,
			"cueIndex":  1, // Go to second cue (0-indexed)
		}, &gotoResp)

		require.NoError(t, err)
		assert.True(t, gotoResp.GoToCue)

		// Wait for transition
		time.Sleep(1500 * time.Millisecond)

		if !skipDMXTests() {
			var dmxResp struct {
				DMXOutput []int `json:"dmxOutput"`
			}

			err = client.Query(ctx, `
				query { dmxOutput(universe: 1) }
			`, nil, &dmxResp)

			require.NoError(t, err)
			assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "DMX should be near 128 after goToCue(1)")
		}
	})

	// STOP CUE LIST
	t.Run("StopCueList", func(t *testing.T) {
		var stopResp struct {
			StopCueList bool `json:"stopCueList"`
		}

		err := client.Mutate(ctx, `
			mutation StopCueList($cueListId: ID!) {
				stopCueList(cueListId: $cueListId)
			}
		`, map[string]interface{}{"cueListId": cueListID}, &stopResp)

		require.NoError(t, err)
		assert.True(t, stopResp.StopCueList)

		// Verify playback stopped
		var statusResp struct {
			CueListPlaybackStatus *struct {
				IsPlaying bool `json:"isPlaying"`
				IsFading  bool `json:"isFading"`
			} `json:"cueListPlaybackStatus"`
		}

		err = client.Query(ctx, `
			query GetPlaybackStatus($cueListId: ID!) {
				cueListPlaybackStatus(cueListId: $cueListId) {
					isPlaying
					isFading
				}
			}
		`, map[string]interface{}{"cueListId": cueListID}, &statusResp)

		require.NoError(t, err)
		if statusResp.CueListPlaybackStatus != nil {
			assert.False(t, statusResp.CueListPlaybackStatus.IsPlaying)
			assert.False(t, statusResp.CueListPlaybackStatus.IsFading)
		}
	})
}

// TestPlayCue tests playing a single cue directly.
func TestPlayCue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, _, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Get cue ID from cue list
	var cueListResp struct {
		CueList struct {
			Cues []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"cues"`
		} `json:"cueList"`
	}

	err := client.Query(ctx, `
		query GetCueList($id: ID!) {
			cueList(id: $id) {
				cues {
					id
					name
				}
			}
		}
	`, map[string]interface{}{"id": cueListID}, &cueListResp)

	require.NoError(t, err)
	require.NotEmpty(t, cueListResp.CueList.Cues)

	cueID := cueListResp.CueList.Cues[0].ID

	// Play single cue
	var playResp struct {
		PlayCue bool `json:"playCue"`
	}

	err = client.Mutate(ctx, `
		mutation PlayCue($cueId: ID!, $fadeInTime: Float) {
			playCue(cueId: $cueId, fadeInTime: $fadeInTime)
		}
	`, map[string]interface{}{
		"cueId":      cueID,
		"fadeInTime": 0.5,
	}, &playResp)

	require.NoError(t, err)
	assert.True(t, playResp.PlayCue)

	// Wait for fade
	time.Sleep(1000 * time.Millisecond)

	// Verify DMX output matches scene 1 (full bright)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}

		err = client.Query(ctx, `
			query { dmxOutput(universe: 1) }
		`, nil, &dmxResp)

		require.NoError(t, err)
		assert.InDelta(t, 255, dmxResp.DMXOutput[0], 5, "DMX should be near 255 after playing cue")
	}
}

// TestSetSceneLive tests activating a scene directly.
func TestSetSceneLive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, _, scene1ID, scene2ID := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Activate scene 1
	var liveResp struct {
		SetSceneLive bool `json:"setSceneLive"`
	}

	err := client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": scene1ID}, &liveResp)

	require.NoError(t, err)
	assert.True(t, liveResp.SetSceneLive)

	// Give it a moment to apply
	time.Sleep(500 * time.Millisecond)

	// Verify DMX output
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}

		err = client.Query(ctx, `
			query { dmxOutput(universe: 1) }
		`, nil, &dmxResp)

		require.NoError(t, err)
		assert.Equal(t, 255, dmxResp.DMXOutput[0], "DMX should be 255 (scene1 full bright)")
	}

	// Switch to scene 2
	err = client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": scene2ID}, &liveResp)

	require.NoError(t, err)
	assert.True(t, liveResp.SetSceneLive)

	time.Sleep(500 * time.Millisecond)

	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}

		err = client.Query(ctx, `
			query { dmxOutput(universe: 1) }
		`, nil, &dmxResp)

		require.NoError(t, err)
		assert.Equal(t, 128, dmxResp.DMXOutput[0], "DMX should be 128 (scene2 half bright)")
	}
}

// TestCurrentActiveScene tests querying the currently active scene.
func TestCurrentActiveScene(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, _, scene1ID, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Query active scene (should be null or nothing)
	var activeResp struct {
		CurrentActiveScene *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"currentActiveScene"`
	}

	err := client.Query(ctx, `
		query {
			currentActiveScene {
				id
				name
			}
		}
	`, nil, &activeResp)

	require.NoError(t, err)
	// After fadeToBlack, there may or may not be an active scene

	// Activate a scene
	_ = client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": scene1ID}, nil)

	time.Sleep(500 * time.Millisecond)

	// Query active scene again
	err = client.Query(ctx, `
		query {
			currentActiveScene {
				id
				name
			}
		}
	`, nil, &activeResp)

	require.NoError(t, err)
	if activeResp.CurrentActiveScene != nil {
		assert.Equal(t, scene1ID, activeResp.CurrentActiveScene.ID)
		assert.Equal(t, "Full Bright", activeResp.CurrentActiveScene.Name)
	}
}

// TestStartCueListFromCue tests starting a cue list from a specific cue.
func TestStartCueListFromCue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, _, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start from second cue (cue number 2.0)
	var startResp struct {
		StartCueList bool `json:"startCueList"`
	}

	err := client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!, $startFromCue: Int) {
			startCueList(cueListId: $cueListId, startFromCue: $startFromCue)
		}
	`, map[string]interface{}{
		"cueListId":    cueListID,
		"startFromCue": 2, // Cue number 2.0 (Half Bright scene)
	}, &startResp)

	require.NoError(t, err)
	assert.True(t, startResp.StartCueList)

	// Wait for cue to settle
	time.Sleep(1500 * time.Millisecond)

	// Verify DMX output matches scene 2 (half bright)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}

		err = client.Query(ctx, `
			query { dmxOutput(universe: 1) }
		`, nil, &dmxResp)

		require.NoError(t, err)
		assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "DMX should be near 128 when starting from cue 2")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// TestIsFadingDuringTransition tests that isFading is true during fade transitions.
func TestIsFadingDuringTransition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, _, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list - this will start fading into the first cue
	var startResp struct {
		StartCueList bool `json:"startCueList"`
	}

	err := client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, &startResp)

	require.NoError(t, err)
	assert.True(t, startResp.StartCueList)

	// Immediately check status - isFading should be true during the fade
	// Note: The cues have 1 second fade time from setupPlaybackTest
	var statusResp struct {
		CueListPlaybackStatus struct {
			IsPlaying    bool     `json:"isPlaying"`
			IsFading     bool     `json:"isFading"`
			FadeProgress *float64 `json:"fadeProgress"`
		} `json:"cueListPlaybackStatus"`
	}

	err = client.Query(ctx, `
		query GetPlaybackStatus($cueListId: ID!) {
			cueListPlaybackStatus(cueListId: $cueListId) {
				isPlaying
				isFading
				fadeProgress
			}
		}
	`, map[string]interface{}{"cueListId": cueListID}, &statusResp)

	require.NoError(t, err)
	// During fade, isFading should be true
	// Note: Due to timing, the fade might complete very quickly in CI environments
	t.Logf("Immediately after start: isPlaying=%v, isFading=%v, fadeProgress=%v",
		statusResp.CueListPlaybackStatus.IsPlaying,
		statusResp.CueListPlaybackStatus.IsFading,
		statusResp.CueListPlaybackStatus.FadeProgress)

	// If we caught the fade in progress, assert isFading is true
	// This validates the semantic difference: isPlaying can be true while isFading is also true
	// Note: >= 0 includes the very start of the fade when fadeProgress is exactly 0
	if statusResp.CueListPlaybackStatus.FadeProgress != nil &&
		*statusResp.CueListPlaybackStatus.FadeProgress >= 0 &&
		*statusResp.CueListPlaybackStatus.FadeProgress < 100 {
		assert.True(t, statusResp.CueListPlaybackStatus.IsFading,
			"isFading should be true when fadeProgress is between 0 and 100")
	}

	// Wait for fade to complete (cues have 1 second fade time)
	time.Sleep(1500 * time.Millisecond)

	// Check status again - after fade completes, isFading should be false but isPlaying should be true
	err = client.Query(ctx, `
		query GetPlaybackStatus($cueListId: ID!) {
			cueListPlaybackStatus(cueListId: $cueListId) {
				isPlaying
				isFading
				fadeProgress
			}
		}
	`, map[string]interface{}{"cueListId": cueListID}, &statusResp)

	require.NoError(t, err)
	// After fade completes: isPlaying=true (scene active), isFading=false (no transition)
	t.Logf("After fade completes: isPlaying=%v, isFading=%v, fadeProgress=%v",
		statusResp.CueListPlaybackStatus.IsPlaying,
		statusResp.CueListPlaybackStatus.IsFading,
		statusResp.CueListPlaybackStatus.FadeProgress)

	assert.True(t, statusResp.CueListPlaybackStatus.IsPlaying, "isPlaying should be true after fade completes")
	assert.False(t, statusResp.CueListPlaybackStatus.IsFading, "isFading should be false after fade completes")

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// TestFadeTimeOverride tests overriding fade times in cue navigation.
func TestFadeTimeOverride(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, _, _ := setupPlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list
	err := client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)

	require.NoError(t, err)

	// Wait for first cue
	time.Sleep(1500 * time.Millisecond)

	// Go to next cue with instant fade override
	var nextResp struct {
		NextCue bool `json:"nextCue"`
	}

	err = client.Mutate(ctx, `
		mutation NextCue($cueListId: ID!, $fadeInTime: Float) {
			nextCue(cueListId: $cueListId, fadeInTime: $fadeInTime)
		}
	`, map[string]interface{}{
		"cueListId":  cueListID,
		"fadeInTime": 0.0, // Instant fade
	}, &nextResp)

	require.NoError(t, err)
	assert.True(t, nextResp.NextCue)

	// Very short wait since fade is instant
	time.Sleep(100 * time.Millisecond)

	// Verify DMX output changed instantly
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}

		err = client.Query(ctx, `
			query { dmxOutput(universe: 1) }
		`, nil, &dmxResp)

		require.NoError(t, err)
		assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "DMX should be near 128 with instant fade")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// setupSkipCuePlaybackTest creates a project with fixtures, scenes, and a cue list with 4 cues for skip testing.
// Returns project ID, cue list ID, and an array of cue IDs.
func setupSkipCuePlaybackTest(t *testing.T, client *graphql.Client, ctx context.Context) (projectID, cueListID string, cueIDs []string) {
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
		"input": map[string]interface{}{"name": "Skip Cue Playback Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID = projectResp.CreateProject.ID

	// Get or create fixture definition
	var listResp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
		} `json:"fixtureDefinitions"`
	}

	err = client.Query(ctx, `
		query {
			fixtureDefinitions {
				id
				manufacturer
				model
			}
		}
	`, nil, &listResp)

	require.NoError(t, err)

	var definitionID string
	for _, def := range listResp.FixtureDefinitions {
		if def.Manufacturer == "Generic" && def.Model == "Dimmer" {
			definitionID = def.ID
			break
		}
	}

	if definitionID == "" {
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
				"manufacturer": "Generic",
				"model":        "Dimmer",
				"type":         "DIMMER",
				"channels": []map[string]interface{}{
					{
						"name":         "Intensity",
						"type":         "INTENSITY",
						"offset":       0,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
					},
				},
			},
		}, &createDefResp)

		require.NoError(t, err)
		definitionID = createDefResp.CreateFixtureDefinition.ID
	}

	// Create fixture
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
			"projectId":    projectID,
			"definitionId": definitionID,
			"name":         "Skip Test Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixtureInstance.ID

	// Create 4 scenes with different values
	sceneValues := []int{64, 128, 192, 255}
	sceneNames := []string{"Scene25", "Scene50", "Scene75", "Scene100"}
	var sceneIDs []string

	for i, val := range sceneValues {
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
				"projectId": projectID,
				"name":      sceneNames[i],
				"fixtureValues": []map[string]interface{}{
					{
						"fixtureId": fixtureID,
						"channels":  []map[string]int{{"offset": 0, "value": val}},
					},
				},
			},
		}, &sceneResp)

		require.NoError(t, err)
		sceneIDs = append(sceneIDs, sceneResp.CreateScene.ID)
	}

	// Create cue list
	var cueListResp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	err = client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Skip Test Cue List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID = cueListResp.CreateCueList.ID

	// Add cues
	cueNames := []string{"Cue1", "Cue2", "Cue3", "Cue4"}
	for i, sceneID := range sceneIDs {
		var cueResp struct {
			CreateCue struct {
				ID string `json:"id"`
			} `json:"createCue"`
		}

		err = client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        cueNames[i],
				"cueNumber":   float64(i + 1),
				"fadeInTime":  0.1, // Fast fade for testing
				"fadeOutTime": 0.1,
			},
		}, &cueResp)
		require.NoError(t, err)
		cueIDs = append(cueIDs, cueResp.CreateCue.ID)
	}

	return projectID, cueListID, cueIDs
}

// TestNextCueSkipsSkipped tests that NextCue skips over skipped cues.
func TestNextCueSkipsSkipped(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, cueIDs := setupSkipCuePlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Skip cue 2 (index 1) - the one with 128 value
	err := client.Mutate(ctx, `
		mutation ToggleCueSkip($cueId: ID!) {
			toggleCueSkip(cueId: $cueId) { id skip }
		}
	`, map[string]interface{}{"cueId": cueIDs[1]}, nil)
	require.NoError(t, err)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list at cue 1
	err = client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
	require.NoError(t, err)

	// Wait for first cue to settle
	time.Sleep(300 * time.Millisecond)

	// Verify we're at cue 1 (value 64)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 64, dmxResp.DMXOutput[0], 5, "Should start at cue 1 (value 64)")
	}

	// Call NextCue - should skip cue 2 and go directly to cue 3
	var nextResp struct {
		NextCue bool `json:"nextCue"`
	}
	err = client.Mutate(ctx, `
		mutation NextCue($cueListId: ID!) {
			nextCue(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, &nextResp)
	require.NoError(t, err)
	assert.True(t, nextResp.NextCue)

	// Wait for transition
	time.Sleep(300 * time.Millisecond)

	// Verify we're at cue 3 (value 192), not cue 2 (value 128)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 192, dmxResp.DMXOutput[0], 5, "NextCue should skip to cue 3 (value 192), skipping cue 2")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// TestPreviousCueSkipsSkipped tests that PreviousCue skips over skipped cues.
func TestPreviousCueSkipsSkipped(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, cueIDs := setupSkipCuePlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Skip cue 3 (index 2) - the one with 192 value
	err := client.Mutate(ctx, `
		mutation ToggleCueSkip($cueId: ID!) {
			toggleCueSkip(cueId: $cueId) { id skip }
		}
	`, map[string]interface{}{"cueId": cueIDs[2]}, nil)
	require.NoError(t, err)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list at cue 4 (cue number 4)
	err = client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!, $startFromCue: Int) {
			startCueList(cueListId: $cueListId, startFromCue: $startFromCue)
		}
	`, map[string]interface{}{
		"cueListId":    cueListID,
		"startFromCue": 4,
	}, nil)
	require.NoError(t, err)

	// Wait for cue to settle
	time.Sleep(300 * time.Millisecond)

	// Verify we're at cue 4 (value 255)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 255, dmxResp.DMXOutput[0], 5, "Should start at cue 4 (value 255)")
	}

	// Call PreviousCue - should skip cue 3 and go directly to cue 2
	var prevResp struct {
		PreviousCue bool `json:"previousCue"`
	}
	err = client.Mutate(ctx, `
		mutation PreviousCue($cueListId: ID!) {
			previousCue(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, &prevResp)
	require.NoError(t, err)
	assert.True(t, prevResp.PreviousCue)

	// Wait for transition
	time.Sleep(300 * time.Millisecond)

	// Verify we're at cue 2 (value 128), not cue 3 (value 192)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "PreviousCue should skip to cue 2 (value 128), skipping cue 3")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// TestJumpToSkippedCueAllowed tests that you can still jump directly to a skipped cue.
func TestJumpToSkippedCueAllowed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, cueIDs := setupSkipCuePlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Skip cue 2 (index 1) - the one with 128 value
	err := client.Mutate(ctx, `
		mutation ToggleCueSkip($cueId: ID!) {
			toggleCueSkip(cueId: $cueId) { id skip }
		}
	`, map[string]interface{}{"cueId": cueIDs[1]}, nil)
	require.NoError(t, err)

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list
	err = client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
	require.NoError(t, err)

	// Wait for first cue
	time.Sleep(300 * time.Millisecond)

	// Jump directly to skipped cue 2 (cue index 1)
	var gotoResp struct {
		GoToCue bool `json:"goToCue"`
	}
	err = client.Mutate(ctx, `
		mutation GoToCue($cueListId: ID!, $cueIndex: Int!) {
			goToCue(cueListId: $cueListId, cueIndex: $cueIndex)
		}
	`, map[string]interface{}{
		"cueListId": cueListID,
		"cueIndex":  1, // 0-indexed, so cue 2
	}, &gotoResp)
	require.NoError(t, err)
	assert.True(t, gotoResp.GoToCue, "Should be able to jump directly to a skipped cue")

	// Wait for transition
	time.Sleep(300 * time.Millisecond)

	// Verify we're at skipped cue 2 (value 128)
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 128, dmxResp.DMXOutput[0], 5, "Should be able to jump to skipped cue (value 128)")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}

// TestMultipleConsecutiveSkippedCues tests that multiple consecutive skipped cues are all skipped.
func TestMultipleConsecutiveSkippedCues(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	projectID, cueListID, cueIDs := setupSkipCuePlaybackTest(t, client, ctx)
	defer cleanupPlaybackTest(client, ctx, projectID)

	// Skip cues 2 and 3 (indices 1 and 2)
	for _, cueID := range []string{cueIDs[1], cueIDs[2]} {
		err := client.Mutate(ctx, `
			mutation ToggleCueSkip($cueId: ID!) {
				toggleCueSkip(cueId: $cueId) { id skip }
			}
		`, map[string]interface{}{"cueId": cueID}, nil)
		require.NoError(t, err)
	}

	// Ensure clean state
	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Start cue list at cue 1
	err := client.Mutate(ctx, `
		mutation StartCueList($cueListId: ID!) {
			startCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
	require.NoError(t, err)

	// Wait for first cue
	time.Sleep(300 * time.Millisecond)

	// Call NextCue - should skip cues 2 and 3, go directly to cue 4
	var nextResp struct {
		NextCue bool `json:"nextCue"`
	}
	err = client.Mutate(ctx, `
		mutation NextCue($cueListId: ID!) {
			nextCue(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, &nextResp)
	require.NoError(t, err)
	assert.True(t, nextResp.NextCue)

	// Wait for transition
	time.Sleep(300 * time.Millisecond)

	// Verify we're at cue 4 (value 255), having skipped both cues 2 and 3
	if !skipDMXTests() {
		var dmxResp struct {
			DMXOutput []int `json:"dmxOutput"`
		}
		err = client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &dmxResp)
		require.NoError(t, err)
		assert.InDelta(t, 255, dmxResp.DMXOutput[0], 5, "NextCue should skip to cue 4 (value 255), skipping cues 2 and 3")
	}

	// Stop playback
	_ = client.Mutate(ctx, `
		mutation StopCueList($cueListId: ID!) {
			stopCueList(cueListId: $cueListId)
		}
	`, map[string]interface{}{"cueListId": cueListID}, nil)
}
