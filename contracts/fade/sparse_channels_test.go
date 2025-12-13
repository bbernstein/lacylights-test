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

// sparseChannelTestSetup contains resources for sparse channel DMX tests.
type sparseChannelTestSetup struct {
	client       *graphql.Client
	projectID    string
	definitionID string
	fixtureID    string
	sceneBoardID string
	sceneIDs     map[string]string // name -> ID
	fixtureIDs   []string          // multiple fixture IDs for multi-fixture tests
}

// newSparseChannelTestSetup creates test fixtures with 4-channel DRGB (Dimmer, Red, Green, Blue) fixture.
func newSparseChannelTestSetup(t *testing.T) *sparseChannelTestSetup {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")
	resetDMXState(t, client)

	setup := &sparseChannelTestSetup{
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
		"input": map[string]interface{}{"name": "Sparse Channels DMX Test"},
	}, &projectResp)
	require.NoError(t, err)
	setup.projectID = projectResp.CreateProject.ID

	// Create fixture definition with 4 channels (Dimmer, Red, Green, Blue)
	modelName := fmt.Sprintf("DRGB Test %d", time.Now().UnixNano())
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
			"manufacturer": "Sparse Channel Test",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Red", "type": "RED", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Green", "type": "GREEN", "offset": 2, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Blue", "type": "BLUE", "offset": 3, "minValue": 0, "maxValue": 255, "defaultValue": 0},
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
			"name":         "Test DRGB Light",
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
			"name":            "Sparse Channels Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &boardResp)
	require.NoError(t, err)
	setup.sceneBoardID = boardResp.CreateSceneBoard.ID

	return setup
}

func (s *sparseChannelTestSetup) cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resetDMXState(t, s.client)

	_ = s.client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
		map[string]interface{}{"id": s.projectID}, nil)
	_ = s.client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
		map[string]interface{}{"id": s.definitionID}, nil)
}

func (s *sparseChannelTestSetup) createSparseScene(t *testing.T, name string, channels []map[string]interface{}) string {
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
					"fixtureId": s.fixtureID,
					"channels":  channels,
				},
			},
		},
	}, &resp)
	require.NoError(t, err)
	s.sceneIDs[name] = resp.CreateScene.ID
	return resp.CreateScene.ID
}

// createMultipleFixtures creates multiple fixture instances and stores their IDs in fixtureIDs.
func (s *sparseChannelTestSetup) createMultipleFixtures(t *testing.T, count int, startChannel int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.fixtureIDs = make([]string, count)

	for i := 0; i < count; i++ {
		var instanceResp struct {
			CreateFixtureInstance struct {
				ID string `json:"id"`
			} `json:"createFixtureInstance"`
		}

		err := s.client.Mutate(ctx, `
			mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
				createFixtureInstance(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":    s.projectID,
				"definitionId": s.definitionID,
				"name":         fmt.Sprintf("Light %d", i+1),
				"universe":     1,
				"startChannel": startChannel + (i * 9), // Separate fixtures by 9 DMX channels (leaving gaps between fixtures)
			},
		}, &instanceResp)
		require.NoError(t, err)
		s.fixtureIDs[i] = instanceResp.CreateFixtureInstance.ID
	}
}

// TestSparseChannelsDMXOutput tests that only specified channels are output to DMX.
// Channels not in the sparse array should not be modified.
func TestSparseChannelsDMXOutput(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	setup := newSparseChannelTestSetup(t)
	defer setup.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create Scene 1: Only set Dimmer (channel 0) to 255
	sceneOnlyDimmerID := setup.createSparseScene(t, "Only Dimmer", []map[string]interface{}{
		{"offset": 0, "value": 255}, // Dimmer only
	})

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

	receiver.ClearFrames()

	// Activate scene with only Dimmer set
	var activateResp struct {
		ActivateSceneFromBoard bool `json:"activateSceneFromBoard"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  sceneOnlyDimmerID,
		"fadeTime": 0.0, // Instant to avoid fade complexity
	}, &activateResp)
	require.NoError(t, err)
	assert.True(t, activateResp.ActivateSceneFromBoard)

	// Wait for DMX output
	time.Sleep(200 * time.Millisecond)

	// Get captured frames
	frames := receiver.GetFrames()

	if len(frames) < 1 {
		t.Skipf("Not enough Art-Net frames captured (%d), skipping DMX verification", len(frames))
	}

	t.Logf("Captured %d Art-Net frames", len(frames))

	// Verify only Dimmer channel was set, RGB channels should remain at 0
	lastFrame := frames[len(frames)-1]
	if lastFrame.Universe == 0 { // Universe 1 = index 0
		dimmer := lastFrame.Channels[0]
		red := lastFrame.Channels[1]
		green := lastFrame.Channels[2]
		blue := lastFrame.Channels[3]

		assert.Equal(t, uint8(255), dimmer, "Dimmer should be set to 255")
		assert.Equal(t, uint8(0), red, "Red should remain at 0 (not in sparse channels)")
		assert.Equal(t, uint8(0), green, "Green should remain at 0 (not in sparse channels)")
		assert.Equal(t, uint8(0), blue, "Blue should remain at 0 (not in sparse channels)")

		t.Logf("DMX values: Dimmer=%d, R=%d, G=%d, B=%d", dimmer, red, green, blue)
	}
}

