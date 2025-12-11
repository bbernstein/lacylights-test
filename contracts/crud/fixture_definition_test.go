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

// TestFixtureDefinitionCRUD tests all fixture definition CRUD operations.
func TestFixtureDefinitionCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// CREATE
	t.Run("CreateFixtureDefinition", func(t *testing.T) {
		var createResp struct {
			CreateFixtureDefinition struct {
				ID           string `json:"id"`
				Manufacturer string `json:"manufacturer"`
				Model        string `json:"model"`
				Type         string `json:"type"`
				IsBuiltIn    bool   `json:"isBuiltIn"`
				Channels     []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					Type         string `json:"type"`
					Offset       int    `json:"offset"`
					DefaultValue int    `json:"defaultValue"`
					MinValue     int    `json:"minValue"`
					MaxValue     int    `json:"maxValue"`
					FadeBehavior string `json:"fadeBehavior"`
					IsDiscrete   bool   `json:"isDiscrete"`
				} `json:"channels"`
			} `json:"createFixtureDefinition"`
		}

		err := client.Mutate(ctx, `
			mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
				createFixtureDefinition(input: $input) {
					id
					manufacturer
					model
					type
					isBuiltIn
					channels {
						id
						name
						type
						offset
						defaultValue
						minValue
						maxValue
						fadeBehavior
						isDiscrete
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"manufacturer": "Test Manufacturer",
				"model":        "Test CRUD Model",
				"type":         "LED_PAR",
				"channels": []map[string]interface{}{
					{
						"name":         "Red",
						"type":         "RED",
						"offset":       0,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
						"fadeBehavior": "FADE",
					},
					{
						"name":         "Green",
						"type":         "GREEN",
						"offset":       1,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
						"fadeBehavior": "FADE",
					},
					{
						"name":         "Blue",
						"type":         "BLUE",
						"offset":       2,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
						"fadeBehavior": "FADE",
					},
					{
						"name":         "Dimmer",
						"type":         "INTENSITY",
						"offset":       3,
						"defaultValue": 0,
						"minValue":     0,
						"maxValue":     255,
						"fadeBehavior": "FADE",
					},
				},
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateFixtureDefinition.ID)
		assert.Equal(t, "Test Manufacturer", createResp.CreateFixtureDefinition.Manufacturer)
		assert.Equal(t, "Test CRUD Model", createResp.CreateFixtureDefinition.Model)
		assert.Equal(t, "LED_PAR", createResp.CreateFixtureDefinition.Type)
		assert.False(t, createResp.CreateFixtureDefinition.IsBuiltIn)
		assert.Len(t, createResp.CreateFixtureDefinition.Channels, 4)

		// Verify FadeBehavior is returned for channels
		for _, ch := range createResp.CreateFixtureDefinition.Channels {
			assert.Equal(t, "FADE", ch.FadeBehavior, "Channel %s should have FADE behavior", ch.Name)
		}

		definitionID := createResp.CreateFixtureDefinition.ID

		// READ
		t.Run("ReadFixtureDefinition", func(t *testing.T) {
			var readResp struct {
				FixtureDefinition struct {
					ID           string `json:"id"`
					Manufacturer string `json:"manufacturer"`
					Model        string `json:"model"`
					Type         string `json:"type"`
					Channels     []struct {
						Name         string `json:"name"`
						Type         string `json:"type"`
						FadeBehavior string `json:"fadeBehavior"`
						IsDiscrete   bool   `json:"isDiscrete"`
					} `json:"channels"`
					Modes []struct {
						ID           string `json:"id"`
						Name         string `json:"name"`
						ChannelCount int    `json:"channelCount"`
					} `json:"modes"`
				} `json:"fixtureDefinition"`
			}

			err := client.Query(ctx, `
				query GetFixtureDefinition($id: ID!) {
					fixtureDefinition(id: $id) {
						id
						manufacturer
						model
						type
						channels {
							name
							type
							fadeBehavior
							isDiscrete
						}
						modes {
							id
							name
							channelCount
						}
					}
				}
			`, map[string]interface{}{"id": definitionID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, definitionID, readResp.FixtureDefinition.ID)
			assert.Equal(t, "Test Manufacturer", readResp.FixtureDefinition.Manufacturer)
			assert.Equal(t, "Test CRUD Model", readResp.FixtureDefinition.Model)

			// Verify FadeBehavior and IsDiscrete are readable
			for _, ch := range readResp.FixtureDefinition.Channels {
				assert.Equal(t, "FADE", ch.FadeBehavior, "Channel %s should have FADE behavior", ch.Name)
				assert.False(t, ch.IsDiscrete, "Channel %s should not be discrete", ch.Name)
			}
		})

		// UPDATE
		t.Run("UpdateFixtureDefinition", func(t *testing.T) {
			var updateResp struct {
				UpdateFixtureDefinition struct {
					ID           string `json:"id"`
					Manufacturer string `json:"manufacturer"`
					Model        string `json:"model"`
					Type         string `json:"type"`
				} `json:"updateFixtureDefinition"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateFixtureDefinition($id: ID!, $input: CreateFixtureDefinitionInput!) {
					updateFixtureDefinition(id: $id, input: $input) {
						id
						manufacturer
						model
						type
					}
				}
			`, map[string]interface{}{
				"id": definitionID,
				"input": map[string]interface{}{
					"manufacturer": "Updated Manufacturer",
					"model":        "Updated Model",
					"type":         "MOVING_HEAD",
					"channels": []map[string]interface{}{
						{
							"name":         "Pan",
							"type":         "PAN",
							"offset":       0,
							"defaultValue": 128,
							"minValue":     0,
							"maxValue":     255,
						},
						{
							"name":         "Tilt",
							"type":         "TILT",
							"offset":       1,
							"defaultValue": 128,
							"minValue":     0,
							"maxValue":     255,
						},
					},
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Updated Manufacturer", updateResp.UpdateFixtureDefinition.Manufacturer)
			assert.Equal(t, "Updated Model", updateResp.UpdateFixtureDefinition.Model)
			assert.Equal(t, "MOVING_HEAD", updateResp.UpdateFixtureDefinition.Type)
		})

		// LIST with filter
		t.Run("ListFixtureDefinitions", func(t *testing.T) {
			var listResp struct {
				FixtureDefinitions []struct {
					ID           string `json:"id"`
					Manufacturer string `json:"manufacturer"`
					Model        string `json:"model"`
					Type         string `json:"type"`
					IsBuiltIn    bool   `json:"isBuiltIn"`
				} `json:"fixtureDefinitions"`
			}

			err := client.Query(ctx, `
				query ListFixtureDefinitions($filter: FixtureDefinitionFilter) {
					fixtureDefinitions(filter: $filter) {
						id
						manufacturer
						model
						type
						isBuiltIn
					}
				}
			`, map[string]interface{}{
				"filter": map[string]interface{}{
					"manufacturer": "Updated Manufacturer",
				},
			}, &listResp)

			require.NoError(t, err)
			// Find our definition
			found := false
			for _, def := range listResp.FixtureDefinitions {
				if def.ID == definitionID {
					found = true
					assert.Equal(t, "Updated Manufacturer", def.Manufacturer)
					break
				}
			}
			assert.True(t, found, "Updated definition should be in list")
		})

		// DELETE
		t.Run("DeleteFixtureDefinition", func(t *testing.T) {
			var deleteResp struct {
				DeleteFixtureDefinition bool `json:"deleteFixtureDefinition"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteFixtureDefinition($id: ID!) {
					deleteFixtureDefinition(id: $id)
				}
			`, map[string]interface{}{"id": definitionID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteFixtureDefinition)

			// Verify deletion
			var verifyResp struct {
				FixtureDefinition *struct {
					ID string `json:"id"`
				} `json:"fixtureDefinition"`
			}

			err = client.Query(ctx, `
				query GetFixtureDefinition($id: ID!) {
					fixtureDefinition(id: $id) {
						id
					}
				}
			`, map[string]interface{}{"id": definitionID}, &verifyResp)

			// Should either return null or error
			if err == nil {
				assert.Nil(t, verifyResp.FixtureDefinition, "Deleted definition should not be found")
			}
		})
	})
}

// TestFixtureDefinitionWithFilters tests querying fixture definitions with various filters.
func TestFixtureDefinitionWithFilters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Test filtering by type
	t.Run("FilterByType", func(t *testing.T) {
		var resp struct {
			FixtureDefinitions []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"fixtureDefinitions"`
		}

		err := client.Query(ctx, `
			query ListFixtureDefinitions($filter: FixtureDefinitionFilter) {
				fixtureDefinitions(filter: $filter) {
					id
					type
				}
			}
		`, map[string]interface{}{
			"filter": map[string]interface{}{
				"type": "DIMMER",
			},
		}, &resp)

		require.NoError(t, err)
		for _, def := range resp.FixtureDefinitions {
			assert.Equal(t, "DIMMER", def.Type)
		}
	})

	// Test filtering by built-in status
	t.Run("FilterByBuiltIn", func(t *testing.T) {
		var resp struct {
			FixtureDefinitions []struct {
				ID        string `json:"id"`
				IsBuiltIn bool   `json:"isBuiltIn"`
			} `json:"fixtureDefinitions"`
		}

		err := client.Query(ctx, `
			query ListFixtureDefinitions($filter: FixtureDefinitionFilter) {
				fixtureDefinitions(filter: $filter) {
					id
					isBuiltIn
				}
			}
		`, map[string]interface{}{
			"filter": map[string]interface{}{
				"isBuiltIn": true,
			},
		}, &resp)

		require.NoError(t, err)
		for _, def := range resp.FixtureDefinitions {
			assert.True(t, def.IsBuiltIn)
		}
	})
}

