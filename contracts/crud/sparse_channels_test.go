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

// TestSparseChannelsCRUD tests scene CRUD operations using the sparse channel format.
// This test validates the new ChannelValueInput/ChannelValue format where only
// modified channels are specified instead of requiring all channels.
func TestSparseChannelsCRUD(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Sparse Channels CRUD Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixtures for testing
	fixture1ID := createTestFixture(t, client, ctx, projectID, "Sparse Test Fixture 1", 1)
	fixture2ID := createTestFixture(t, client, ctx, projectID, "Sparse Test Fixture 2", 10)

	// CREATE - Scene with sparse channel values
	t.Run("CreateSceneWithSparseChannels", func(t *testing.T) {
		var createResp struct {
			CreateScene struct {
				ID            string  `json:"id"`
				Name          string  `json:"name"`
				Description   *string `json:"description"`
				FixtureValues []struct {
					ID      string `json:"id"`
					Fixture struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"fixture"`
					Channels []struct {
						Offset int `json:"offset"`
						Value  int `json:"value"`
					} `json:"channels"`
					SceneOrder *int `json:"sceneOrder"`
				} `json:"fixtureValues"`
			} `json:"createScene"`
		}

		err := client.Mutate(ctx, `
			mutation CreateScene($input: CreateSceneInput!) {
				createScene(input: $input) {
					id
					name
					description
					fixtureValues {
						id
						fixture {
							id
							name
						}
						channels {
							offset
							value
						}
						sceneOrder
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":   projectID,
				"name":        "Sparse Channel Scene",
				"description": "Scene using sparse channel format - only channel 0",
				"fixtureValues": []map[string]interface{}{
					{
						"fixtureId": fixture1ID,
						"channels": []map[string]interface{}{
							{"offset": 0, "value": 255}, // Only set channel 0
						},
					},
					{
						"fixtureId": fixture2ID,
						"channels": []map[string]interface{}{
							{"offset": 0, "value": 128}, // Only set channel 0
						},
					},
				},
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateScene.ID)
		assert.Equal(t, "Sparse Channel Scene", createResp.CreateScene.Name)
		assert.NotNil(t, createResp.CreateScene.Description)
		assert.Len(t, createResp.CreateScene.FixtureValues, 2)

		// Verify sparse channels were stored correctly
		for _, fv := range createResp.CreateScene.FixtureValues {
			assert.Len(t, fv.Channels, 1, "Should only have 1 channel specified")
			assert.Equal(t, 0, fv.Channels[0].Offset)
			if fv.Fixture.ID == fixture1ID {
				assert.Equal(t, 255, fv.Channels[0].Value)
			} else {
				assert.Equal(t, 128, fv.Channels[0].Value)
			}
		}

		sceneID := createResp.CreateScene.ID

		// READ - Query scene returns sparse channels
		t.Run("ReadSceneWithSparseChannels", func(t *testing.T) {
			var readResp struct {
				Scene struct {
					ID            string  `json:"id"`
					Name          string  `json:"name"`
					Description   *string `json:"description"`
					FixtureValues []struct {
						ID       string `json:"id"`
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"scene"`
			}

			err := client.Query(ctx, `
				query GetScene($id: ID!) {
					scene(id: $id) {
						id
						name
						description
						fixtureValues {
							id
							channels {
								offset
								value
							}
						}
					}
				}
			`, map[string]interface{}{"id": sceneID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, sceneID, readResp.Scene.ID)
			assert.Equal(t, "Sparse Channel Scene", readResp.Scene.Name)
			assert.Len(t, readResp.Scene.FixtureValues, 2)

			// Verify sparse channels are returned correctly
			for _, fv := range readResp.Scene.FixtureValues {
				assert.Len(t, fv.Channels, 1, "Should only return specified channels")
				assert.Equal(t, 0, fv.Channels[0].Offset)
			}
		})

		// UPDATE - Update scene with sparse channels
		t.Run("UpdateSceneWithSparseChannels", func(t *testing.T) {
			var updateResp struct {
				UpdateScene struct {
					ID            string  `json:"id"`
					Name          string  `json:"name"`
					Description   *string `json:"description"`
					FixtureValues []struct {
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"updateScene"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateScene($id: ID!, $input: UpdateSceneInput!) {
					updateScene(id: $id, input: $input) {
						id
						name
						description
						fixtureValues {
							channels {
								offset
								value
							}
						}
					}
				}
			`, map[string]interface{}{
				"id": sceneID,
				"input": map[string]interface{}{
					"name":        "Updated Sparse Scene",
					"description": "Updated to use multiple sparse channels",
					"fixtureValues": []map[string]interface{}{
						{
							"fixtureId": fixture1ID,
							"channels": []map[string]interface{}{
								{"offset": 0, "value": 200},
								{"offset": 2, "value": 100}, // Add channel 2
							},
						},
						{
							"fixtureId": fixture2ID,
							"channels": []map[string]interface{}{
								{"offset": 0, "value": 64},
							},
						},
					},
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Updated Sparse Scene", updateResp.UpdateScene.Name)

			// Verify fixture1 now has 2 channels
			fixture1Found := false
			for _, fv := range updateResp.UpdateScene.FixtureValues {
				if len(fv.Channels) == 2 {
					fixture1Found = true
					assert.Equal(t, 0, fv.Channels[0].Offset)
					assert.Equal(t, 200, fv.Channels[0].Value)
					assert.Equal(t, 2, fv.Channels[1].Offset)
					assert.Equal(t, 100, fv.Channels[1].Value)
				}
			}
			assert.True(t, fixture1Found, "Should find fixture with 2 channels")
		})

		// DELETE
		t.Run("DeleteScene", func(t *testing.T) {
			var deleteResp struct {
				DeleteScene bool `json:"deleteScene"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteScene($id: ID!) {
					deleteScene(id: $id)
				}
			`, map[string]interface{}{"id": sceneID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteScene)
		})
	})
}

// TestSparseChannelsAddFixtures tests adding fixtures to a scene using sparse channels.
func TestSparseChannelsAddFixtures(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Sparse Add Fixtures Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixtures
	fixture1ID := createTestFixture(t, client, ctx, projectID, "Add Fixture 1", 1)
	fixture2ID := createTestFixture(t, client, ctx, projectID, "Add Fixture 2", 10)
	fixture3ID := createTestFixture(t, client, ctx, projectID, "Add Fixture 3", 20)

	// Create scene with one fixture using sparse channels
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
			"name":      "Sparse Add Fixtures Scene",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 100},
					},
				},
			},
		},
	}, &sceneResp)

	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// ADD FIXTURES with sparse channels
	t.Run("AddFixturesToSceneWithSparseChannels", func(t *testing.T) {
		var addResp struct {
			AddFixturesToScene struct {
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
			} `json:"addFixturesToScene"`
		}

		err := client.Mutate(ctx, `
			mutation AddFixtures($sceneId: ID!, $fixtureValues: [FixtureValueInput!]!) {
				addFixturesToScene(sceneId: $sceneId, fixtureValues: $fixtureValues) {
					id
					fixtureValues {
						fixture {
							id
						}
						channels {
							offset
							value
						}
					}
				}
			}
		`, map[string]interface{}{
			"sceneId": sceneID,
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 150},
						{"offset": 1, "value": 200},
					},
				},
				{
					"fixtureId": fixture3ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255},
					},
				},
			},
		}, &addResp)

		require.NoError(t, err)
		assert.Len(t, addResp.AddFixturesToScene.FixtureValues, 3)

		// Verify sparse channels for each fixture
		fixtureChannelCounts := make(map[string]int)
		for _, fv := range addResp.AddFixturesToScene.FixtureValues {
			fixtureChannelCounts[fv.Fixture.ID] = len(fv.Channels)
		}

		assert.Equal(t, 1, fixtureChannelCounts[fixture1ID], "Fixture 1 should have 1 channel")
		assert.Equal(t, 2, fixtureChannelCounts[fixture2ID], "Fixture 2 should have 2 channels")
		assert.Equal(t, 1, fixtureChannelCounts[fixture3ID], "Fixture 3 should have 1 channel")
	})
}

