// Package e2e provides end-to-end migration tests for the LacyLights system.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullMigrationWorkflow simulates a complete migration from Node to Go
func TestFullMigrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e migration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	t.Log("=== Phase 1: Create data with Node server ===")

	// Create a complete project with Node
	projectID := createTestProject(t, ctx, nodeClient)
	defer cleanupProject(t, context.Background(), nodeClient, projectID)

	// Create fixtures
	fixtureIDs := createTestFixtures(t, ctx, nodeClient, projectID)

	// Create scenes
	sceneIDs := createTestScenes(t, ctx, nodeClient, projectID, fixtureIDs)

	// Create cue list
	cueListID := createTestCueList(t, ctx, nodeClient, projectID, sceneIDs)

	t.Log("=== Phase 2: Verify Go server can read Node data ===")

	// Verify project
	verifyProject(t, ctx, goClient, projectID)

	// Verify fixtures
	verifyFixtures(t, ctx, goClient, projectID, fixtureIDs)

	// Verify scenes
	verifyScenes(t, ctx, goClient, projectID, sceneIDs)

	// Verify cue list
	verifyCueList(t, ctx, goClient, cueListID, sceneIDs)

	t.Log("=== Phase 3: Modify data with Go server ===")

	// Update project with Go
	updateProject(t, ctx, goClient, projectID)

	// Add a new fixture with Go
	newFixtureID := createFixture(t, ctx, goClient, projectID, "Go Created Fixture")
	fixtureIDs = append(fixtureIDs, newFixtureID)

	// Add a new scene with Go
	newSceneID := createScene(t, ctx, goClient, projectID, []string{newFixtureID})
	sceneIDs = append(sceneIDs, newSceneID)

	t.Log("=== Phase 4: Verify Node server can read Go modifications ===")

	// Verify updated project with Node
	verifyProjectUpdate(t, ctx, nodeClient, projectID)

	// Verify new fixture with Node
	verifyFixture(t, ctx, nodeClient, newFixtureID)

	// Verify new scene with Node
	verifyScene(t, ctx, nodeClient, newSceneID)

	t.Log("=== Migration workflow completed successfully ===")
}

// TestRollbackScenario simulates a rollback from Go to Node
func TestRollbackScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rollback test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	t.Log("=== Phase 1: Start with Go server ===")

	// Create project with Go
	projectID := createTestProject(t, ctx, goClient)
	defer cleanupProject(t, context.Background(), goClient, projectID)

	// Create fixtures with Go
	fixtureIDs := createTestFixtures(t, ctx, goClient, projectID)

	// Create scenes with Go
	sceneIDs := createTestScenes(t, ctx, goClient, projectID, fixtureIDs)

	t.Log("=== Phase 2: Rollback to Node server ===")

	// Verify Node can read all Go-created data
	verifyProject(t, ctx, nodeClient, projectID)
	verifyFixtures(t, ctx, nodeClient, projectID, fixtureIDs)
	verifyScenes(t, ctx, nodeClient, projectID, sceneIDs)

	t.Log("=== Phase 3: Continue operations with Node ===")

	// Update data with Node after rollback
	updateProject(t, ctx, nodeClient, projectID)
	newFixtureID := createFixture(t, ctx, nodeClient, projectID, "Post-Rollback Fixture")

	// Verify updates
	verifyFixture(t, ctx, nodeClient, newFixtureID)

	t.Log("=== Rollback scenario completed successfully ===")
}

// TestDataIntegrityDuringMigration verifies data integrity throughout migration
func TestDataIntegrityDuringMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data integrity test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Create baseline data with Node
	projectID := createTestProject(t, ctx, nodeClient)
	defer cleanupProject(t, context.Background(), nodeClient, projectID)

	_ = createTestFixtures(t, ctx, nodeClient, projectID)

	// Get baseline state
	baselineState := captureProjectState(t, ctx, nodeClient, projectID)

	// Verify Go sees identical state
	goState := captureProjectState(t, ctx, goClient, projectID)

	// Compare states
	assert.Equal(t, baselineState.ProjectName, goState.ProjectName,
		"Project names should match")
	assert.Equal(t, len(baselineState.Fixtures), len(goState.Fixtures),
		"Fixture counts should match")
	assert.Equal(t, len(baselineState.Scenes), len(goState.Scenes),
		"Scene counts should match")

	t.Log("Data integrity verified across Node and Go servers")
}

// TestMigrationPerformance compares performance between Node and Go
func TestMigrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
	goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))

	// Create test project
	projectID := createTestProject(t, ctx, nodeClient)
	defer cleanupProject(t, context.Background(), nodeClient, projectID)

	query := `
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
	`

	variables := map[string]interface{}{
		"id": projectID,
	}

	// Benchmark Node server
	nodeStart := time.Now()
	for i := 0; i < 10; i++ {
		var resp struct {
			Project interface{} `json:"project"`
		}
		err := nodeClient.Query(ctx, query, variables, &resp)
		require.NoError(t, err)
	}
	nodeDuration := time.Since(nodeStart)

	// Benchmark Go server
	goStart := time.Now()
	for i := 0; i < 10; i++ {
		var resp struct {
			Project interface{} `json:"project"`
		}
		err := goClient.Query(ctx, query, variables, &resp)
		require.NoError(t, err)
	}
	goDuration := time.Since(goStart)

	t.Logf("Node server: %v for 10 queries (avg: %v)", nodeDuration, nodeDuration/10)
	t.Logf("Go server: %v for 10 queries (avg: %v)", goDuration, goDuration/10)

	// Go should be at least comparable in performance
	// (Not enforcing strict performance requirements in tests)
	assert.True(t, goDuration < nodeDuration*2,
		"Go server should have reasonable performance compared to Node")
}