// TestBuiltInFixtureDefinitions tests that built-in fixtures exist and have expected properties.
func TestBuiltInFixtureDefinitions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
			Type         string `json:"type"`
			IsBuiltIn    bool   `json:"isBuiltIn"`
			Channels     []struct {
				Name         string `json:"name"`
				Type         string `json:"type"`
				FadeBehavior string `json:"fadeBehavior"`
				IsDiscrete   bool   `json:"isDiscrete"`
			} `json:"channels"`
		} `json:"fixtureDefinitions"`
	}

	err := client.Query(ctx, `
		query {
			fixtureDefinitions {
				id
				manufacturer
				model
				type
				isBuiltIn
				channels {
					name
					type
					fadeBehavior
					isDiscrete
				}
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.FixtureDefinitions, "Should have at least some fixture definitions")

	// Check that Generic Dimmer exists (typically a built-in)
	foundDimmer := false
	for _, def := range resp.FixtureDefinitions {
		if def.Manufacturer == "Generic" && def.Model == "Dimmer" {
			foundDimmer = true
			assert.Equal(t, "DIMMER", def.Type)
			assert.NotEmpty(t, def.Channels)

			// Verify channels have valid FadeBehavior
			for _, ch := range def.Channels {
				assert.Contains(t, []string{"FADE", "SNAP", "SNAP_END"}, ch.FadeBehavior,
					"Channel %s should have valid FadeBehavior", ch.Name)
			}
			break
		}
	}
	assert.True(t, foundDimmer, "Should have Generic Dimmer fixture definition")
}
