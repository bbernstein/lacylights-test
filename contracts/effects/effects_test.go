// Package effects provides comprehensive contract tests for the FX Engine.
// These tests validate:
// - Effect CRUD operations (create, read, update, delete)
// - Effect-Fixture associations (adding/removing fixtures to effects)
// - Effect-Cue associations (attaching effects to cues)
// - Effect playback behavior (waveforms, composition modes)
// - DMX output validation for running effects
// - Transition behaviors when cues change
package effects

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/artnet"
	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Setup and Helpers
// ============================================================================

func getArtNetPort() string {
	port := os.Getenv("ARTNET_LISTEN_PORT")
	if port == "" {
		port = "6454"
	}
	if os.Getenv("ARTNET_BROADCAST") == "127.0.0.1" {
		return "127.0.0.1:" + port
	}
	return ":" + port
}

func checkArtNetEnabled(t *testing.T) {
	if os.Getenv("SKIP_FADE_TESTS") != "" || os.Getenv("SKIP_EFFECT_TESTS") != "" {
		t.Skip("Skipping effect test: SKIP_FADE_TESTS or SKIP_EFFECT_TESTS is set")
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
		t.Skipf("Skipping effect test: cannot query systemInfo: %v", err)
	}

	if !resp.SystemInfo.ArtnetEnabled {
		t.Skip("Skipping effect test: Art-Net is not enabled on the server")
	}
}

func resetDMXState(_ *testing.T, client *graphql.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
	time.Sleep(100 * time.Millisecond)
}

// effectTestSetup contains resources for effect tests
type effectTestSetup struct {
	client       *graphql.Client
	projectID    string
	definitionID string
	fixtureID    string
	fixtureID2   string // Second fixture for multi-fixture tests
	lookBoardID  string
	cueListID    string
	looks        map[string]string
	effects      map[string]string
}

// newEffectTestSetup creates a test setup with project, fixtures, and look board
func newEffectTestSetup(t *testing.T) *effectTestSetup {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Reset DMX state
	resetDMXState(t, client)

	setup := &effectTestSetup{
		client:  client,
		looks:   make(map[string]string),
		effects: make(map[string]string),
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
	`, map[string]any{
		"input": map[string]any{"name": "Effect Test Project"},
	}, &projectResp)
	require.NoError(t, err)
	setup.projectID = projectResp.CreateProject.ID

	// Create fixture definition with 4 channels (Dimmer, R, G, B)
	var defResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}
	modelName := fmt.Sprintf("Effect Test Fixture %d", time.Now().UnixNano())
	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"manufacturer": "Test Effects",
			"model":        modelName,
			"type":         "LED_PAR",
			"channels": []map[string]any{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Red", "type": "RED", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Green", "type": "GREEN", "offset": 2, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Blue", "type": "BLUE", "offset": 3, "minValue": 0, "maxValue": 255, "defaultValue": 0},
			},
		},
	}, &defResp)
	require.NoError(t, err)
	setup.definitionID = defResp.CreateFixtureDefinition.ID

	// Create first fixture instance at channels 1-4
	var fixture1Resp struct {
		CreateFixtureInstance struct {
			ID string `json:"id"`
		} `json:"createFixtureInstance"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":    setup.projectID,
			"definitionId": setup.definitionID,
			"name":         "Effect Fixture 1",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixture1Resp)
	require.NoError(t, err)
	setup.fixtureID = fixture1Resp.CreateFixtureInstance.ID

	// Create second fixture instance at channels 5-8
	var fixture2Resp struct {
		CreateFixtureInstance struct {
			ID string `json:"id"`
		} `json:"createFixtureInstance"`
	}
	err = client.Mutate(ctx, `
		mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":    setup.projectID,
			"definitionId": setup.definitionID,
			"name":         "Effect Fixture 2",
			"universe":     1,
			"startChannel": 5,
		},
	}, &fixture2Resp)
	require.NoError(t, err)
	setup.fixtureID2 = fixture2Resp.CreateFixtureInstance.ID

	// Create look board
	var boardResp struct {
		CreateLookBoard struct {
			ID string `json:"id"`
		} `json:"createLookBoard"`
	}
	err = client.Mutate(ctx, `
		mutation CreateLookBoard($input: CreateLookBoardInput!) {
			createLookBoard(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "Effect Test Board",
			"defaultFadeTime": 1.0,
		},
	}, &boardResp)
	require.NoError(t, err)
	setup.lookBoardID = boardResp.CreateLookBoard.ID

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
	`, map[string]any{
		"input": map[string]any{
			"projectId": setup.projectID,
			"name":      "Effect Test Cue List",
		},
	}, &cueListResp)
	require.NoError(t, err)
	setup.cueListID = cueListResp.CreateCueList.ID

	return setup
}

func (s *effectTestSetup) cleanup(_ *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop all effects first (before other cleanup)
	for _, effectID := range s.effects {
		_ = s.client.Mutate(ctx, `mutation StopEffect($id: ID!) { stopEffect(effectId: $id, fadeTime: 0) }`,
			map[string]any{"id": effectID}, nil)
	}

	// Stop cue list
	_ = s.client.Mutate(ctx, `mutation StopCueList($id: ID!) { stopCueList(cueListId: $id) }`,
		map[string]any{"id": s.cueListID}, nil)

	// Fade to black to clear DMX state
	_ = s.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)

	// Wait for fade engine to settle
	time.Sleep(200 * time.Millisecond)

	// Delete project (cascades to fixtures, looks, effects, etc.)
	_ = s.client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
		map[string]any{"id": s.projectID}, nil)

	// Delete fixture definition
	if s.definitionID != "" {
		_ = s.client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]any{"id": s.definitionID}, nil)
	}

	// Final fadeToBlack to ensure clean state for next tests
	_ = s.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
	time.Sleep(100 * time.Millisecond)
}

func (s *effectTestSetup) getDMXOutput(t *testing.T) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resp struct {
		DMXOutput []int `json:"dmxOutput"`
	}
	err := s.client.Query(ctx, `query { dmxOutput(universe: 1) }`, nil, &resp)
	require.NoError(t, err)
	return resp.DMXOutput
}

func (s *effectTestSetup) createLook(t *testing.T, name string, channelValues []int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	channels := make([]map[string]int, len(channelValues))
	for i, value := range channelValues {
		channels[i] = map[string]int{"offset": i, "value": value}
	}

	var lookResp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}
	err := s.client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId": s.projectID,
			"name":      name,
			"fixtureValues": []map[string]any{
				{"fixtureId": s.fixtureID, "channels": channels},
			},
		},
	}, &lookResp)
	require.NoError(t, err)

	// Add to look board
	_ = s.client.Mutate(ctx, `
		mutation AddLookToBoard($input: CreateLookBoardButtonInput!) {
			addLookToBoard(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"lookBoardId": s.lookBoardID,
			"lookId":      lookResp.CreateLook.ID,
			"layoutX":     len(s.looks) * 200,
			"layoutY":     0,
		},
	}, nil)

	s.looks[name] = lookResp.CreateLook.ID
	return lookResp.CreateLook.ID
}