// Helper types

type ProjectState struct {
	ProjectName string
	Fixtures    []string
	Scenes      []string
	CueLists    []string
}

// Helper functions

func createTestProject(t *testing.T, ctx context.Context, client *graphql.Client) string {
	t.Helper()

	var resp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}

	err := client.Mutate(ctx, `
		mutation CreateProject($input: CreateProjectInput!) {
			createProject(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "E2E Migration Test Project",
			"description": "Created for end-to-end migration testing",
		},
	}, &resp)

	require.NoError(t, err)
	require.NotEmpty(t, resp.CreateProject.ID)

	t.Logf("Created project: %s", resp.CreateProject.ID)
	return resp.CreateProject.ID
}

func createTestFixtures(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) []string {
	t.Helper()

	fixtures := []struct {
		name  string
		start int
	}{
		{"Front PAR 1", 1},
		{"Front PAR 2", 4},
		{"Back PAR 1", 7},
	}

	fixtureIDs := make([]string, 0, len(fixtures))

	for _, f := range fixtures {
		id := createFixture(t, ctx, client, projectID, f.name)
		fixtureIDs = append(fixtureIDs, id)
	}

	return fixtureIDs
}

func createFixture(t *testing.T, ctx context.Context, client *graphql.Client, projectID, name string) string {
	t.Helper()

	var resp struct {
		CreateFixtureInstance struct {
			ID string `json:"id"`
		} `json:"createFixtureInstance"`
	}

	err := client.Mutate(ctx, `
		mutation CreateFixture($input: CreateFixtureInstanceInput!) {
			createFixtureInstance(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":    projectID,
			"name":         name,
			"manufacturer": "Generic",
			"model":        "RGB PAR",
			"type":         "LED_PAR",
			"universe":     1,
			"startChannel": 1,
		},
	}, &resp)

	require.NoError(t, err)
	require.NotEmpty(t, resp.CreateFixtureInstance.ID)

	t.Logf("Created fixture: %s", resp.CreateFixtureInstance.ID)
	return resp.CreateFixtureInstance.ID
}

func createTestScenes(t *testing.T, ctx context.Context, client *graphql.Client, projectID string, fixtureIDs []string) []string {
	t.Helper()

	scenes := []string{"Red Wash", "Blue Wash", "White Wash"}
	sceneIDs := make([]string, 0, len(scenes))

	for range scenes {
		id := createScene(t, ctx, client, projectID, fixtureIDs)
		sceneIDs = append(sceneIDs, id)
	}

	return sceneIDs
}

func createScene(t *testing.T, ctx context.Context, client *graphql.Client, projectID string, fixtureIDs []string) string {
	t.Helper()

	var resp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	fixtureValues := make([]interface{}, 0, len(fixtureIDs))
	for _, fid := range fixtureIDs {
		fixtureValues = append(fixtureValues, map[string]interface{}{
			"fixtureId":     fid,
			"channelValues": []int{255, 128, 64},
		})
	}

	err := client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":     projectID,
			"name":          fmt.Sprintf("Scene %d", time.Now().UnixNano()),
			"description":   "Test scene",
			"fixtureValues": fixtureValues,
		},
	}, &resp)

	require.NoError(t, err)
	require.NotEmpty(t, resp.CreateScene.ID)

	t.Logf("Created scene: %s", resp.CreateScene.ID)
	return resp.CreateScene.ID
}

