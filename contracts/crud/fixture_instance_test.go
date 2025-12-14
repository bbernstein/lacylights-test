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

// getOrCreateFixtureDefinition ensures we have a fixture definition to use.
func getOrCreateFixtureDefinition(t *testing.T, client *graphql.Client, ctx context.Context) string {
	// First try to find existing Generic Dimmer
	var listResp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
		} `json:"fixtureDefinitions"`
	}

	err := client.Query(ctx, `
		query {
			fixtureDefinitions {
				id
				manufacturer
				model
			}
		}
	`, nil, &listResp)

	require.NoError(t, err)

	// Find Generic Dimmer
	for _, def := range listResp.FixtureDefinitions {
		if def.Manufacturer == "Generic" && def.Model == "Dimmer" {
			return def.ID
		}
	}

	// If not found, create one
	var createResp struct {
		CreateFixtureDefinition struct {
			ID string `json:"id"`
		} `json:"createFixtureDefinition"`
	}

	err = client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) {
				id
			}
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
	}, &createResp)

	require.NoError(t, err)
	return createResp.CreateFixtureDefinition.ID
}

// TestFixtureInstanceCRUD tests all fixture instance CRUD operations.
func TestFixtureInstanceCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a project first
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
		"input": map[string]interface{}{"name": "Fixture Instance Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Get or create a fixture definition
	definitionID := getOrCreateFixtureDefinition(t, client, ctx)

	// CREATE
	t.Run("CreateFixtureInstance", func(t *testing.T) {
		var createResp struct {
			CreateFixtureInstance struct {
				ID           string   `json:"id"`
				Name         string   `json:"name"`
				Manufacturer string   `json:"manufacturer"`
				Model        string   `json:"model"`
				Universe     int      `json:"universe"`
				StartChannel int      `json:"startChannel"`
				ChannelCount int      `json:"channelCount"`
				Tags         []string `json:"tags"`
				Description  *string  `json:"description"`
			} `json:"createFixtureInstance"`
		}

		err := client.Mutate(ctx, `
			mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
				createFixtureInstance(input: $input) {
					id
					name
					manufacturer
					model
					universe
					startChannel
					channelCount
					tags
					description
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":    projectID,
				"definitionId": definitionID,
				"name":         "Stage Left Par 1",
				"universe":     1,
				"startChannel": 1,
				"tags":         []string{"stage-left", "par"},
				"description":  "First par on stage left",
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateFixtureInstance.ID)
		assert.Equal(t, "Stage Left Par 1", createResp.CreateFixtureInstance.Name)
		assert.Equal(t, 1, createResp.CreateFixtureInstance.Universe)
		assert.Equal(t, 1, createResp.CreateFixtureInstance.StartChannel)
		assert.Contains(t, createResp.CreateFixtureInstance.Tags, "stage-left")

		fixtureID := createResp.CreateFixtureInstance.ID

		// READ
		t.Run("ReadFixtureInstance", func(t *testing.T) {
			var readResp struct {
				FixtureInstance struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					Manufacturer string `json:"manufacturer"`
					Model        string `json:"model"`
					Universe     int    `json:"universe"`
					StartChannel int    `json:"startChannel"`
					Channels     []struct {
						ID           string `json:"id"`
						Name         string `json:"name"`
						Type         string `json:"type"`
						FadeBehavior string `json:"fadeBehavior"`
						IsDiscrete   bool   `json:"isDiscrete"`
					} `json:"channels"`
				} `json:"fixtureInstance"`
			}

			err := client.Query(ctx, `
				query GetFixtureInstance($id: ID!) {
					fixtureInstance(id: $id) {
						id
						name
						manufacturer
						model
						universe
						startChannel
						channels {
							id
							name
							type
							fadeBehavior
							isDiscrete
						}
					}
				}
			`, map[string]interface{}{"id": fixtureID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, fixtureID, readResp.FixtureInstance.ID)
			assert.Equal(t, "Stage Left Par 1", readResp.FixtureInstance.Name)
			assert.NotEmpty(t, readResp.FixtureInstance.Channels)

			// Verify FadeBehavior is inherited from definition
			for _, ch := range readResp.FixtureInstance.Channels {
				assert.Contains(t, []string{"FADE", "SNAP", "SNAP_END"}, ch.FadeBehavior,
					"Instance channel %s should have valid FadeBehavior", ch.Name)
			}
		})

		// UPDATE
		t.Run("UpdateFixtureInstance", func(t *testing.T) {
			var updateResp struct {
				UpdateFixtureInstance struct {
					ID           string   `json:"id"`
					Name         string   `json:"name"`
					Universe     int      `json:"universe"`
					StartChannel int      `json:"startChannel"`
					Tags         []string `json:"tags"`
				} `json:"updateFixtureInstance"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateFixtureInstance($id: ID!, $input: UpdateFixtureInstanceInput!) {
					updateFixtureInstance(id: $id, input: $input) {
						id
						name
						universe
						startChannel
						tags
					}
				}
			`, map[string]interface{}{
				"id": fixtureID,
				"input": map[string]interface{}{
					"name":         "Stage Left Par 1 Updated",
					"universe":     2,
					"startChannel": 50,
					"tags":         []string{"stage-left", "par", "updated"},
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Stage Left Par 1 Updated", updateResp.UpdateFixtureInstance.Name)
			assert.Equal(t, 2, updateResp.UpdateFixtureInstance.Universe)
			assert.Equal(t, 50, updateResp.UpdateFixtureInstance.StartChannel)
			assert.Contains(t, updateResp.UpdateFixtureInstance.Tags, "updated")
		})

		// LIST with pagination
		t.Run("ListFixtureInstances", func(t *testing.T) {
			var listResp struct {
				FixtureInstances struct {
					Fixtures []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"fixtures"`
					Pagination struct {
						Total      int  `json:"total"`
						Page       int  `json:"page"`
						PerPage    int  `json:"perPage"`
						HasMore    bool `json:"hasMore"`
						TotalPages int  `json:"totalPages"`
					} `json:"pagination"`
				} `json:"fixtureInstances"`
			}

			err := client.Query(ctx, `
				query ListFixtures($projectId: ID!, $page: Int, $perPage: Int) {
					fixtureInstances(projectId: $projectId, page: $page, perPage: $perPage) {
						fixtures {
							id
							name
						}
						pagination {
							total
							page
							perPage
							hasMore
							totalPages
						}
					}
				}
			`, map[string]interface{}{
				"projectId": projectID,
				"page":      1,
				"perPage":   10,
			}, &listResp)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, listResp.FixtureInstances.Pagination.Total, 1)
			found := false
			for _, f := range listResp.FixtureInstances.Fixtures {
				if f.ID == fixtureID {
					found = true
					break
				}
			}
			assert.True(t, found, "Created fixture should be in list")
		})

		// DELETE
		t.Run("DeleteFixtureInstance", func(t *testing.T) {
			var deleteResp struct {
				DeleteFixtureInstance bool `json:"deleteFixtureInstance"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteFixtureInstance($id: ID!) {
					deleteFixtureInstance(id: $id)
				}
			`, map[string]interface{}{"id": fixtureID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteFixtureInstance)

			// Verify deletion
			var verifyResp struct {
				FixtureInstance *struct {
					ID string `json:"id"`
				} `json:"fixtureInstance"`
			}

			err = client.Query(ctx, `
				query GetFixtureInstance($id: ID!) {
					fixtureInstance(id: $id) {
						id
					}
				}
			`, map[string]interface{}{"id": fixtureID}, &verifyResp)

			if err == nil {
				assert.Nil(t, verifyResp.FixtureInstance, "Deleted fixture should not be found")
			}
		})
	})
}