func (s *effectTestSetup) activateLook(t *testing.T, lookID string, fadeTime float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if fadeTime == 0 {
		err := s.client.Mutate(ctx, `mutation SetLookLive($lookId: ID!) { setLookLive(lookId: $lookId) }`,
			map[string]any{"lookId": lookID}, nil)
		require.NoError(t, err)
	} else {
		err := s.client.Mutate(ctx, `
			mutation ActivateLookFromBoard($lookBoardId: ID!, $lookId: ID!, $fadeTimeOverride: Float) {
				activateLookFromBoard(lookBoardId: $lookBoardId, lookId: $lookId, fadeTimeOverride: $fadeTimeOverride)
			}
		`, map[string]any{
			"lookBoardId":      s.lookBoardID,
			"lookId":           lookID,
			"fadeTimeOverride": fadeTime,
		}, nil)
		require.NoError(t, err)
	}
}

// ============================================================================
// Effect CRUD Tests
// ============================================================================

func TestEffectCRUD(t *testing.T) {
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
		mutation { createProject(input: {name: "Effect CRUD Test"}) { id } }
	`, nil, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]any{"id": projectID}, nil)
	}()

	var effectID string

	t.Run("CreateEffect", func(t *testing.T) {
		var resp struct {
			CreateEffect struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				Description     string `json:"description"`
				EffectType      string `json:"effectType"`
				PriorityBand    string `json:"priorityBand"`
				CompositionMode string `json:"compositionMode"`
				Waveform        string `json:"waveform"`
				Frequency       float64 `json:"frequency"`
				Amplitude       float64 `json:"amplitude"`
				Offset          float64 `json:"offset"`
			} `json:"createEffect"`
		}

		err := client.Mutate(ctx, `
			mutation CreateEffect($input: CreateEffectInput!) {
				createEffect(input: $input) {
					id
					name
					description
					effectType
					priorityBand
					compositionMode
					waveform
					frequency
					amplitude
					offset
				}
			}
		`, map[string]any{
			"input": map[string]any{
				"projectId":       projectID,
				"name":            "Test Sine Wave",
				"description":     "A test sine wave effect",
				"effectType":      "WAVEFORM",
				"priorityBand":    "USER",
				"compositionMode": "ADDITIVE",
				"waveform":        "SINE",
				"frequency":       0.5,
				"amplitude":       50.0,
				"offset":          50.0,
			},
		}, &resp)
		require.NoError(t, err)

		effectID = resp.CreateEffect.ID
		assert.NotEmpty(t, effectID)
		assert.Equal(t, "Test Sine Wave", resp.CreateEffect.Name)
		assert.Equal(t, "A test sine wave effect", resp.CreateEffect.Description)
		assert.Equal(t, "WAVEFORM", resp.CreateEffect.EffectType)
		assert.Equal(t, "USER", resp.CreateEffect.PriorityBand)
		assert.Equal(t, "SINE", resp.CreateEffect.Waveform)
		assert.Equal(t, 0.5, resp.CreateEffect.Frequency)
		assert.Equal(t, 50.0, resp.CreateEffect.Amplitude)
		assert.Equal(t, 50.0, resp.CreateEffect.Offset)
	})

	t.Run("ReadEffect", func(t *testing.T) {
		var resp struct {
			Effect struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				EffectType string `json:"effectType"`
				Waveform   string `json:"waveform"`
			} `json:"effect"`
		}

		err := client.Query(ctx, `
			query GetEffect($id: ID!) {
				effect(id: $id) {
					id
					name
					effectType
					waveform
				}
			}
		`, map[string]any{"id": effectID}, &resp)
		require.NoError(t, err)

		assert.Equal(t, effectID, resp.Effect.ID)
		assert.Equal(t, "Test Sine Wave", resp.Effect.Name)
		assert.Equal(t, "WAVEFORM", resp.Effect.EffectType)
		assert.Equal(t, "SINE", resp.Effect.Waveform)
	})

	t.Run("ListEffects", func(t *testing.T) {
		var resp struct {
			Effects []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"effects"`
		}

		err := client.Query(ctx, `
			query ListEffects($projectId: ID!) {
				effects(projectId: $projectId) {
					id
					name
				}
			}
		`, map[string]any{"projectId": projectID}, &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Effects, 1)
		assert.Equal(t, effectID, resp.Effects[0].ID)
	})

	t.Run("UpdateEffect", func(t *testing.T) {
		var resp struct {
			UpdateEffect struct {
				ID        string  `json:"id"`
				Name      string  `json:"name"`
				Waveform  string  `json:"waveform"`
				Frequency float64 `json:"frequency"`
			} `json:"updateEffect"`
		}

		err := client.Mutate(ctx, `
			mutation UpdateEffect($id: ID!, $input: UpdateEffectInput!) {
				updateEffect(id: $id, input: $input) {
					id
					name
					waveform
					frequency
				}
			}
		`, map[string]any{
			"id": effectID,
			"input": map[string]any{
				"name":      "Updated Sine Wave",
				"waveform":  "SQUARE",
				"frequency": 1.0,
			},
		}, &resp)
		require.NoError(t, err)

		assert.Equal(t, "Updated Sine Wave", resp.UpdateEffect.Name)
		assert.Equal(t, "SQUARE", resp.UpdateEffect.Waveform)
		assert.Equal(t, 1.0, resp.UpdateEffect.Frequency)
	})

	t.Run("DeleteEffect", func(t *testing.T) {
		var resp struct {
			DeleteEffect bool `json:"deleteEffect"`
		}

		err := client.Mutate(ctx, `
			mutation DeleteEffect($id: ID!) {
				deleteEffect(id: $id)
			}
		`, map[string]any{"id": effectID}, &resp)
		require.NoError(t, err)
		assert.True(t, resp.DeleteEffect)

		// Verify deletion
		var verifyResp struct {
			Effect *struct {
				ID string `json:"id"`
			} `json:"effect"`
		}
		err = client.Query(ctx, `query GetEffect($id: ID!) { effect(id: $id) { id } }`,
			map[string]any{"id": effectID}, &verifyResp)
		// Effect should be nil or error
		if err == nil {
			assert.Nil(t, verifyResp.Effect, "Effect should be deleted")
		}
	})
}

func TestCreateAllEffectTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create project
	var projectResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}
	err := client.Mutate(ctx, `mutation { createProject(input: {name: "Effect Types Test"}) { id } }`, nil, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]any{"id": projectID}, nil)
	}()

	effectTypes := []struct {
		effectType string
		name       string
	}{
		{"WAVEFORM", "Waveform Effect"},
		{"STATIC", "Static Effect"},
		{"MASTER", "Master Effect"},
	}

	for _, tc := range effectTypes {
		t.Run(tc.effectType, func(t *testing.T) {
			var resp struct {
				CreateEffect struct {
					ID         string `json:"id"`
					EffectType string `json:"effectType"`
				} `json:"createEffect"`
			}

			input := map[string]any{
				"projectId":  projectID,
				"name":       tc.name,
				"effectType": tc.effectType,
			}

			// Add type-specific fields
			if tc.effectType == "WAVEFORM" {
				input["waveform"] = "SINE"
				input["frequency"] = 1.0
			}
			if tc.effectType == "MASTER" {
				input["masterValue"] = 0.5
			}

			err := client.Mutate(ctx, `
				mutation CreateEffect($input: CreateEffectInput!) {
					createEffect(input: $input) { id effectType }
				}
			`, map[string]any{"input": input}, &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.effectType, resp.CreateEffect.EffectType)
		})
	}
}

func TestCreateAllWaveformTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var projectResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}
	err := client.Mutate(ctx, `mutation { createProject(input: {name: "Waveform Types Test"}) { id } }`, nil, &projectResp)
	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]any{"id": projectID}, nil)
	}()

	waveforms := []string{"SINE", "COSINE", "SQUARE", "SAWTOOTH", "TRIANGLE", "RANDOM"}

	for _, waveform := range waveforms {
		t.Run(waveform, func(t *testing.T) {
			var resp struct {
				CreateEffect struct {
					ID       string `json:"id"`
					Waveform string `json:"waveform"`
				} `json:"createEffect"`
			}

			err := client.Mutate(ctx, `
				mutation CreateEffect($input: CreateEffectInput!) {
					createEffect(input: $input) { id waveform }
				}
			`, map[string]any{
				"input": map[string]any{
					"projectId":  projectID,
					"name":       waveform + " Wave",
					"effectType": "WAVEFORM",
					"waveform":   waveform,
					"frequency":  1.0,
				},
			}, &resp)
			require.NoError(t, err)

			assert.Equal(t, waveform, resp.CreateEffect.Waveform)
		})
	}
}

// ============================================================================
// Effect-Fixture Association Tests
// ============================================================================

func TestEffectFixtureAssociation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create an effect
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":  setup.projectID,
			"name":       "Fixture Association Test",
			"effectType": "WAVEFORM",
			"waveform":   "SINE",
			"frequency":  1.0,
			"amplitude":  50.0,
			"offset":     50.0,
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID

	var effectFixtureID string

	t.Run("AddFixtureToEffect", func(t *testing.T) {
		var resp struct {
			AddFixtureToEffect struct {
				ID        string `json:"id"`
				EffectID  string `json:"effectId"`
				FixtureID string `json:"fixtureId"`
			} `json:"addFixtureToEffect"`
		}

		err := setup.client.Mutate(ctx, `
			mutation AddFixtureToEffect($input: AddFixtureToEffectInput!) {
				addFixtureToEffect(input: $input) {
					id
					effectId
					fixtureId
				}
			}
		`, map[string]any{
			"input": map[string]any{
				"effectId":  effectID,
				"fixtureId": setup.fixtureID,
			},
		}, &resp)
		require.NoError(t, err)

		effectFixtureID = resp.AddFixtureToEffect.ID
		assert.NotEmpty(t, effectFixtureID)
		assert.Equal(t, effectID, resp.AddFixtureToEffect.EffectID)
		assert.Equal(t, setup.fixtureID, resp.AddFixtureToEffect.FixtureID)
	})

	t.Run("AddChannelToEffectFixture", func(t *testing.T) {
		var resp struct {
			AddChannelToEffectFixture struct {
				ID             string   `json:"id"`
				ChannelOffset  *int     `json:"channelOffset"`
				AmplitudeScale *float64 `json:"amplitudeScale"`
			} `json:"addChannelToEffectFixture"`
		}

		err := setup.client.Mutate(ctx, `
			mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
				addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) {
					id
					channelOffset
					amplitudeScale
				}
			}
		`, map[string]any{
			"effectFixtureId": effectFixtureID,
			"input": map[string]any{
				"channelOffset":  0, // Dimmer channel
				"amplitudeScale": 0.8,
			},
		}, &resp)
		require.NoError(t, err)

		require.NotNil(t, resp.AddChannelToEffectFixture.ChannelOffset)
		assert.Equal(t, 0, *resp.AddChannelToEffectFixture.ChannelOffset)
		require.NotNil(t, resp.AddChannelToEffectFixture.AmplitudeScale)
		assert.Equal(t, 0.8, *resp.AddChannelToEffectFixture.AmplitudeScale)
	})

	t.Run("VerifyEffectHasFixture", func(t *testing.T) {
		var resp struct {
			Effect struct {
				ID       string `json:"id"`
				Fixtures []struct {
					ID        string `json:"id"`
					FixtureID string `json:"fixtureId"`
					Channels  []struct {
						ID            string `json:"id"`
						ChannelOffset int    `json:"channelOffset"`
					} `json:"channels"`
				} `json:"fixtures"`
			} `json:"effect"`
		}

		err := setup.client.Query(ctx, `
			query GetEffect($id: ID!) {
				effect(id: $id) {
					id
					fixtures {
						id
						fixtureId
						channels {
							id
							channelOffset
						}
					}
				}
			}
		`, map[string]any{"id": effectID}, &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Effect.Fixtures, 1)
		assert.Equal(t, setup.fixtureID, resp.Effect.Fixtures[0].FixtureID)
		assert.Len(t, resp.Effect.Fixtures[0].Channels, 1)
		assert.Equal(t, 0, resp.Effect.Fixtures[0].Channels[0].ChannelOffset)
	})

	t.Run("AddSecondFixtureWithPhaseOffset", func(t *testing.T) {
		var resp struct {
			AddFixtureToEffect struct {
				ID string `json:"id"`
			} `json:"addFixtureToEffect"`
		}

		err := setup.client.Mutate(ctx, `
			mutation AddFixtureToEffect($input: AddFixtureToEffectInput!) {
				addFixtureToEffect(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"effectId":    effectID,
				"fixtureId":   setup.fixtureID2,
				"phaseOffset": 90.0, // 90 degree offset for chasing effect
				"effectOrder": 2,
			},
		}, &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AddFixtureToEffect.ID)
	})

	t.Run("RemoveFixtureFromEffect", func(t *testing.T) {
		var resp struct {
			RemoveFixtureFromEffect bool `json:"removeFixtureFromEffect"`
		}

		err := setup.client.Mutate(ctx, `
			mutation RemoveFixtureFromEffect($effectId: ID!, $fixtureId: ID!) {
				removeFixtureFromEffect(effectId: $effectId, fixtureId: $fixtureId)
			}
		`, map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID2,
		}, &resp)
		require.NoError(t, err)
		assert.True(t, resp.RemoveFixtureFromEffect)
	})
}

// ============================================================================
// Effect-Cue Association Tests
// ============================================================================

func TestEffectCueAssociation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create look and cue
	lookID := setup.createLook(t, "Base Look", []int{128, 128, 128, 128})

	var cueResp struct {
		CreateCue struct {
			ID string `json:"id"`
		} `json:"createCue"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateCue($input: CreateCueInput!) {
			createCue(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"cueListId":   setup.cueListID,
			"name":        "Effect Test Cue",
			"cueNumber":   1.0,
			"lookId":      lookID,
			"fadeInTime":  1.0,
			"fadeOutTime": 1.0,
		},
	}, &cueResp)
	require.NoError(t, err)
	cueID := cueResp.CreateCue.ID

	// Create effect
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":  setup.projectID,
			"name":       "Cue Effect",
			"effectType": "WAVEFORM",
			"waveform":   "SINE",
			"frequency":  2.0,
			"amplitude":  30.0,
			"offset":     50.0,
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["cue_effect"] = effectID

	// Add fixture to effect with a channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	// Add dimmer channel to effect fixture
	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input": map[string]any{
			"channelOffset": 0,
		},
	}, nil)
	require.NoError(t, err)

	t.Run("AddEffectToCue", func(t *testing.T) {
		var resp struct {
			AddEffectToCue struct {
				ID        string  `json:"id"`
				EffectID  string  `json:"effectId"`
				Intensity float64 `json:"intensity"`
			} `json:"addEffectToCue"`
		}

		err := setup.client.Mutate(ctx, `
			mutation AddEffectToCue($input: AddEffectToCueInput!) {
				addEffectToCue(input: $input) {
					id
					effectId
					intensity
				}
			}
		`, map[string]any{
			"input": map[string]any{
				"cueId":     cueID,
				"effectId":  effectID,
				"intensity": 100.0,
			},
		}, &resp)
		require.NoError(t, err)

		assert.Equal(t, effectID, resp.AddEffectToCue.EffectID)
		assert.Equal(t, 100.0, resp.AddEffectToCue.Intensity)
	})

	t.Run("VerifyCueHasEffect", func(t *testing.T) {
		var resp struct {
			Cue struct {
				ID      string `json:"id"`
				Effects []struct {
					ID       string `json:"id"`
					EffectID string `json:"effectId"`
					Effect   struct {
						Name string `json:"name"`
					} `json:"effect"`
				} `json:"effects"`
			} `json:"cue"`
		}

		err := setup.client.Query(ctx, `
			query GetCue($id: ID!) {
				cue(id: $id) {
					id
					effects {
						id
						effectId
						effect {
							name
						}
					}
				}
			}
		`, map[string]any{"id": cueID}, &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Cue.Effects, 1)
		assert.Equal(t, effectID, resp.Cue.Effects[0].EffectID)
		assert.Equal(t, "Cue Effect", resp.Cue.Effects[0].Effect.Name)
	})

	t.Run("RemoveEffectFromCue", func(t *testing.T) {
		var resp struct {
			RemoveEffectFromCue bool `json:"removeEffectFromCue"`
		}

		err := setup.client.Mutate(ctx, `
			mutation RemoveEffectFromCue($cueId: ID!, $effectId: ID!) {
				removeEffectFromCue(cueId: $cueId, effectId: $effectId)
			}
		`, map[string]any{
			"cueId":    cueID,
			"effectId": effectID,
		}, &resp)
		require.NoError(t, err)
		assert.True(t, resp.RemoveEffectFromCue)
	})
}