// TestSparseChannelsExcludedRetainValues tests that channels excluded from sparse array
// retain their previous values during scene transitions.
func TestSparseChannelsExcludedRetainValues(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	setup := newSparseChannelTestSetup(t)
	defer setup.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Scene 1: Set all channels to known values
	sceneAllID := setup.createSparseScene(t, "All Channels", []map[string]interface{}{
		{"offset": 0, "value": 255}, // Dimmer
		{"offset": 1, "value": 200}, // Red
		{"offset": 2, "value": 150}, // Green
		{"offset": 3, "value": 100}, // Blue
	})

	// Scene 2: Only modify Dimmer, should preserve R, G, B
	sceneOnlyDimmerID := setup.createSparseScene(t, "Only Dimmer Changed", []map[string]interface{}{
		{"offset": 0, "value": 128}, // Dimmer changed to 128
		// R, G, B not specified - should retain previous values
	})

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	// Start from black
	var fadeResp struct {
		FadeToBlack bool `json:"fadeToBlack"`
	}
	err = setup.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, &fadeResp)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Activate Scene 1 (all channels set)
	var activateResp struct {
		ActivateSceneFromBoard bool `json:"activateSceneFromBoard"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  sceneAllID,
		"fadeTime": 0.0,
	}, &activateResp)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	receiver.ClearFrames()

	// Activate Scene 2 (only Dimmer specified)
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  sceneOnlyDimmerID,
		"fadeTime": 0.0,
	}, &activateResp)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	frames := receiver.GetFrames()
	if len(frames) < 1 {
		t.Skipf("Not enough Art-Net frames captured (%d), skipping DMX verification", len(frames))
	}

	t.Logf("Captured %d Art-Net frames after Scene 2 activation", len(frames))

	// Verify RGB retained previous values, only Dimmer changed
	lastFrame := frames[len(frames)-1]
	if lastFrame.Universe == 0 {
		dimmer := lastFrame.Channels[0]
		red := lastFrame.Channels[1]
		green := lastFrame.Channels[2]
		blue := lastFrame.Channels[3]

		assert.Equal(t, uint8(128), dimmer, "Dimmer should change to 128")
		assert.Equal(t, uint8(200), red, "Red should retain previous value of 200")
		assert.Equal(t, uint8(150), green, "Green should retain previous value of 150")
		assert.Equal(t, uint8(100), blue, "Blue should retain previous value of 100")

		t.Logf("DMX values after Scene 2: Dimmer=%d (changed), R=%d (retained), G=%d (retained), B=%d (retained)",
			dimmer, red, green, blue)
	}
}

// TestSparseChannelsFadeOnlySpecified tests that fade transitions only affect
// channels specified in the sparse array.
func TestSparseChannelsFadeOnlySpecified(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	setup := newSparseChannelTestSetup(t)
	defer setup.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Scene 1: All channels at high values
	scene1ID := setup.createSparseScene(t, "All High", []map[string]interface{}{
		{"offset": 0, "value": 255}, // Dimmer
		{"offset": 1, "value": 255}, // Red
		{"offset": 2, "value": 255}, // Green
		{"offset": 3, "value": 255}, // Blue
	})

	// Scene 2: Only fade Red to 0, other channels not specified (should retain)
	scene2ID := setup.createSparseScene(t, "Only Red Fades", []map[string]interface{}{
		{"offset": 1, "value": 0}, // Red fades to 0
		// Dimmer, Green, Blue not specified - should stay at 255
	})

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	// Activate Scene 1 (instant)
	var activateResp struct {
		ActivateSceneFromBoard bool `json:"activateSceneFromBoard"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  scene1ID,
		"fadeTime": 0.0,
	}, &activateResp)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	receiver.ClearFrames()

	// Activate Scene 2 with 1-second fade (only Red should fade)
	err = setup.client.Mutate(ctx, `
		mutation ActivateScene($boardId: ID!, $sceneId: ID!, $fadeTime: Float) {
			activateSceneFromBoard(sceneBoardId: $boardId, sceneId: $sceneId, fadeTimeOverride: $fadeTime)
		}
	`, map[string]interface{}{
		"boardId":  setup.sceneBoardID,
		"sceneId":  scene2ID,
		"fadeTime": 1.0,
	}, &activateResp)
	require.NoError(t, err)

	// Wait for fade to complete
	time.Sleep(1200 * time.Millisecond)

	frames := receiver.GetFrames()
	if len(frames) < 10 {
		t.Skipf("Not enough Art-Net frames captured (%d), skipping fade verification", len(frames))
	}

	t.Logf("Captured %d Art-Net frames during fade", len(frames))

	// Analyze frames to verify only Red is fading
	redChanged := false
	dimmerChanged := false
	greenChanged := false
	blueChanged := false

	for _, frame := range frames {
		if frame.Universe != 0 {
			continue
		}

		dimmer := frame.Channels[0]
		red := frame.Channels[1]
		green := frame.Channels[2]
		blue := frame.Channels[3]

		// Check if Red is changing (between 0 and 255)
		if red > 0 && red < 255 {
			redChanged = true
		}

		// Check if Dimmer, Green, Blue changed from 255
		if dimmer != 255 {
			dimmerChanged = true
		}
		if green != 255 {
			greenChanged = true
		}
		if blue != 255 {
			blueChanged = true
		}
	}

	assert.True(t, redChanged, "Red should fade from 255 to 0")
	assert.False(t, dimmerChanged, "Dimmer should stay at 255 (not in sparse channels)")
	assert.False(t, greenChanged, "Green should stay at 255 (not in sparse channels)")
	assert.False(t, blueChanged, "Blue should stay at 255 (not in sparse channels)")

	// Verify final state
	lastFrame := frames[len(frames)-1]
	if lastFrame.Universe == 0 {
		dimmer := lastFrame.Channels[0]
		red := lastFrame.Channels[1]
		green := lastFrame.Channels[2]
		blue := lastFrame.Channels[3]

		assert.Equal(t, uint8(255), dimmer, "Dimmer should remain at 255")
		assert.Equal(t, uint8(0), red, "Red should reach target of 0")
		assert.Equal(t, uint8(255), green, "Green should remain at 255")
		assert.Equal(t, uint8(255), blue, "Blue should remain at 255")

		t.Logf("Final DMX values: Dimmer=%d, R=%d, G=%d, B=%d", dimmer, red, green, blue)
	}
}