// TestSparseChannelsPartialUpdate tests partial updates using sparse channels.
func TestSparseChannelsPartialUpdate(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Sparse Partial Update Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixtures
	fixture1ID := createTestFixture(t, client, ctx, projectID, "Partial Update Fixture 1", 1)
	fixture2ID := createTestFixture(t, client, ctx, projectID, "Partial Update Fixture 2", 10)

	// Create scene with multiple sparse channels
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
			"name":      "Original Sparse Scene",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 100},
						{"offset": 1, "value": 150},
					},
				},
			},
		},
	}, &sceneResp)

	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// Partial update: merge new fixture without replacing existing
	t.Run("MergeFixturesWithSparseChannels", func(t *testing.T) {
		var updateResp struct {
			UpdateScenePartial struct {
				FixtureValues []struct {
					Fixture struct {
						ID string `json:"id"`
					} `json:"fixture"`
					Channels []struct {
						Offset int `json:"offset"`
						Value  int `json:"value"`
					} `json:"channels"`
				} `json:"fixtureValues"`
			} `json:"updateScenePartial"`
		}

		err := client.Mutate(ctx, `
			mutation UpdateScenePartial($sceneId: ID!, $fixtureValues: [FixtureValueInput!], $mergeFixtures: Boolean) {
				updateScenePartial(sceneId: $sceneId, fixtureValues: $fixtureValues, mergeFixtures: $mergeFixtures) {
					fixtureValues {
						fixture {
							id
						}
						channels {
							offset
							value
						}
					}
				}
			}
		`, map[string]interface{}{
			"sceneId": sceneID,
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 200},
					},
				},
			},
			"mergeFixtures": true,
		}, &updateResp)

		require.NoError(t, err)
		// Should now have both fixtures
		assert.Len(t, updateResp.UpdateScenePartial.FixtureValues, 2)

		// Verify fixture1 still has its original sparse channels
		fixture1Found := false
		fixture2Found := false
		for _, fv := range updateResp.UpdateScenePartial.FixtureValues {
			if fv.Fixture.ID == fixture1ID {
				fixture1Found = true
				assert.Len(t, fv.Channels, 2, "Fixture 1 should still have 2 channels")
			}
			if fv.Fixture.ID == fixture2ID {
				fixture2Found = true
				assert.Len(t, fv.Channels, 1, "Fixture 2 should have 1 channel")
			}
		}
		assert.True(t, fixture1Found, "Should find fixture 1")
		assert.True(t, fixture2Found, "Should find fixture 2")
	})
}

