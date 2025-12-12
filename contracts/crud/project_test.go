// Package crud provides CRUD contract tests for all LacyLights entities.
package crud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectCRUD tests all project CRUD operations.
func TestProjectCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// CREATE
	t.Run("CreateProject", func(t *testing.T) {
		var createResp struct {
			CreateProject struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description *string `json:"description"`
			} `json:"createProject"`
		}

		err := client.Mutate(ctx, `
			mutation CreateProject($input: CreateProjectInput!) {
				createProject(input: $input) {
					id
					name
					description
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"name":        "CRUD Test Project",
				"description": "Created by CRUD tests",
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateProject.ID)
		assert.Equal(t, "CRUD Test Project", createResp.CreateProject.Name)
		assert.NotNil(t, createResp.CreateProject.Description)
		assert.Equal(t, "Created by CRUD tests", *createResp.CreateProject.Description)

		projectID := createResp.CreateProject.ID

		// READ
		t.Run("ReadProject", func(t *testing.T) {
			var readResp struct {
				Project struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
				} `json:"project"`
			}

			err := client.Query(ctx, `
				query GetProject($id: ID!) {
					project(id: $id) {
						id
						name
						description
					}
				}
			`, map[string]interface{}{"id": projectID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, projectID, readResp.Project.ID)
			assert.Equal(t, "CRUD Test Project", readResp.Project.Name)
		})

		// UPDATE
		t.Run("UpdateProject", func(t *testing.T) {
			var updateResp struct {
				UpdateProject struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
				} `json:"updateProject"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateProject($id: ID!, $input: CreateProjectInput!) {
					updateProject(id: $id, input: $input) {
						id
						name
						description
					}
				}
			`, map[string]interface{}{
				"id": projectID,
				"input": map[string]interface{}{
					"name":        "Updated Project Name",
					"description": "Updated description",
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Updated Project Name", updateResp.UpdateProject.Name)
			assert.NotNil(t, updateResp.UpdateProject.Description)
			assert.Equal(t, "Updated description", *updateResp.UpdateProject.Description)
		})

		// LIST
		t.Run("ListProjects", func(t *testing.T) {
			var listResp struct {
				Projects []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"projects"`
			}

			err := client.Query(ctx, `
				query {
					projects {
						id
						name
					}
				}
			`, nil, &listResp)

			require.NoError(t, err)
			assert.NotEmpty(t, listResp.Projects)

			// Find our project in the list
			found := false
			for _, p := range listResp.Projects {
				if p.ID == projectID {
					found = true
					assert.Equal(t, "Updated Project Name", p.Name)
					break
				}
			}
			assert.True(t, found, "Created project should be in list")
		})

		// DELETE
		t.Run("DeleteProject", func(t *testing.T) {
			var deleteResp struct {
				DeleteProject bool `json:"deleteProject"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteProject($id: ID!) {
					deleteProject(id: $id)
				}
			`, map[string]interface{}{"id": projectID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteProject)

			// Verify deletion - project should not be found
			var verifyResp struct {
				Project *struct {
					ID string `json:"id"`
				} `json:"project"`
			}

			err = client.Query(ctx, `
				query GetProject($id: ID!) {
					project(id: $id) {
						id
					}
				}
			`, map[string]interface{}{"id": projectID}, &verifyResp)

			// Should either return null or error
			if err == nil {
				assert.Nil(t, verifyResp.Project, "Deleted project should not be found")
			}
		})
	})
}

// TestProjectWithRelations tests project with fixtures and scenes.
func TestProjectWithRelations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create project
	var createResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}

	err := client.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"name": "Relations Test Project"},
	}, &createResp)

	require.NoError(t, err)
	projectID := createResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create a fixture definition for testing (don't rely on built-in fixtures)
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
			"manufacturer": "Test Relations",
			"model":        fmt.Sprintf("Fixture %d", time.Now().UnixNano()),
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
			},
		},
	}, &defResp)
	require.NoError(t, err)
	definitionID := defResp.CreateFixtureDefinition.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": definitionID}, nil)
	}()

	// Create fixture instance
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
			"name":         "Test Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)

	// Create scene
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
			"projectId":     projectID,
			"name":          "Test Scene",
			"fixtureValues": []map[string]interface{}{},
		},
	}, &sceneResp)

	require.NoError(t, err)

	// Query project with relations
	var queryResp struct {
		Project struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Fixtures []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"fixtures"`
			Scenes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"scenes"`
		} `json:"project"`
	}

	err = client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
				fixtures {
					id
					name
				}
				scenes {
					id
					name
				}
			}
		}
	`, map[string]interface{}{"id": projectID}, &queryResp)

	require.NoError(t, err)
	assert.Equal(t, "Relations Test Project", queryResp.Project.Name)
	assert.Len(t, queryResp.Project.Fixtures, 1)
	assert.Equal(t, "Test Fixture", queryResp.Project.Fixtures[0].Name)
	assert.Len(t, queryResp.Project.Scenes, 1)
	assert.Equal(t, "Test Scene", queryResp.Project.Scenes[0].Name)
}