// TestBulkFixtureOperations tests bulk create and update operations.
func TestBulkFixtureOperations(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Bulk Fixture Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	definitionID := getOrCreateFixtureDefinition(t, client, ctx)

	// BULK CREATE
	t.Run("BulkCreateFixtures", func(t *testing.T) {
		var bulkCreateResp struct {
			BulkCreateFixtures []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"bulkCreateFixtures"`
		}

		err := client.Mutate(ctx, `
			mutation BulkCreateFixtures($input: BulkFixtureCreateInput!) {
				bulkCreateFixtures(input: $input) {
					id
					name
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"fixtures": []map[string]interface{}{
					{
						"projectId":    projectID,
						"definitionId": definitionID,
						"name":         "Bulk Fixture 1",
						"universe":     1,
						"startChannel": 1,
					},
					{
						"projectId":    projectID,
						"definitionId": definitionID,
						"name":         "Bulk Fixture 2",
						"universe":     1,
						"startChannel": 10,
					},
					{
						"projectId":    projectID,
						"definitionId": definitionID,
						"name":         "Bulk Fixture 3",
						"universe":     1,
						"startChannel": 20,
					},
				},
			},
		}, &bulkCreateResp)

		require.NoError(t, err)
		assert.Len(t, bulkCreateResp.BulkCreateFixtures, 3)

		// Store IDs for bulk update test
		fixtureIDs := make([]string, len(bulkCreateResp.BulkCreateFixtures))
		for i, f := range bulkCreateResp.BulkCreateFixtures {
			fixtureIDs[i] = f.ID
		}

		// BULK UPDATE
		t.Run("BulkUpdateFixtures", func(t *testing.T) {
			var bulkUpdateResp struct {
				BulkUpdateFixtures []struct {
					ID   string   `json:"id"`
					Name string   `json:"name"`
					Tags []string `json:"tags"`
				} `json:"bulkUpdateFixtures"`
			}

			err := client.Mutate(ctx, `
				mutation BulkUpdateFixtures($input: BulkFixtureUpdateInput!) {
					bulkUpdateFixtures(input: $input) {
						id
						name
						tags
					}
				}
			`, map[string]interface{}{
				"input": map[string]interface{}{
					"fixtures": []map[string]interface{}{
						{
							"fixtureId": fixtureIDs[0],
							"name":      "Updated Bulk Fixture 1",
							"tags":      []string{"updated"},
						},
						{
							"fixtureId": fixtureIDs[1],
							"name":      "Updated Bulk Fixture 2",
							"tags":      []string{"updated"},
						},
					},
				},
			}, &bulkUpdateResp)

			require.NoError(t, err)
			assert.Len(t, bulkUpdateResp.BulkUpdateFixtures, 2)
			for _, f := range bulkUpdateResp.BulkUpdateFixtures {
				assert.Contains(t, f.Name, "Updated")
				assert.Contains(t, f.Tags, "updated")
			}
		})
	})
}