func createTestCueList(t *testing.T, ctx context.Context, client *graphql.Client, projectID string, sceneIDs []string) string {
	t.Helper()

	var resp struct {
		CreateCueList struct {
			ID string `json:"id"`
		} `json:"createCueList"`
	}

	cues := make([]interface{}, 0, len(sceneIDs))
	for i, sid := range sceneIDs {
		cues = append(cues, map[string]interface{}{
			"cueNumber":  float64(i + 1),
			"sceneId":    sid,
			"name":       fmt.Sprintf("Cue %d", i+1),
			"fadeInTime": 3.0,
		})
	}

	err := client.Mutate(ctx, `
		mutation CreateCueList($input: CreateCueListInput!) {
			createCueList(input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":   projectID,
			"name":        "E2E Test Cue List",
			"description": "Test cue list",
			"cues":        cues,
		},
	}, &resp)

	require.NoError(t, err)
	require.NotEmpty(t, resp.CreateCueList.ID)

	t.Logf("Created cue list: %s", resp.CreateCueList.ID)
	return resp.CreateCueList.ID
}

func verifyProject(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) {
	t.Helper()

	var resp struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	}

	err := client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, projectID, resp.Project.ID)
	assert.NotEmpty(t, resp.Project.Name)
}

func verifyFixtures(t *testing.T, ctx context.Context, client *graphql.Client, projectID string, fixtureIDs []string) {
	t.Helper()

	var resp struct {
		Project struct {
			Fixtures []struct {
				ID string `json:"id"`
			} `json:"fixtures"`
		} `json:"project"`
	}

	err := client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				fixtures {
					id
				}
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.Project.Fixtures), len(fixtureIDs))
}

func verifyFixture(t *testing.T, ctx context.Context, client *graphql.Client, fixtureID string) {
	t.Helper()

	var resp struct {
		FixtureInstance struct {
			ID string `json:"id"`
		} `json:"fixtureInstance"`
	}

	err := client.Query(ctx, `
		query GetFixture($id: ID!) {
			fixtureInstance(id: $id) {
				id
			}
		}
	`, map[string]interface{}{
		"id": fixtureID,
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, fixtureID, resp.FixtureInstance.ID)
}

func verifyScenes(t *testing.T, ctx context.Context, client *graphql.Client, projectID string, sceneIDs []string) {
	t.Helper()

	var resp struct {
		Project struct {
			Scenes []struct {
				ID string `json:"id"`
			} `json:"scenes"`
		} `json:"project"`
	}

	err := client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				scenes {
					id
				}
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.Project.Scenes), len(sceneIDs))
}

func verifyScene(t *testing.T, ctx context.Context, client *graphql.Client, sceneID string) {
	t.Helper()

	var resp struct {
		Scene struct {
			ID string `json:"id"`
		} `json:"scene"`
	}

	err := client.Query(ctx, `
		query GetScene($id: ID!) {
			scene(id: $id) {
				id
			}
		}
	`, map[string]interface{}{
		"id": sceneID,
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, sceneID, resp.Scene.ID)
}

func verifyCueList(t *testing.T, ctx context.Context, client *graphql.Client, cueListID string, sceneIDs []string) {
	t.Helper()

	var resp struct {
		CueList struct {
			ID   string `json:"id"`
			Cues []struct {
				ID string `json:"id"`
			} `json:"cues"`
		} `json:"cueList"`
	}

	err := client.Query(ctx, `
		query GetCueList($id: ID!) {
			cueList(id: $id) {
				id
				cues {
					id
				}
			}
		}
	`, map[string]interface{}{
		"id": cueListID,
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, cueListID, resp.CueList.ID)
	assert.GreaterOrEqual(t, len(resp.CueList.Cues), len(sceneIDs))
}

func updateProject(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) {
	t.Helper()

	var resp struct {
		UpdateProject struct {
			ID string `json:"id"`
		} `json:"updateProject"`
	}

	err := client.Mutate(ctx, `
		mutation UpdateProject($id: ID!, $input: UpdateProjectInput!) {
			updateProject(id: $id, input: $input) {
				id
			}
		}
	`, map[string]interface{}{
		"id": projectID,
		"input": map[string]interface{}{
			"name": "Updated E2E Test Project",
		},
	}, &resp)

	require.NoError(t, err)
}

func verifyProjectUpdate(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) {
	t.Helper()

	var resp struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	}

	err := client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				id
				name
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, "Updated E2E Test Project", resp.Project.Name)
}

func captureProjectState(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) ProjectState {
	t.Helper()

	var resp struct {
		Project struct {
			Name     string `json:"name"`
			Fixtures []struct {
				ID string `json:"id"`
			} `json:"fixtures"`
			Scenes []struct {
				ID string `json:"id"`
			} `json:"scenes"`
			CueLists []struct {
				ID string `json:"id"`
			} `json:"cueLists"`
		} `json:"project"`
	}

	err := client.Query(ctx, `
		query GetProject($id: ID!) {
			project(id: $id) {
				name
				fixtures { id }
				scenes { id }
				cueLists { id }
			}
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	require.NoError(t, err)

	state := ProjectState{
		ProjectName: resp.Project.Name,
		Fixtures:    make([]string, len(resp.Project.Fixtures)),
		Scenes:      make([]string, len(resp.Project.Scenes)),
		CueLists:    make([]string, len(resp.Project.CueLists)),
	}

	for i, f := range resp.Project.Fixtures {
		state.Fixtures[i] = f.ID
	}
	for i, s := range resp.Project.Scenes {
		state.Scenes[i] = s.ID
	}
	for i, c := range resp.Project.CueLists {
		state.CueLists[i] = c.ID
	}

	return state
}

func cleanupProject(t *testing.T, ctx context.Context, client *graphql.Client, projectID string) {
	t.Helper()

	var resp struct {
		DeleteProject bool `json:"deleteProject"`
	}

	err := client.Mutate(ctx, `
		mutation DeleteProject($id: ID!) {
			deleteProject(id: $id)
		}
	`, map[string]interface{}{
		"id": projectID,
	}, &resp)

	if err != nil {
		t.Logf("Warning: Failed to cleanup project %s: %v", projectID, err)
	}
}