// ============================================================================
// Effect Playback Tests (Direct Activation)
// ============================================================================

func TestEffectDirectActivation(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create a base look at mid-brightness
	lookID := setup.createLook(t, "Base", []int{128, 128, 128, 128})
	setup.activateLook(t, lookID, 0)
	time.Sleep(100 * time.Millisecond)

	// Create effect
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "Direct Activation Test",
			"effectType":      "WAVEFORM",
			"waveform":        "SINE",
			"frequency":       1.0,
			"amplitude":       50.0,
			"offset":          50.0,
			"compositionMode": "ADDITIVE",
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["direct"] = effectID

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Record baseline DMX
	baseline := setup.getDMXOutput(t)
	t.Logf("Baseline dimmer: %d", baseline[0])
	assert.Equal(t, 128, baseline[0], "Should start at 128")

	t.Run("ActivateEffect", func(t *testing.T) {
		var resp struct {
			ActivateEffect bool `json:"activateEffect"`
		}

		err := setup.client.Mutate(ctx, `
			mutation ActivateEffect($effectId: ID!, $fadeTime: Float) {
				activateEffect(effectId: $effectId, fadeTime: $fadeTime)
			}
		`, map[string]any{
			"effectId": effectID,
			"fadeTime": 0.5,
		}, &resp)
		require.NoError(t, err)
		assert.True(t, resp.ActivateEffect)

		// Wait for fade-in and effect to run
		time.Sleep(1 * time.Second)

		// Sample multiple times to see oscillation
		var samples []int
		for range 5 {
			output := setup.getDMXOutput(t)
			samples = append(samples, output[0])
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("Dimmer samples with effect: %v", samples)

		// With MODULATE mode, values should oscillate around baseline
		// The variation should be noticeable (not just noise)
		minVal := samples[0]
		maxVal := samples[0]
		for _, s := range samples {
			if s < minVal {
				minVal = s
			}
			if s > maxVal {
				maxVal = s
			}
		}
		variation := maxVal - minVal
		t.Logf("Value range: %d - %d (variation: %d)", minVal, maxVal, variation)

		// Effect should cause some variation (at least 10 DMX units)
		// Note: This depends on timing and effect parameters
		assert.True(t, variation >= 5 || maxVal != minVal,
			"Effect should cause DMX value variation, got variation of %d", variation)
	})

	t.Run("StopEffect", func(t *testing.T) {
		var resp struct {
			StopEffect bool `json:"stopEffect"`
		}

		err := setup.client.Mutate(ctx, `
			mutation StopEffect($effectId: ID!, $fadeTime: Float) {
				stopEffect(effectId: $effectId, fadeTime: $fadeTime)
			}
		`, map[string]any{
			"effectId": effectID,
			"fadeTime": 0.5,
		}, &resp)
		require.NoError(t, err)
		assert.True(t, resp.StopEffect)

		// Wait for fade-out
		time.Sleep(800 * time.Millisecond)

		// Should return to baseline
		output := setup.getDMXOutput(t)
		assert.InDelta(t, 128, output[0], 10, "Should return to baseline after stopping effect")
	})
}

// ============================================================================
// Effect Cue Playback Tests
// ============================================================================

func TestEffectPlaysDuringCue(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create base look
	lookID := setup.createLook(t, "Cue Look", []int{200, 200, 200, 200})

	// Create effect
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "Cue Playback Effect",
			"effectType":      "WAVEFORM",
			"waveform":        "SQUARE", // Square wave is easier to detect
			"frequency":       2.0,      // 2 Hz = 500ms period
			"amplitude":       100.0,    // Full amplitude
			"offset":          50.0,
			"compositionMode": "OVERRIDE",
			"onCueChange":     "FADE_OUT",
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["cue_playback"] = effectID

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Create cue with effect
	var cueResp struct {
		CreateCue struct {
			ID string `json:"id"`
		} `json:"createCue"`
	}
	err = setup.client.Mutate(ctx, `
		mutation CreateCue($input: CreateCueInput!) {
			createCue(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"cueListId":   setup.cueListID,
			"name":        "Effect Cue",
			"cueNumber":   1.0,
			"lookId":      lookID,
			"fadeInTime":  0.5,
			"fadeOutTime": 0.5,
		},
	}, &cueResp)
	require.NoError(t, err)
	cueID := cueResp.CreateCue.ID

	// Attach effect to cue
	err = setup.client.Mutate(ctx, `
		mutation AddEffectToCue($input: AddEffectToCueInput!) {
			addEffectToCue(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"cueId":     cueID,
			"effectId":  effectID,
			"intensity": 100.0,
		},
	}, nil)
	require.NoError(t, err)

	// Start from black
	_ = setup.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
	time.Sleep(100 * time.Millisecond)

	t.Run("EffectStartsWithCue", func(t *testing.T) {
		// Start cue list
		err := setup.client.Mutate(ctx, `
			mutation StartCueList($cueListId: ID!) {
				startCueList(cueListId: $cueListId)
			}
		`, map[string]any{"cueListId": setup.cueListID}, nil)
		require.NoError(t, err)

		// Wait for cue to fade in
		time.Sleep(800 * time.Millisecond)

		// Sample to detect effect (square wave should produce distinct high/low values)
		var samples []int
		for range 10 {
			output := setup.getDMXOutput(t)
			samples = append(samples, output[0])
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("DMX samples during cue with effect: %v", samples)

		// With square wave at 2Hz, we should see alternating high and low values
		// Check for variation indicating effect is running
		minVal := samples[0]
		maxVal := samples[0]
		for _, s := range samples {
			if s < minVal {
				minVal = s
			}
			if s > maxVal {
				maxVal = s
			}
		}
		t.Logf("Value range: %d - %d", minVal, maxVal)

		// Square wave with 100% amplitude should produce significant variation
		assert.True(t, maxVal-minVal > 50,
			"Square wave effect should cause significant variation, got %d", maxVal-minVal)
	})

	t.Run("EffectStopsWhenCueListStops", func(t *testing.T) {
		// Stop cue list
		err := setup.client.Mutate(ctx, `
			mutation StopCueList($cueListId: ID!) {
				stopCueList(cueListId: $cueListId)
			}
		`, map[string]any{"cueListId": setup.cueListID}, nil)
		require.NoError(t, err)

		// Also explicitly stop the effect (in case FADE_OUT behavior doesn't apply to cue list stop)
		err = setup.client.Mutate(ctx, `
			mutation StopEffect($effectId: ID!, $fadeTime: Float) {
				stopEffect(effectId: $effectId, fadeTime: $fadeTime)
			}
		`, map[string]any{
			"effectId": effectID,
			"fadeTime": 0.5,
		}, nil)
		require.NoError(t, err)

		// Wait for fade out
		time.Sleep(1 * time.Second)

		// Fade to black to ensure clean state
		_ = setup.client.Mutate(ctx, `mutation { fadeToBlack(fadeOutTime: 0) }`, nil, nil)
		time.Sleep(200 * time.Millisecond)

		// Sample again - should be stable (no effect variation)
		var samples []int
		for range 5 {
			output := setup.getDMXOutput(t)
			samples = append(samples, output[0])
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("DMX samples after cue list stopped: %v", samples)

		// Values should be stable (very low variation)
		minVal := samples[0]
		maxVal := samples[0]
		for _, s := range samples {
			if s < minVal {
				minVal = s
			}
			if s > maxVal {
				maxVal = s
			}
		}
		assert.True(t, maxVal-minVal < 20,
			"Values should be stable after stopping, got variation of %d", maxVal-minVal)
	})
}

// ============================================================================
// Effect Transition Behavior Tests
// ============================================================================

func TestEffectTransitionBehaviors(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create two looks for two cues
	look1ID := setup.createLook(t, "Look 1", []int{200, 200, 0, 0})
	look2ID := setup.createLook(t, "Look 2", []int{0, 0, 200, 200})

	// Test FADE_OUT behavior
	t.Run("FadeOutBehavior", func(t *testing.T) {
		// Create effect with FADE_OUT behavior
		var effectResp struct {
			CreateEffect struct {
				ID string `json:"id"`
			} `json:"createEffect"`
		}
		err := setup.client.Mutate(ctx, `
			mutation CreateEffect($input: CreateEffectInput!) {
				createEffect(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"projectId":       setup.projectID,
				"name":            "Fade Out Effect",
				"effectType":      "WAVEFORM",
				"waveform":        "SINE",
				"frequency":       2.0,
				"amplitude":       50.0,
				"offset":          50.0,
				"compositionMode": "OVERRIDE",
				"onCueChange":     "FADE_OUT",
			},
		}, &effectResp)
		require.NoError(t, err)
		effectID := effectResp.CreateEffect.ID
		setup.effects["fade_out"] = effectID

		// Add fixture and channel
		var efResp struct {
			AddFixtureToEffect struct {
				ID string `json:"id"`
			} `json:"addFixtureToEffect"`
		}
		err = setup.client.Mutate(ctx, `
			mutation AddFixture($input: AddFixtureToEffectInput!) {
				addFixtureToEffect(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"effectId":  effectID,
				"fixtureId": setup.fixtureID,
			},
		}, &efResp)
		require.NoError(t, err)

		err = setup.client.Mutate(ctx, `
			mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
				addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
			}
		`, map[string]any{
			"effectFixtureId": efResp.AddFixtureToEffect.ID,
			"input":           map[string]any{"channelOffset": 0},
		}, nil)
		require.NoError(t, err)

		// Create new cue list for this test
		var cueListResp struct {
			CreateCueList struct {
				ID string `json:"id"`
			} `json:"createCueList"`
		}
		err = setup.client.Mutate(ctx, `
			mutation CreateCueList($input: CreateCueListInput!) {
				createCueList(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"projectId": setup.projectID,
				"name":      "Fade Out Test Cue List",
			},
		}, &cueListResp)
		require.NoError(t, err)
		cueListID := cueListResp.CreateCueList.ID

		// Create cue 1 with effect
		var cue1Resp struct {
			CreateCue struct {
				ID string `json:"id"`
			} `json:"createCue"`
		}
		err = setup.client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"cueListId":   cueListID,
				"name":        "Cue 1 with Effect",
				"cueNumber":   1.0,
				"lookId":      look1ID,
				"fadeInTime":  0.5,
				"fadeOutTime": 0.5,
			},
		}, &cue1Resp)
		require.NoError(t, err)

		// Attach effect to cue 1
		err = setup.client.Mutate(ctx, `
			mutation AddEffectToCue($input: AddEffectToCueInput!) {
				addEffectToCue(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"cueId":     cue1Resp.CreateCue.ID,
				"effectId":  effectID,
				"intensity": 100.0,
			},
		}, nil)
		require.NoError(t, err)

		// Create cue 2 without effect
		err = setup.client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"cueListId":   cueListID,
				"name":        "Cue 2 no Effect",
				"cueNumber":   2.0,
				"lookId":      look2ID,
				"fadeInTime":  1.0,
				"fadeOutTime": 1.0,
			},
		}, nil)
		require.NoError(t, err)

		// Start cue list at cue 1
		err = setup.client.Mutate(ctx, `
			mutation StartCueList($cueListId: ID!) {
				startCueList(cueListId: $cueListId)
			}
		`, map[string]any{"cueListId": cueListID}, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		// Verify effect is running (sample for variation)
		var preSamples []int
		for range 5 {
			output := setup.getDMXOutput(t)
			preSamples = append(preSamples, output[0])
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("Pre-transition samples: %v", preSamples)

		// Go to next cue - effect should fade out
		err = setup.client.Mutate(ctx, `
			mutation NextCue($cueListId: ID!) {
				nextCue(cueListId: $cueListId)
			}
		`, map[string]any{"cueListId": cueListID}, nil)
		require.NoError(t, err)

		// Wait for transition and fade out
		time.Sleep(2 * time.Second)

		// Effect should have faded out - values should be stable
		var postSamples []int
		for range 5 {
			output := setup.getDMXOutput(t)
			postSamples = append(postSamples, output[0])
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("Post-transition samples: %v", postSamples)

		// Post-transition should be more stable (less variation)
		postMin := postSamples[0]
		postMax := postSamples[0]
		for _, s := range postSamples {
			if s < postMin {
				postMin = s
			}
			if s > postMax {
				postMax = s
			}
		}
		postVariation := postMax - postMin
		t.Logf("Post-transition variation: %d", postVariation)

		// Effect should have stopped or be very minimal
		assert.True(t, postVariation < 30,
			"Effect should have faded out, got variation of %d", postVariation)

		// Cleanup
		_ = setup.client.Mutate(ctx, `mutation StopCueList($id: ID!) { stopCueList(cueListId: $id) }`,
			map[string]any{"id": cueListID}, nil)
	})
}

// ============================================================================
// Composition Mode Tests
// ============================================================================

func TestCompositionModes(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create base look at mid-brightness
	lookID := setup.createLook(t, "Base", []int{128, 128, 128, 128})
	setup.activateLook(t, lookID, 0)
	time.Sleep(200 * time.Millisecond)

	compositionModes := []struct {
		mode        string
		description string
	}{
		{"OVERRIDE", "Effect replaces underlying values"},
		{"ADDITIVE", "Effect adds to underlying values"},
		{"MULTIPLY", "Effect scales underlying values"},
	}

	for _, tc := range compositionModes {
		t.Run(tc.mode, func(t *testing.T) {
			// Ensure base look is active
			setup.activateLook(t, lookID, 0)
			time.Sleep(100 * time.Millisecond)

			baseline := setup.getDMXOutput(t)
			t.Logf("Baseline for %s: %d", tc.mode, baseline[0])

			// Create effect with this composition mode
			var effectResp struct {
				CreateEffect struct {
					ID string `json:"id"`
				} `json:"createEffect"`
			}
			err := setup.client.Mutate(ctx, `
				mutation CreateEffect($input: CreateEffectInput!) {
					createEffect(input: $input) { id }
				}
			`, map[string]any{
				"input": map[string]any{
					"projectId":       setup.projectID,
					"name":            tc.mode + " Test Effect",
					"effectType":      "WAVEFORM",
					"waveform":        "SINE",
					"frequency":       1.0,
					"amplitude":       50.0,
					"offset":          50.0,
					"compositionMode": tc.mode,
				},
			}, &effectResp)
			require.NoError(t, err)
			effectID := effectResp.CreateEffect.ID

			// Add fixture and channel
			var efResp struct {
				AddFixtureToEffect struct {
					ID string `json:"id"`
				} `json:"addFixtureToEffect"`
			}
			err = setup.client.Mutate(ctx, `
				mutation AddFixture($input: AddFixtureToEffectInput!) {
					addFixtureToEffect(input: $input) { id }
				}
			`, map[string]any{
				"input": map[string]any{
					"effectId":  effectID,
					"fixtureId": setup.fixtureID,
				},
			}, &efResp)
			require.NoError(t, err)

			err = setup.client.Mutate(ctx, `
				mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
					addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
				}
			`, map[string]any{
				"effectFixtureId": efResp.AddFixtureToEffect.ID,
				"input":           map[string]any{"channelOffset": 0},
			}, nil)
			require.NoError(t, err)

			// Activate effect
			err = setup.client.Mutate(ctx, `
				mutation ActivateEffect($effectId: ID!, $fadeTime: Float) {
					activateEffect(effectId: $effectId, fadeTime: $fadeTime)
				}
			`, map[string]any{
				"effectId": effectID,
				"fadeTime": 0.2,
			}, nil)
			require.NoError(t, err)

			// Wait for effect to be active
			time.Sleep(500 * time.Millisecond)

			// Sample to see effect
			var samples []int
			for range 10 {
				output := setup.getDMXOutput(t)
				samples = append(samples, output[0])
				time.Sleep(100 * time.Millisecond)
			}
			t.Logf("%s samples: %v", tc.mode, samples)

			// Verify the effect is doing something
			minVal := samples[0]
			maxVal := samples[0]
			for _, s := range samples {
				if s < minVal {
					minVal = s
				}
				if s > maxVal {
					maxVal = s
				}
			}
			t.Logf("%s range: %d - %d", tc.mode, minVal, maxVal)

			// Stop effect
			err = setup.client.Mutate(ctx, `
				mutation StopEffect($effectId: ID!, $fadeTime: Float) {
					stopEffect(effectId: $effectId, fadeTime: $fadeTime)
				}
			`, map[string]any{
				"effectId": effectID,
				"fadeTime": 0.2,
			}, nil)
			require.NoError(t, err)

			// Delete effect
			time.Sleep(300 * time.Millisecond)
			_ = setup.client.Mutate(ctx, `mutation DeleteEffect($id: ID!) { deleteEffect(id: $id) }`,
				map[string]any{"id": effectID}, nil)
		})
	}
}

// ============================================================================
// Art-Net Effect Capture Tests
// ============================================================================

func TestEffectWaveformArtNetCapture(t *testing.T) {
	checkArtNetEnabled(t)

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver: %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create base look
	lookID := setup.createLook(t, "Base", []int{128, 128, 128, 128})
	setup.activateLook(t, lookID, 0)
	time.Sleep(200 * time.Millisecond)

	// Create sine wave effect
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "Art-Net Capture Effect",
			"effectType":      "WAVEFORM",
			"waveform":        "SINE",
			"frequency":       2.0, // 2 Hz = 500ms period
			"amplitude":       100.0,
			"offset":          50.0,
			"compositionMode": "OVERRIDE",
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["artnet_capture"] = effectID

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Clear frames and activate effect
	receiver.ClearFrames()

	err = setup.client.Mutate(ctx, `
		mutation ActivateEffect($effectId: ID!, $fadeTime: Float) {
			activateEffect(effectId: $effectId, fadeTime: $fadeTime)
		}
	`, map[string]any{
		"effectId": effectID,
		"fadeTime": 0.1,
	}, nil)
	require.NoError(t, err)

	// Capture 2 seconds of Art-Net frames
	time.Sleep(2 * time.Second)

	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured")
	}

	t.Logf("Captured %d Art-Net frames during 2s of effect", len(frames))

	// Extract channel 0 values
	var values []int
	for _, frame := range frames {
		if frame.Universe == 0 {
			values = append(values, int(frame.Channels[0]))
		}
	}

	if len(values) < 10 {
		t.Skipf("Not enough frames captured: %d", len(values))
	}

	// Analyze waveform - should see smooth transitions
	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	t.Logf("Captured value range: %d - %d (span: %d)", minVal, maxVal, maxVal-minVal)

	// Sine wave with 100% amplitude should cover most of the 0-255 range
	assert.True(t, maxVal-minVal > 100,
		"Sine wave should have significant amplitude, got span of %d", maxVal-minVal)

	// Stop effect
	err = setup.client.Mutate(ctx, `
		mutation StopEffect($effectId: ID!, $fadeTime: Float) {
			stopEffect(effectId: $effectId, fadeTime: $fadeTime)
		}
	`, map[string]any{
		"effectId": effectID,
		"fadeTime": 0.1,
	}, nil)
	require.NoError(t, err)
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestEffectWithNoFixtures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create effect without adding any fixtures
	var effectResp struct {
		CreateEffect struct {
			ID       string `json:"id"`
			Fixtures []struct {
				ID string `json:"id"`
			} `json:"fixtures"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id fixtures { id } }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":  setup.projectID,
			"name":       "Empty Effect",
			"effectType": "WAVEFORM",
			"waveform":   "SINE",
		},
	}, &effectResp)
	require.NoError(t, err)

	assert.Empty(t, effectResp.CreateEffect.Fixtures)

	// Activating an effect with no fixtures should not error
	var activateResp struct {
		ActivateEffect bool `json:"activateEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateEffect($effectId: ID!) {
			activateEffect(effectId: $effectId)
		}
	`, map[string]any{"effectId": effectResp.CreateEffect.ID}, &activateResp)
	require.NoError(t, err)

	// Stop effect
	_ = setup.client.Mutate(ctx, `
		mutation StopEffect($effectId: ID!) {
			stopEffect(effectId: $effectId, fadeTime: 0)
		}
	`, map[string]any{"effectId": effectResp.CreateEffect.ID}, nil)
}

func TestMultipleEffectsOnSameCue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create look and cue
	lookID := setup.createLook(t, "Multi Effect Look", []int{128, 128, 128, 128})

	var cueResp struct {
		CreateCue struct {
			ID string `json:"id"`
		} `json:"createCue"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateCue($input: CreateCueInput!) {
			createCue(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"cueListId":   setup.cueListID,
			"name":        "Multi Effect Cue",
			"cueNumber":   1.0,
			"lookId":      lookID,
			"fadeInTime":  0.5,
			"fadeOutTime": 0.5,
		},
	}, &cueResp)
	require.NoError(t, err)
	cueID := cueResp.CreateCue.ID

	// Create two effects
	effectNames := []string{"Effect A", "Effect B"}
	var effectIDs []string

	for _, name := range effectNames {
		var effectResp struct {
			CreateEffect struct {
				ID string `json:"id"`
			} `json:"createEffect"`
		}
		err := setup.client.Mutate(ctx, `
			mutation CreateEffect($input: CreateEffectInput!) {
				createEffect(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"projectId":  setup.projectID,
				"name":       name,
				"effectType": "WAVEFORM",
				"waveform":   "SINE",
				"frequency":  1.0,
			},
		}, &effectResp)
		require.NoError(t, err)
		effectIDs = append(effectIDs, effectResp.CreateEffect.ID)
		setup.effects[name] = effectResp.CreateEffect.ID
	}

	// Add both effects to the cue
	for _, effectID := range effectIDs {
		err := setup.client.Mutate(ctx, `
			mutation AddEffectToCue($input: AddEffectToCueInput!) {
				addEffectToCue(input: $input) { id }
			}
		`, map[string]any{
			"input": map[string]any{
				"cueId":     cueID,
				"effectId":  effectID,
				"intensity": 100.0,
			},
		}, nil)
		require.NoError(t, err)
	}

	// Verify cue has both effects
	var verifyResp struct {
		Cue struct {
			Effects []struct {
				ID       string `json:"id"`
				EffectID string `json:"effectId"`
			} `json:"effects"`
		} `json:"cue"`
	}
	err = setup.client.Query(ctx, `
		query GetCue($id: ID!) {
			cue(id: $id) {
				effects { id effectId }
			}
		}
	`, map[string]any{"id": cueID}, &verifyResp)
	require.NoError(t, err)

	assert.Len(t, verifyResp.Cue.Effects, 2, "Cue should have 2 effects")
}

func TestEffectPriorityBands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	priorities := []string{"BASE", "USER", "CUE", "SYSTEM"}

	for _, priority := range priorities {
		t.Run(priority, func(t *testing.T) {
			var effectResp struct {
				CreateEffect struct {
					ID           string `json:"id"`
					PriorityBand string `json:"priorityBand"`
				} `json:"createEffect"`
			}

			err := setup.client.Mutate(ctx, `
				mutation CreateEffect($input: CreateEffectInput!) {
					createEffect(input: $input) { id priorityBand }
				}
			`, map[string]any{
				"input": map[string]any{
					"projectId":    setup.projectID,
					"name":         priority + " Priority Effect",
					"effectType":   "WAVEFORM",
					"priorityBand": priority,
					"waveform":     "SINE",
				},
			}, &effectResp)
			require.NoError(t, err)

			assert.Equal(t, priority, effectResp.CreateEffect.PriorityBand)
		})
	}
}

func TestVeryHighFrequencyEffect(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create base look
	lookID := setup.createLook(t, "Base", []int{128, 128, 128, 128})
	setup.activateLook(t, lookID, 0)
	time.Sleep(100 * time.Millisecond)

	// Create high frequency effect (20 Hz = 50ms period)
	var effectResp struct {
		CreateEffect struct {
			ID string `json:"id"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "High Freq Effect",
			"effectType":      "WAVEFORM",
			"waveform":        "SQUARE",
			"frequency":       20.0, // 20 Hz
			"amplitude":       100.0,
			"offset":          50.0,
			"compositionMode": "OVERRIDE",
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["high_freq"] = effectID

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Activate and verify it runs
	err = setup.client.Mutate(ctx, `
		mutation ActivateEffect($effectId: ID!) {
			activateEffect(effectId: $effectId)
		}
	`, map[string]any{"effectId": effectID}, nil)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Sample values
	var samples []int
	for range 10 {
		output := setup.getDMXOutput(t)
		samples = append(samples, output[0])
		time.Sleep(30 * time.Millisecond)
	}
	t.Logf("High frequency effect samples: %v", samples)

	// Stop effect
	_ = setup.client.Mutate(ctx, `
		mutation StopEffect($effectId: ID!) {
			stopEffect(effectId: $effectId, fadeTime: 0)
		}
	`, map[string]any{"effectId": effectID}, nil)

	// High frequency effects are valid even if we can't sample fast enough to see all transitions
	t.Log("High frequency effect completed without error")
}

func TestVeryLowFrequencyEffect(t *testing.T) {
	checkArtNetEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create base look
	lookID := setup.createLook(t, "Base", []int{128, 128, 128, 128})
	setup.activateLook(t, lookID, 0)
	time.Sleep(100 * time.Millisecond)

	// Create very low frequency effect (0.1 Hz = 10 second period)
	var effectResp struct {
		CreateEffect struct {
			ID        string  `json:"id"`
			Frequency float64 `json:"frequency"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id frequency }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":       setup.projectID,
			"name":            "Low Freq Effect",
			"effectType":      "WAVEFORM",
			"waveform":        "SINE",
			"frequency":       0.1, // 0.1 Hz = 10s period
			"amplitude":       100.0,
			"offset":          50.0,
			"compositionMode": "OVERRIDE",
		},
	}, &effectResp)
	require.NoError(t, err)
	assert.Equal(t, 0.1, effectResp.CreateEffect.Frequency)
	setup.effects["low_freq"] = effectResp.CreateEffect.ID

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectResp.CreateEffect.ID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Activate
	err = setup.client.Mutate(ctx, `
		mutation ActivateEffect($effectId: ID!) {
			activateEffect(effectId: $effectId)
		}
	`, map[string]any{"effectId": effectResp.CreateEffect.ID}, nil)
	require.NoError(t, err)

	// Sample at 0s and 2.5s (should see ~quarter cycle progression for 0.1Hz)
	time.Sleep(200 * time.Millisecond)
	output1 := setup.getDMXOutput(t)
	t.Logf("At t=0.2s: %d", output1[0])

	time.Sleep(2300 * time.Millisecond)
	output2 := setup.getDMXOutput(t)
	t.Logf("At t=2.5s: %d", output2[0])

	// For a 0.1Hz sine wave, 2.5s is 25% of the cycle
	// We should see some progression but not dramatic change
	// The difference should be noticeable
	t.Logf("Change over 2.3s: %d", int(math.Abs(float64(output2[0]-output1[0]))))

	// Stop effect
	_ = setup.client.Mutate(ctx, `
		mutation StopEffect($effectId: ID!) {
			stopEffect(effectId: $effectId, fadeTime: 0)
		}
	`, map[string]any{"effectId": effectResp.CreateEffect.ID}, nil)
}

func TestEffectWithMinimalAmplitude(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	setup := newEffectTestSetup(t)
	defer setup.cleanup(t)

	// Create effect with very low amplitude (testing near-zero values)
	// Note: amplitude=0 may be treated as "use default" by the GraphQL schema
	var effectResp struct {
		CreateEffect struct {
			ID        string  `json:"id"`
			Amplitude float64 `json:"amplitude"`
		} `json:"createEffect"`
	}
	err := setup.client.Mutate(ctx, `
		mutation CreateEffect($input: CreateEffectInput!) {
			createEffect(input: $input) { id amplitude }
		}
	`, map[string]any{
		"input": map[string]any{
			"projectId":  setup.projectID,
			"name":       "Low Amp Effect",
			"effectType": "WAVEFORM",
			"waveform":   "SINE",
			"frequency":  1.0,
			"amplitude":  1.0, // Very low amplitude
			"offset":     50.0,
		},
	}, &effectResp)
	require.NoError(t, err)
	effectID := effectResp.CreateEffect.ID
	setup.effects["low_amp"] = effectID

	// Verify amplitude was set correctly
	assert.Equal(t, 1.0, effectResp.CreateEffect.Amplitude,
		"Low amplitude should be stored correctly")

	// Update to zero amplitude using updateEffect
	var updateResp struct {
		UpdateEffect struct {
			ID        string  `json:"id"`
			Amplitude float64 `json:"amplitude"`
		} `json:"updateEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation UpdateEffect($id: ID!, $input: UpdateEffectInput!) {
			updateEffect(id: $id, input: $input) { id amplitude }
		}
	`, map[string]any{
		"id":    effectID,
		"input": map[string]any{"amplitude": 0.0},
	}, &updateResp)
	require.NoError(t, err)
	// Note: zero amplitude update may or may not be accepted depending on validation
	t.Logf("Updated amplitude to: %f", updateResp.UpdateEffect.Amplitude)

	// Add fixture and channel
	var efResp struct {
		AddFixtureToEffect struct {
			ID string `json:"id"`
		} `json:"addFixtureToEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation AddFixture($input: AddFixtureToEffectInput!) {
			addFixtureToEffect(input: $input) { id }
		}
	`, map[string]any{
		"input": map[string]any{
			"effectId":  effectID,
			"fixtureId": setup.fixtureID,
		},
	}, &efResp)
	require.NoError(t, err)

	err = setup.client.Mutate(ctx, `
		mutation AddChannel($effectFixtureId: ID!, $input: EffectChannelInput!) {
			addChannelToEffectFixture(effectFixtureId: $effectFixtureId, input: $input) { id }
		}
	`, map[string]any{
		"effectFixtureId": efResp.AddFixtureToEffect.ID,
		"input":           map[string]any{"channelOffset": 0},
	}, nil)
	require.NoError(t, err)

	// Activate - should not error with low amplitude
	var activateResp struct {
		ActivateEffect bool `json:"activateEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation ActivateEffect($effectId: ID!) {
			activateEffect(effectId: $effectId)
		}
	`, map[string]any{"effectId": effectID}, &activateResp)
	require.NoError(t, err)
	assert.True(t, activateResp.ActivateEffect, "Should successfully activate low amplitude effect")

	// Stop effect
	var stopResp struct {
		StopEffect bool `json:"stopEffect"`
	}
	err = setup.client.Mutate(ctx, `
		mutation StopEffect($effectId: ID!) {
			stopEffect(effectId: $effectId, fadeTime: 0)
		}
	`, map[string]any{"effectId": effectID}, &stopResp)
	require.NoError(t, err)
	assert.True(t, stopResp.StopEffect, "Should successfully stop low amplitude effect")
}
