// Package crud provides CRUD contract tests for all LacyLights entities.
package crud

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopyFixturesToLooks tests the copyFixturesToLooks mutation.
// This mutation copies fixture channel values from a source look to multiple target looks.
func TestCopyFixturesToLooks(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Copy Fixtures Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixtures
	fixture1ID := createTestFixture(t, client, ctx, projectID, "Copy Test Fixture 1", 1)
	fixture2ID := createTestFixture(t, client, ctx, projectID, "Copy Test Fixture 2", 10)
	fixture3ID := createTestFixture(t, client, ctx, projectID, "Copy Test Fixture 3", 20)

	// Create source look with specific fixture values
	var sourceLookResp struct {
		CreateLook struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			FixtureValues []struct {
				Fixture struct {
					ID string `json:"id"`
				} `json:"fixture"`
				Channels []struct {
					Offset int `json:"offset"`
					Value  int `json:"value"`
				} `json:"channels"`
			} `json:"fixtureValues"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) {
				id
				name
				fixtureValues {
					fixture { id }
					channels { offset value }
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Source Look",
			"description": "Look with values to copy",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 200},
					},
				},
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 150},
					},
				},
				{
					"fixtureId": fixture3ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 100},
					},
				},
			},
		},
	}, &sourceLookResp)

	require.NoError(t, err)
	sourceLookID := sourceLookResp.CreateLook.ID

	// Create target looks with different initial values
	var target1Resp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Target Look 1",
			"description": "First target",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 50},
					},
				},
			},
		},
	}, &target1Resp)

	require.NoError(t, err)
	target1ID := target1Resp.CreateLook.ID

	var target2Resp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Target Look 2",
			"description": "Second target",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 75},
					},
				},
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 25},
					},
				},
			},
		},
	}, &target2Resp)

	require.NoError(t, err)
	target2ID := target2Resp.CreateLook.ID

	t.Run("BasicCopyFlow", func(t *testing.T) {
		// Copy fixture1 from source to both targets
		var copyResp struct {
			CopyFixturesToLooks struct {
				UpdatedLookCount int    `json:"updatedLookCount"`
				AffectedCueCount int    `json:"affectedCueCount"`
				OperationID      string `json:"operationId"`
				UpdatedLooks     []struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					FixtureValues []struct {
						Fixture struct {
							ID string `json:"id"`
						} `json:"fixture"`
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"updatedLooks"`
			} `json:"copyFixturesToLooks"`
		}

		err := client.Mutate(ctx, `
			mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
				copyFixturesToLooks(input: $input) {
					updatedLookCount
					affectedCueCount
					operationId
					updatedLooks {
						id
						name
						fixtureValues {
							fixture { id }
							channels { offset value }
						}
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"sourceLookId":  sourceLookID,
				"fixtureIds":    []string{fixture1ID},
				"targetLookIds": []string{target1ID, target2ID},
			},
		}, &copyResp)

		require.NoError(t, err)
		assert.Equal(t, 2, copyResp.CopyFixturesToLooks.UpdatedLookCount, "should update 2 looks")
		assert.NotEmpty(t, copyResp.CopyFixturesToLooks.OperationID, "should return operation ID for undo")

		// Verify the updated looks have correct values
		for _, look := range copyResp.CopyFixturesToLooks.UpdatedLooks {
			// Find fixture1's values in this look
			var fixture1Value *int
			for _, fv := range look.FixtureValues {
				if fv.Fixture.ID == fixture1ID {
					for _, ch := range fv.Channels {
						if ch.Offset == 0 {
							fixture1Value = &ch.Value
							break
						}
					}
				}
			}
			require.NotNil(t, fixture1Value, "fixture1 should exist in look %s", look.Name)
			assert.Equal(t, 200, *fixture1Value, "fixture1 in %s should have value 200 from source", look.Name)
		}
	})

	t.Run("CopyMultipleFixtures", func(t *testing.T) {
		// Create a fresh target look
		var freshTargetResp struct {
			CreateLook struct {
				ID string `json:"id"`
			} `json:"createLook"`
		}

		err := client.Mutate(ctx, `
			mutation CreateLook($input: CreateLookInput!) {
				createLook(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":     projectID,
				"name":          "Fresh Target",
				"fixtureValues": []map[string]interface{}{},
			},
		}, &freshTargetResp)

		require.NoError(t, err)
		freshTargetID := freshTargetResp.CreateLook.ID

		// Copy multiple fixtures
		var copyResp struct {
			CopyFixturesToLooks struct {
				UpdatedLookCount int `json:"updatedLookCount"`
				UpdatedLooks     []struct {
					ID            string `json:"id"`
					FixtureValues []struct {
						Fixture struct {
							ID string `json:"id"`
						} `json:"fixture"`
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"updatedLooks"`
			} `json:"copyFixturesToLooks"`
		}

		err = client.Mutate(ctx, `
			mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
				copyFixturesToLooks(input: $input) {
					updatedLookCount
					updatedLooks {
						id
						fixtureValues {
							fixture { id }
							channels { offset value }
						}
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"sourceLookId":  sourceLookID,
				"fixtureIds":    []string{fixture1ID, fixture2ID, fixture3ID},
				"targetLookIds": []string{freshTargetID},
			},
		}, &copyResp)

		require.NoError(t, err)
		assert.Equal(t, 1, copyResp.CopyFixturesToLooks.UpdatedLookCount)
		require.Len(t, copyResp.CopyFixturesToLooks.UpdatedLooks, 1)

		updatedLook := copyResp.CopyFixturesToLooks.UpdatedLooks[0]
		assert.Equal(t, freshTargetID, updatedLook.ID)

		// Verify all three fixtures were copied
		fixtureValues := make(map[string]int)
		for _, fv := range updatedLook.FixtureValues {
			for _, ch := range fv.Channels {
				if ch.Offset == 0 {
					fixtureValues[fv.Fixture.ID] = ch.Value
				}
			}
		}

		assert.Equal(t, 200, fixtureValues[fixture1ID], "fixture1 should have value 200")
		assert.Equal(t, 150, fixtureValues[fixture2ID], "fixture2 should have value 150")
		assert.Equal(t, 100, fixtureValues[fixture3ID], "fixture3 should have value 100")
	})

	t.Run("SourceLookInTargetsIsProcessed", func(t *testing.T) {
		// When the source look is included in targets, it is still processed (as a no-op).
		// This keeps the mutation atomic across all requested looks.
		var copyResp struct {
			CopyFixturesToLooks struct {
				UpdatedLookCount int `json:"updatedLookCount"`
				UpdatedLooks     []struct {
					ID string `json:"id"`
				} `json:"updatedLooks"`
			} `json:"copyFixturesToLooks"`
		}

		err := client.Mutate(ctx, `
			mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
				copyFixturesToLooks(input: $input) {
					updatedLookCount
					updatedLooks { id }
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"sourceLookId":  sourceLookID,
				"fixtureIds":    []string{fixture1ID},
				"targetLookIds": []string{sourceLookID, target1ID}, // source included
			},
		}, &copyResp)

		require.NoError(t, err)
		// Both looks are processed (source is included but copy to itself is effectively a no-op)
		assert.Equal(t, 2, copyResp.CopyFixturesToLooks.UpdatedLookCount)
		assert.Len(t, copyResp.CopyFixturesToLooks.UpdatedLooks, 2)
	})

	t.Run("ErrorOnInvalidSourceLook", func(t *testing.T) {
		resp, err := client.Execute(ctx, `
			mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
				copyFixturesToLooks(input: $input) {
					updatedLookCount
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"sourceLookId":  "invalid-look-id",
				"fixtureIds":    []string{fixture1ID},
				"targetLookIds": []string{target1ID},
			},
		})

		require.NoError(t, err, "HTTP request should succeed")
		require.NotEmpty(t, resp.Errors, "should have GraphQL errors for invalid source look")
		assert.Contains(t, resp.Errors[0].Message, "source look not found")
	})

	t.Run("ErrorOnMissingFixtureInSource", func(t *testing.T) {
		resp, err := client.Execute(ctx, `
			mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
				copyFixturesToLooks(input: $input) {
					updatedLookCount
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"sourceLookId":  sourceLookID,
				"fixtureIds":    []string{"non-existent-fixture-id"},
				"targetLookIds": []string{target1ID},
			},
		})

		require.NoError(t, err, "HTTP request should succeed")
		require.NotEmpty(t, resp.Errors, "should have GraphQL errors when fixture doesn't exist in source look")
		assert.Contains(t, resp.Errors[0].Message, "none of the specified fixtures exist")
	})
}

// TestCopyFixturesToLooks_UndoSupport tests that copyFixturesToLooks supports undo.
func TestCopyFixturesToLooks_UndoSupport(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Copy Fixtures Undo Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture
	fixtureID := createTestFixture(t, client, ctx, projectID, "Undo Test Fixture", 1)

	// Create source look with value 255
	var sourceLookResp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Undo Source Look",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255},
					},
				},
			},
		},
	}, &sourceLookResp)

	require.NoError(t, err)
	sourceLookID := sourceLookResp.CreateLook.ID

	// Create target look with value 50
	var targetLookResp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Undo Target Look",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 50},
					},
				},
			},
		},
	}, &targetLookResp)

	require.NoError(t, err)
	targetLookID := targetLookResp.CreateLook.ID

	// Helper to get fixture value from a look
	getFixtureValue := func(lookID, fixtureID string) int {
		var lookResp struct {
			Look struct {
				FixtureValues []struct {
					Fixture struct {
						ID string `json:"id"`
					} `json:"fixture"`
					Channels []struct {
						Offset int `json:"offset"`
						Value  int `json:"value"`
					} `json:"channels"`
				} `json:"fixtureValues"`
			} `json:"look"`
		}

		err := client.Query(ctx, `
			query GetLook($id: ID!) {
				look(id: $id) {
					fixtureValues {
						fixture { id }
						channels { offset value }
					}
				}
			}
		`, map[string]interface{}{"id": lookID}, &lookResp)

		require.NoError(t, err)

		for _, fv := range lookResp.Look.FixtureValues {
			if fv.Fixture.ID == fixtureID {
				for _, ch := range fv.Channels {
					if ch.Offset == 0 {
						return ch.Value
					}
				}
			}
		}
		t.Fatalf("fixture %s not found in look %s", fixtureID, lookID)
		return -1
	}

	// Verify initial value
	initialValue := getFixtureValue(targetLookID, fixtureID)
	assert.Equal(t, 50, initialValue, "target should start with value 50")

	// Perform the copy operation
	var copyResp struct {
		CopyFixturesToLooks struct {
			UpdatedLookCount int    `json:"updatedLookCount"`
			OperationID      string `json:"operationId"`
		} `json:"copyFixturesToLooks"`
	}

	err = client.Mutate(ctx, `
		mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
			copyFixturesToLooks(input: $input) {
				updatedLookCount
				operationId
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"sourceLookId":  sourceLookID,
			"fixtureIds":    []string{fixtureID},
			"targetLookIds": []string{targetLookID},
		},
	}, &copyResp)

	require.NoError(t, err)
	assert.NotEmpty(t, copyResp.CopyFixturesToLooks.OperationID, "should return operation ID")

	// Verify value was copied
	copiedValue := getFixtureValue(targetLookID, fixtureID)
	assert.Equal(t, 255, copiedValue, "target should now have value 255 from source")

	// Undo the operation
	var undoResp struct {
		Undo struct {
			Success           bool   `json:"success"`
			Message           string `json:"message"`
			RestoredEntityID  string `json:"restoredEntityId"`
		} `json:"undo"`
	}

	err = client.Mutate(ctx, `
		mutation Undo($projectId: ID!) {
			undo(projectId: $projectId) {
				success
				message
				restoredEntityId
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &undoResp)

	require.NoError(t, err)
	assert.True(t, undoResp.Undo.Success, "undo should succeed")

	// Verify value was restored
	restoredValue := getFixtureValue(targetLookID, fixtureID)
	assert.Equal(t, 50, restoredValue, "target should be restored to original value 50 after undo")

	// Redo the operation
	var redoResp struct {
		Redo struct {
			Success          bool   `json:"success"`
			Message          string `json:"message"`
			RestoredEntityID string `json:"restoredEntityId"`
		} `json:"redo"`
	}

	err = client.Mutate(ctx, `
		mutation Redo($projectId: ID!) {
			redo(projectId: $projectId) {
				success
				message
				restoredEntityId
			}
		}
	`, map[string]interface{}{
		"projectId": projectID,
	}, &redoResp)

	require.NoError(t, err)
	assert.True(t, redoResp.Redo.Success, "redo should succeed")

	// Verify value was re-applied
	redoneValue := getFixtureValue(targetLookID, fixtureID)
	assert.Equal(t, 255, redoneValue, "target should have value 255 again after redo")
}

// TestCopyFixturesToLooks_AffectedCueCount tests that affected cue count is reported.
func TestCopyFixturesToLooks_AffectedCueCount(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Affected Cue Count Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture
	fixtureID := createTestFixture(t, client, ctx, projectID, "Cue Count Test Fixture", 1)

	// Create source look
	var sourceLookResp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Cue Source Look",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 200},
					},
				},
			},
		},
	}, &sourceLookResp)

	require.NoError(t, err)
	sourceLookID := sourceLookResp.CreateLook.ID

	// Create target look
	var targetLookResp struct {
		CreateLook struct {
			ID string `json:"id"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":     projectID,
			"name":          "Cue Target Look",
			"fixtureValues": []map[string]interface{}{},
		},
	}, &targetLookResp)

	require.NoError(t, err)
	targetLookID := targetLookResp.CreateLook.ID

	// Create a cue list with cues using the target look
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
			"name":      "Test Cue List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Add cues that use the target look
	for i := 1; i <= 3; i++ {
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
				"name":        "Test Cue",
				"cueNumber":   float64(i),
				"lookId":      targetLookID,
				"fadeInTime":  float64(1.0),
				"fadeOutTime": float64(1.0),
			},
		}, &cueResp)

		require.NoError(t, err)
	}

	// Copy fixtures to the target look
	var copyResp struct {
		CopyFixturesToLooks struct {
			UpdatedLookCount int `json:"updatedLookCount"`
			AffectedCueCount int `json:"affectedCueCount"`
		} `json:"copyFixturesToLooks"`
	}

	err = client.Mutate(ctx, `
		mutation CopyFixturesToLooks($input: CopyFixturesToLooksInput!) {
			copyFixturesToLooks(input: $input) {
				updatedLookCount
				affectedCueCount
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"sourceLookId":  sourceLookID,
			"fixtureIds":    []string{fixtureID},
			"targetLookIds": []string{targetLookID},
		},
	}, &copyResp)

	require.NoError(t, err)
	assert.Equal(t, 1, copyResp.CopyFixturesToLooks.UpdatedLookCount)
	assert.Equal(t, 3, copyResp.CopyFixturesToLooks.AffectedCueCount, "should report 3 affected cues")
}