// TestSparseChannelsSceneOrder tests that sceneOrder is preserved with sparse channels.
func TestSparseChannelsSceneOrder(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Sparse Scene Order Test"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixtures
	fixture1ID := createTestFixture(t, client, ctx, projectID, "Order Fixture 1", 1)
	fixture2ID := createTestFixture(t, client, ctx, projectID, "Order Fixture 2", 10)

	// Create scene with explicit scene order
	var createResp struct {
		CreateScene struct {
			ID            string `json:"id"`
			FixtureValues []struct {
				Fixture struct {
					ID string `json:"id"`
				} `json:"fixture"`
				Channels []struct {
					Offset int `json:"offset"`
					Value  int `json:"value"`
				} `json:"channels"`
				SceneOrder *int `json:"sceneOrder"`
			} `json:"fixtureValues"`
		} `json:"createScene"`
	}

	err = client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) {
				id
				fixtureValues {
					fixture {
						id
					}
					channels {
						offset
						value
					}
					sceneOrder
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Ordered Sparse Scene",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255},
					},
					"sceneOrder": 2,
				},
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 128},
					},
					"sceneOrder": 1,
				},
			},
		},
	}, &createResp)

	require.NoError(t, err)

	// Verify sceneOrder is preserved with sparse channels
	for _, fv := range createResp.CreateScene.FixtureValues {
		assert.NotNil(t, fv.SceneOrder, "Scene order should be set")
		if fv.Fixture.ID == fixture1ID {
			assert.Equal(t, 2, *fv.SceneOrder)
		} else if fv.Fixture.ID == fixture2ID {
			assert.Equal(t, 1, *fv.SceneOrder)
		}
	}
}