// TestFixtureInstanceUsage tests querying fixture usage across scenes.
func TestFixtureInstanceUsage(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Fixture Usage Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	definitionID := getOrCreateFixtureDefinition(t, client, ctx)

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
			"name":         "Usage Test Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixtureInstance.ID

	// Create scene with this fixture
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
			"name":      "Usage Test Scene",
			"fixtureValues": []map[string]interface{}{
				{
					"fixtureId": fixtureID,
					"channels": []map[string]interface{}{
						{"offset": 0, "value": 255},
					},
				},
			},
		},
	}, &sceneResp)

	require.NoError(t, err)

	// Query fixture usage
	var usageResp struct {
		FixtureUsage struct {
			FixtureID   string `json:"fixtureId"`
			FixtureName string `json:"fixtureName"`
			Scenes      []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"scenes"`
		} `json:"fixtureUsage"`
	}

	err = client.Query(ctx, `
		query GetFixtureUsage($fixtureId: ID!) {
			fixtureUsage(fixtureId: $fixtureId) {
				fixtureId
				fixtureName
				scenes {
					id
					name
				}
			}
		}
	`, map[string]interface{}{"fixtureId": fixtureID}, &usageResp)

	require.NoError(t, err)
	assert.Equal(t, fixtureID, usageResp.FixtureUsage.FixtureID)
	assert.Len(t, usageResp.FixtureUsage.Scenes, 1)
	assert.Equal(t, "Usage Test Scene", usageResp.FixtureUsage.Scenes[0].Name)
}

// TestChannelMap tests the channel map query for a project.
func TestChannelMap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create project with fixtures
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
		"input": map[string]interface{}{"name": "Channel Map Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	definitionID := getOrCreateFixtureDefinition(t, client, ctx)

	// Create fixtures at different channel addresses
	for i := 0; i < 3; i++ {
		err = client.Mutate(ctx, `
			mutation CreateFixtureInstance($input: CreateFixtureInstanceInput!) {
				createFixtureInstance(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":    projectID,
				"definitionId": definitionID,
				"name":         "Channel Map Fixture " + string(rune('A'+i)),
				"universe":     1,
				"startChannel": 1 + i*10,
			},
		}, nil)
		require.NoError(t, err)
	}

	// Query channel map
	var channelMapResp struct {
		ChannelMap struct {
			ProjectID string `json:"projectId"`
			Universes []struct {
				Universe          int `json:"universe"`
				UsedChannels      int `json:"usedChannels"`
				AvailableChannels int `json:"availableChannels"`
				Fixtures          []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					StartChannel int    `json:"startChannel"`
					EndChannel   int    `json:"endChannel"`
					ChannelCount int    `json:"channelCount"`
				} `json:"fixtures"`
			} `json:"universes"`
		} `json:"channelMap"`
	}

	err = client.Query(ctx, `
		query GetChannelMap($projectId: ID!) {
			channelMap(projectId: $projectId) {
				projectId
				universes {
					universe
					usedChannels
					availableChannels
					fixtures {
						id
						name
						startChannel
						endChannel
						channelCount
					}
				}
			}
		}
	`, map[string]interface{}{"projectId": projectID}, &channelMapResp)

	require.NoError(t, err)
	assert.Equal(t, projectID, channelMapResp.ChannelMap.ProjectID)
	assert.NotEmpty(t, channelMapResp.ChannelMap.Universes)

	// Find universe 1
	var universe1 *struct {
		Universe          int `json:"universe"`
		UsedChannels      int `json:"usedChannels"`
		AvailableChannels int `json:"availableChannels"`
		Fixtures          []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			StartChannel int    `json:"startChannel"`
			EndChannel   int    `json:"endChannel"`
			ChannelCount int    `json:"channelCount"`
		} `json:"fixtures"`
	}
	for i := range channelMapResp.ChannelMap.Universes {
		if channelMapResp.ChannelMap.Universes[i].Universe == 1 {
			universe1 = &channelMapResp.ChannelMap.Universes[i]
			break
		}
	}

	require.NotNil(t, universe1, "Should have universe 1 in channel map")
	assert.Len(t, universe1.Fixtures, 3)
	assert.Greater(t, universe1.UsedChannels, 0)
}
