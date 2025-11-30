// Package integration provides migration integration tests for the LacyLights system.
package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseSchemaCompatibility verifies that Go server can read Node's SQLite database
func TestDatabaseSchemaCompatibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test database with Node schema
	dbPath := createTestDatabase(t)
	defer os.Remove(dbPath)

	// Populate database with test data
	populateTestData(t, dbPath)

	// Start Go server with this database
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Query projects from Go server
	var resp struct {
		Projects []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"projects"`
	}

	err := goClient.Query(ctx, `
		query {
			projects {
				id
				name
				description
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Projects, "Go server should read projects from Node database")

	// Verify data matches what we inserted
	found := false
	for _, p := range resp.Projects {
		if p.Name == "Migration Test Project" {
			found = true
			assert.NotNil(t, p.Description)
			if p.Description != nil {
				assert.Equal(t, "Created by Node, read by Go", *p.Description)
			}
			break
		}
	}
	assert.True(t, found, "Should find test project created in Node database")
}

// TestDatabaseTableStructure verifies all expected tables exist
func TestDatabaseTableStructure(t *testing.T) {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		t.Skip("DATABASE_PATH not set, skipping database structure test")
	}

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Expected tables in LacyLights schema
	expectedTables := []string{
		"projects",
		"fixture_definitions",
		"fixture_instances",
		"channel_definitions",
		"fixture_modes",
		"scenes",
		"scene_fixtures",
		"cue_lists",
		"cues",
		"settings",
		"preview_sessions",
		"preview_channels",
		"wifi_networks",
		"network_interfaces",
		"artnet_settings",
		"dmx_universes",
		"system_logs",
	}

	for _, tableName := range expectedTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRow(query, tableName).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "Table %s should exist", tableName)
	}
}

// TestDataPreservation verifies that data written by Node is preserved when read by Go
func TestDataPreservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Create a project with Node
	var createResp struct {
		CreateProject struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"createProject"`
	}

	testDesc := "Test for data preservation"
	err := nodeClient.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Data Preservation Test",
			"description": testDesc,
		},
	}, &createResp)

	require.NoError(t, err)
	projectID := createResp.CreateProject.ID
	t.Logf("Created project with Node: %s", projectID)

	// Read the same project with Go
	var getResp struct {
		Project struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"project"`
	}

	err = goClient.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &getResp)

	require.NoError(t, err)
	assert.Equal(t, projectID, getResp.Project.ID)
	assert.Equal(t, "Data Preservation Test", getResp.Project.Name)
	assert.NotNil(t, getResp.Project.Description)
	if getResp.Project.Description != nil {
		assert.Equal(t, testDesc, *getResp.Project.Description)
	}

	// Cleanup
	var deleteResp struct {
		DeleteProject bool `json:"deleteProject"`
	}
	err = nodeClient.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &deleteResp)
	require.NoError(t, err)
}

// TestRollbackCompatibility verifies that data written by Go can be read by Node
func TestRollbackCompatibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Create a project with Go server
	var createResp struct {
		CreateProject struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"createProject"`
	}

	testDesc := "Created by Go for rollback test"
	err := goClient.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Rollback Test Project",
			"description": testDesc,
		},
	}, &createResp)

	require.NoError(t, err)
	projectID := createResp.CreateProject.ID
	t.Logf("Created project with Go: %s", projectID)

	// Read the same project with Node (simulating rollback scenario)
	var getResp struct {
		Project struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		} `json:"project"`
	}

	err = nodeClient.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
				description
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &getResp)

	require.NoError(t, err)
	assert.Equal(t, projectID, getResp.Project.ID)
	assert.Equal(t, "Rollback Test Project", getResp.Project.Name)
	assert.NotNil(t, getResp.Project.Description)
	if getResp.Project.Description != nil {
		assert.Equal(t, testDesc, *getResp.Project.Description)
	}

	// Cleanup with Go server
	var deleteResp struct {
		DeleteProject bool `json:"deleteProject"`
	}
	err = goClient.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &deleteResp)
	require.NoError(t, err)
}

// TestComplexDataMigration tests migration of complex nested data
func TestComplexDataMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Create a project with fixtures and scenes using Node
	var projectResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}

	err := nodeClient.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Complex Migration Test",
			"description": "Testing nested data migration",
		},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID

	defer func() {
		// Cleanup
		var deleteResp struct {
			DeleteProject bool `json:"deleteProject"`
		}
		_ = nodeClient.Mutate(context.Background(), `
			mutation DeleteProject($id: ID!) {
				deleteProject(id: $id)
			}
		`, map[string]interface{}{
			"id": projectID,
		}, &deleteResp)
	}()

	// Create a fixture instance
	var fixtureResp struct {
		CreateFixtureInstance struct {
			ID string `json:"id"`
		} `json:"createFixtureInstance"`
	}

	err = nodeClient.Mutate(ctx, `
		mutation CreateFixture($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"name":         "Test PAR",
			"manufacturer": "Generic",
			"model":        "RGB PAR",
			"type":         "LED_PAR",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixtureInstance.ID

	// Create a scene
	var sceneResp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err = nodeClient.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "Test Scene",
			"description": "Scene for migration testing",
			"fixtureValues": []interface{}{
				map[string]interface{}{
					"fixtureId":     fixtureID,
					"channelValues": []int{255, 128, 64},
				},
			},
		},
	}, &sceneResp)

	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// Now read the entire structure with Go server
	var getResp struct {
		Project struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Fixtures []struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				Manufacturer  string `json:"manufacturer"`
				Model         string `json:"model"`
				Universe      int    `json:"universe"`
				StartChannel  int    `json:"startChannel"`
			} `json:"fixtures"`
			Scenes []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"scenes"`
		} `json:"project"`
	}

	err = goClient.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
				fixtures {
					id
					name
					manufacturer
					model
					universe
					startChannel
				}
				scenes {
					id
					name
					description
				}
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &getResp)

	require.NoError(t, err)
	assert.Equal(t, projectID, getResp.Project.ID)
	assert.Equal(t, "Complex Migration Test", getResp.Project.Name)

	// Verify fixture data preserved
	require.Len(t, getResp.Project.Fixtures, 1)
	fixture := getResp.Project.Fixtures[0]
	assert.Equal(t, fixtureID, fixture.ID)
	assert.Equal(t, "Test PAR", fixture.Name)
	assert.Equal(t, "Generic", fixture.Manufacturer)
	assert.Equal(t, "RGB PAR", fixture.Model)
	assert.Equal(t, 1, fixture.Universe)
	assert.Equal(t, 1, fixture.StartChannel)

	// Verify scene data preserved
	require.Len(t, getResp.Project.Scenes, 1)
	scene := getResp.Project.Scenes[0]
	assert.Equal(t, sceneID, scene.ID)
	assert.Equal(t, "Test Scene", scene.Name)
	assert.Equal(t, "Scene for migration testing", scene.Description)
}

// Helper functions

func createTestDatabase(t *testing.T) string {
	t.Helper()

	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migration.db")

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Create basic schema (simplified version of LacyLights schema)
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return dbPath
}

func populateTestData(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Insert test project
	_, err = db.Exec(`
		INSERT INTO projects (id, name, description)
		VALUES ('test-project-1', 'Migration Test Project', 'Created by Node, read by Go')
	`)
	require.NoError(t, err)
}
