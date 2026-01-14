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

// TestSparseChannelsCRUD tests look CRUD operations using the sparse channel format.
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

	// CREATE - Look with sparse channel values
	t.Run("CreateLookWithSparseChannels", func(t *testing.T) {
		var createResp struct {
			CreateLook struct {
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
					LookOrder *int `json:"lookOrder"`
				} `json:"fixtureValues"`
			} `json:"createLook"`
		}

		err := client.Mutate(ctx, `
			mutation CreateLook($input: CreateLookInput!) {
				createLook(input: $input) {
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
						lookOrder
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":   projectID,
				"name":        "Sparse Channel Look",
				"description": "Look using sparse channel format - only channel 0",
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
		assert.NotEmpty(t, createResp.CreateLook.ID)
		assert.Equal(t, "Sparse Channel Look", createResp.CreateLook.Name)
		assert.NotNil(t, createResp.CreateLook.Description)
		assert.Len(t, createResp.CreateLook.FixtureValues, 2)

		// Verify sparse channels were stored correctly
		for _, fv := range createResp.CreateLook.FixtureValues {
			assert.Len(t, fv.Channels, 1, "Should only have 1 channel specified")
			assert.Equal(t, 0, fv.Channels[0].Offset)
			if fv.Fixture.ID == fixture1ID {
				assert.Equal(t, 255, fv.Channels[0].Value)
			} else {
				assert.Equal(t, 128, fv.Channels[0].Value)
			}
		}

		lookID := createResp.CreateLook.ID

		// READ - Query look returns sparse channels
		t.Run("ReadLookWithSparseChannels", func(t *testing.T) {
			var readResp struct {
				Look struct {
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
				} `json:"look"`
			}

			err := client.Query(ctx, `
				query GetLook($id: ID!) {
					look(id: $id) {
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
			`, map[string]interface{}{"id": lookID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, lookID, readResp.Look.ID)
			assert.Equal(t, "Sparse Channel Look", readResp.Look.Name)
			assert.Len(t, readResp.Look.FixtureValues, 2)

			// Verify sparse channels are returned correctly
			for _, fv := range readResp.Look.FixtureValues {
				assert.Len(t, fv.Channels, 1, "Should only return specified channels")
				assert.Equal(t, 0, fv.Channels[0].Offset)
			}
		})

		// UPDATE - Update look with sparse channels
		t.Run("UpdateLookWithSparseChannels", func(t *testing.T) {
			var updateResp struct {
				UpdateLook struct {
					ID            string  `json:"id"`
					Name          string  `json:"name"`
					Description   *string `json:"description"`
					FixtureValues []struct {
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"updateLook"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateLook($id: ID!, $input: UpdateLookInput!) {
					updateLook(id: $id, input: $input) {
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
				"id": lookID,
				"input": map[string]interface{}{
					"name":        "Updated Sparse Look",
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
			assert.Equal(t, "Updated Sparse Look", updateResp.UpdateLook.Name)

			// Verify fixture1 now has 2 channels
			fixture1Found := false
			for _, fv := range updateResp.UpdateLook.FixtureValues {
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
		t.Run("DeleteLook", func(t *testing.T) {
			var deleteResp struct {
				DeleteLook bool `json:"deleteLook"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteLook($id: ID!) {
					deleteLook(id: $id)
				}
			`, map[string]interface{}{"id": lookID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteLook)
		})
	})
}

// TestSparseChannelsAddFixtures tests adding fixtures to a look using sparse channels.
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

	// Create look with one fixture using sparse channels
	var lookResp struct {
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
			"name":      "Sparse Add Fixtures Look",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 100},
					},
				},
			},
		},
	}, &lookResp)

	require.NoError(t, err)
	lookID := lookResp.CreateLook.ID

	// ADD FIXTURES with sparse channels
	t.Run("AddFixturesToLookWithSparseChannels", func(t *testing.T) {
		var addResp struct {
			AddFixturesToLook struct {
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
			} `json:"addFixturesToLook"`
		}

		err := client.Mutate(ctx, `
			mutation AddFixtures($lookId: ID!, $fixtureValues: [FixtureValueInput!]!) {
				addFixturesToLook(lookId: $lookId, fixtureValues: $fixtureValues) {
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
			"lookId": lookID,
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
		assert.Len(t, addResp.AddFixturesToLook.FixtureValues, 3)

		// Verify sparse channels for each fixture
		fixtureChannelCounts := make(map[string]int)
		for _, fv := range addResp.AddFixturesToLook.FixtureValues {
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

	// Create look with multiple sparse channels
	var lookResp struct {
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
			"name":      "Original Sparse Look",
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
	}, &lookResp)

	require.NoError(t, err)
	lookID := lookResp.CreateLook.ID

	// Partial update: merge new fixture without replacing existing
	t.Run("MergeFixturesWithSparseChannels", func(t *testing.T) {
		var updateResp struct {
			UpdateLookPartial struct {
				FixtureValues []struct {
					Fixture struct {
						ID string `json:"id"`
					} `json:"fixture"`
					Channels []struct {
						Offset int `json:"offset"`
						Value  int `json:"value"`
					} `json:"channels"`
				} `json:"fixtureValues"`
			} `json:"updateLookPartial"`
		}

		err := client.Mutate(ctx, `
			mutation UpdateLookPartial($lookId: ID!, $fixtureValues: [FixtureValueInput!], $mergeFixtures: Boolean) {
				updateLookPartial(lookId: $lookId, fixtureValues: $fixtureValues, mergeFixtures: $mergeFixtures) {
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
			"lookId": lookID,
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
		assert.Len(t, updateResp.UpdateLookPartial.FixtureValues, 2)

		// Verify fixture1 still has its original sparse channels
		fixture1Found := false
		fixture2Found := false
		for _, fv := range updateResp.UpdateLookPartial.FixtureValues {
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

// TestSparseChannelsLookOrder tests that lookOrder is preserved with sparse channels.
func TestSparseChannelsLookOrder(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Sparse Look Order Test"},
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

	// Create look with explicit look order
	var createResp struct {
		CreateLook struct {
			ID            string `json:"id"`
			FixtureValues []struct {
				Fixture struct {
					ID string `json:"id"`
				} `json:"fixture"`
				Channels []struct {
					Offset int `json:"offset"`
					Value  int `json:"value"`
				} `json:"channels"`
				LookOrder *int `json:"lookOrder"`
			} `json:"fixtureValues"`
		} `json:"createLook"`
	}

	err = client.Mutate(ctx, `
		mutation CreateLook($input: CreateLookInput!) {
			createLook(input: $input) {
				id
				fixtureValues {
					fixture {
						id
					}
					channels {
						offset
						value
					}
					lookOrder
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"name":      "Ordered Sparse Look",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixture1ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255},
					},
					"lookOrder": 2,
				},
				{
					"fixtureId": fixture2ID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 128},
					},
					"lookOrder": 1,
				},
			},
		},
	}, &createResp)

	require.NoError(t, err)

	// Verify lookOrder is preserved with sparse channels
	for _, fv := range createResp.CreateLook.FixtureValues {
		assert.NotNil(t, fv.LookOrder, "Look order should be set")
		switch fv.Fixture.ID {
		case fixture1ID:
			assert.Equal(t, 2, *fv.LookOrder)
		case fixture2ID:
			assert.Equal(t, 1, *fv.LookOrder)
		}
	}
}