// TestSparseChannelsMultipleFixtures tests sparse channels with multiple fixtures.
func TestSparseChannelsMultipleFixtures(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping fade test: SKIP_FADE_TESTS is set")
	}

	setup := newSparseChannelTestSetup(t)
	defer setup.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create two fixture instances using the helper
	setup.createMultipleFixtures(t, 2, 1)
	fixture1ID := setup.fixtureIDs[0]
	fixture2ID := setup.fixtureIDs[1]

	// Create scene with different sparse channels for each fixture
	var sceneResp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err := setup.client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": setup.projectID,
			"name":      "Multi-Fixture Sparse Scene",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255}, // Fixture 1: Only Dimmer
					},
				},
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 1, "value": 200}, // Fixture 2: Only Red
					},
				},
			},
		},
	}, &sceneResp)
	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err = receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	// Activate scene
	var activateResp struct {
		SetSceneLive bool `json:"setSceneLive"`
	}
	err = setup.client.Mutate(ctx, `
		mutation SetSceneLive($sceneId: ID!) {
			setSceneLive(sceneId: $sceneId)
		}
	`, map[string]interface{}{"sceneId": sceneID}, &activateResp)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	frames := receiver.GetFrames()
	if len(frames) < 1 {
		t.Skipf("Not enough Art-Net frames captured (%d), skipping verification", len(frames))
	}

	lastFrame := frames[len(frames)-1]
	if lastFrame.Universe == 0 {
		// Fixture 1 (channels 1-4): Only Dimmer should be 255
		fixture1Dimmer := lastFrame.Channels[0]
		fixture1Red := lastFrame.Channels[1]

		// Fixture 2 (channels 10-13): Only Red should be 200
		fixture2Dimmer := lastFrame.Channels[9]
		fixture2Red := lastFrame.Channels[10]

		assert.Equal(t, uint8(255), fixture1Dimmer, "Fixture 1 Dimmer should be 255")
		assert.Equal(t, uint8(0), fixture1Red, "Fixture 1 Red should be 0 (not specified)")

		assert.Equal(t, uint8(0), fixture2Dimmer, "Fixture 2 Dimmer should be 0 (not specified)")
		assert.Equal(t, uint8(200), fixture2Red, "Fixture 2 Red should be 200")

		t.Logf("Fixture 1: Dimmer=%d, Red=%d", fixture1Dimmer, fixture1Red)
		t.Logf("Fixture 2: Dimmer=%d, Red=%d", fixture2Dimmer, fixture2Red)
	}
}
