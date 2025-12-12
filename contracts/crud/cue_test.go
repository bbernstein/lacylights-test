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

// createTestScene creates a scene for cue tests.
func createTestScene(t *testing.T, client *graphql.Client, ctx context.Context, projectID string, name string) string {
	var resp struct {
		CreateScene struct {
			ID string `json:"id"`
		} `json:"createScene"`
	}

	err := client.Mutate(ctx, `
		mutation CreateScene($input: CreateSceneInput!) {
			createScene(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":     projectID,
			"name":          name,
			"fixtureValues": []map[string]interface{}{},
		},
	}, &resp)

	require.NoError(t, err)
	return resp.CreateScene.ID
}

// TestCueListCRUD tests all cue list CRUD operations.
func TestCueListCRUD(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Cue List CRUD Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// CREATE
	t.Run("CreateCueList", func(t *testing.T) {
		var createResp struct {
			CreateCueList struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description *string `json:"description"`
				Loop        bool    `json:"loop"`
				CueCount    int     `json:"cueCount"`
			} `json:"createCueList"`
		}

		err := client.Mutate(ctx, `
			mutation CreateCueList($input: CreateCueListInput!) {
				createCueList(input: $input) {
					id
					name
					description
					loop
					cueCount
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"projectId":   projectID,
				"name":        "Act 1",
				"description": "First act cue list",
				"loop":        false,
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateCueList.ID)
		assert.Equal(t, "Act 1", createResp.CreateCueList.Name)
		assert.NotNil(t, createResp.CreateCueList.Description)
		assert.False(t, createResp.CreateCueList.Loop)
		assert.Equal(t, 0, createResp.CreateCueList.CueCount)

		cueListID := createResp.CreateCueList.ID

		// READ
		t.Run("ReadCueList", func(t *testing.T) {
			var readResp struct {
				CueList struct {
					ID            string  `json:"id"`
					Name          string  `json:"name"`
					Description   *string `json:"description"`
					Loop          bool    `json:"loop"`
					CueCount      int     `json:"cueCount"`
					TotalDuration float64 `json:"totalDuration"`
					CreatedAt     string  `json:"createdAt"`
					Cues          []struct {
						ID string `json:"id"`
					} `json:"cues"`
				} `json:"cueList"`
			}

			err := client.Query(ctx, `
				query GetCueList($id: ID!) {
					cueList(id: $id) {
						id
						name
						description
						loop
						cueCount
						totalDuration
						createdAt
						cues {
							id
						}
					}
				}
			`, map[string]interface{}{"id": cueListID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, cueListID, readResp.CueList.ID)
			assert.Equal(t, "Act 1", readResp.CueList.Name)
		})

		// UPDATE
		t.Run("UpdateCueList", func(t *testing.T) {
			var updateResp struct {
				UpdateCueList struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
					Loop        bool    `json:"loop"`
				} `json:"updateCueList"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateCueList($id: ID!, $input: CreateCueListInput!) {
					updateCueList(id: $id, input: $input) {
						id
						name
						description
						loop
					}
				}
			`, map[string]interface{}{
				"id": cueListID,
				"input": map[string]interface{}{
					"projectId":   projectID,
					"name":        "Act 1 - Updated",
					"description": "Updated description",
					"loop":        true,
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Act 1 - Updated", updateResp.UpdateCueList.Name)
			assert.True(t, updateResp.UpdateCueList.Loop)
		})

		// LIST
		t.Run("ListCueLists", func(t *testing.T) {
			var listResp struct {
				CueLists []struct {
					ID            string  `json:"id"`
					Name          string  `json:"name"`
					CueCount      int     `json:"cueCount"`
					TotalDuration float64 `json:"totalDuration"`
				} `json:"cueLists"`
			}

			err := client.Query(ctx, `
				query ListCueLists($projectId: ID!) {
					cueLists(projectId: $projectId) {
						id
						name
						cueCount
						totalDuration
					}
				}
			`, map[string]interface{}{"projectId": projectID}, &listResp)

			require.NoError(t, err)
			assert.NotEmpty(t, listResp.CueLists)
			found := false
			for _, cl := range listResp.CueLists {
				if cl.ID == cueListID {
					found = true
					break
				}
			}
			assert.True(t, found)
		})

		// DELETE
		t.Run("DeleteCueList", func(t *testing.T) {
			var deleteResp struct {
				DeleteCueList bool `json:"deleteCueList"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteCueList($id: ID!) {
					deleteCueList(id: $id)
				}
			`, map[string]interface{}{"id": cueListID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteCueList)

			// Verify deletion
			var verifyResp struct {
				CueList *struct {
					ID string `json:"id"`
				} `json:"cueList"`
			}

			err = client.Query(ctx, `
				query GetCueList($id: ID!) {
					cueList(id: $id) {
						id
					}
				}
			`, map[string]interface{}{"id": cueListID}, &verifyResp)

			if err == nil {
				assert.Nil(t, verifyResp.CueList, "Deleted cue list should not be found")
			}
		})
	})
}

// TestCueCRUD tests all cue CRUD operations.
func TestCueCRUD(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Cue CRUD Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create scene and cue list
	sceneID := createTestScene(t, client, ctx, projectID, "Cue Test Scene")

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
			"name":      "Cue Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// CREATE CUE
	t.Run("CreateCue", func(t *testing.T) {
		var createResp struct {
			CreateCue struct {
				ID          string   `json:"id"`
				Name        string   `json:"name"`
				CueNumber   float64  `json:"cueNumber"`
				FadeInTime  float64  `json:"fadeInTime"`
				FadeOutTime float64  `json:"fadeOutTime"`
				FollowTime  *float64 `json:"followTime"`
				EasingType  *string  `json:"easingType"`
				Notes       *string  `json:"notes"`
				Scene       struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"scene"`
			} `json:"createCue"`
		}

		err := client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) {
					id
					name
					cueNumber
					fadeInTime
					fadeOutTime
					followTime
					easingType
					notes
					scene {
						id
						name
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        "Opening Cue",
				"cueNumber":   1.0,
				"fadeInTime":  3.0,
				"fadeOutTime": 2.0,
				"followTime":  5.0,
				"easingType":  "LINEAR",
				"notes":       "Opening of act 1",
			},
		}, &createResp)

		require.NoError(t, err)
		assert.NotEmpty(t, createResp.CreateCue.ID)
		assert.Equal(t, "Opening Cue", createResp.CreateCue.Name)
		assert.Equal(t, 1.0, createResp.CreateCue.CueNumber)
		assert.Equal(t, 3.0, createResp.CreateCue.FadeInTime)
		assert.Equal(t, 2.0, createResp.CreateCue.FadeOutTime)
		assert.NotNil(t, createResp.CreateCue.FollowTime)
		assert.Equal(t, 5.0, *createResp.CreateCue.FollowTime)
		assert.Equal(t, sceneID, createResp.CreateCue.Scene.ID)

		cueID := createResp.CreateCue.ID

		// READ CUE
		t.Run("ReadCue", func(t *testing.T) {
			var readResp struct {
				Cue struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					CueNumber   float64 `json:"cueNumber"`
					FadeInTime  float64 `json:"fadeInTime"`
					FadeOutTime float64 `json:"fadeOutTime"`
					CueList     struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"cueList"`
				} `json:"cue"`
			}

			err := client.Query(ctx, `
				query GetCue($id: ID!) {
					cue(id: $id) {
						id
						name
						cueNumber
						fadeInTime
						fadeOutTime
						cueList {
							id
							name
						}
					}
				}
			`, map[string]interface{}{"id": cueID}, &readResp)

			require.NoError(t, err)
			assert.Equal(t, cueID, readResp.Cue.ID)
			assert.Equal(t, "Opening Cue", readResp.Cue.Name)
			assert.Equal(t, cueListID, readResp.Cue.CueList.ID)
		})

		// UPDATE CUE
		t.Run("UpdateCue", func(t *testing.T) {
			var updateResp struct {
				UpdateCue struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					FadeInTime  float64 `json:"fadeInTime"`
					FadeOutTime float64 `json:"fadeOutTime"`
				} `json:"updateCue"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateCue($id: ID!, $input: CreateCueInput!) {
					updateCue(id: $id, input: $input) {
						id
						name
						fadeInTime
						fadeOutTime
					}
				}
			`, map[string]interface{}{
				"id": cueID,
				"input": map[string]interface{}{
					"cueListId":   cueListID,
					"sceneId":     sceneID,
					"name":        "Updated Opening Cue",
					"cueNumber":   1.0,
					"fadeInTime":  5.0,
					"fadeOutTime": 3.0,
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "Updated Opening Cue", updateResp.UpdateCue.Name)
			assert.Equal(t, 5.0, updateResp.UpdateCue.FadeInTime)
		})

		// DELETE CUE
		t.Run("DeleteCue", func(t *testing.T) {
			var deleteResp struct {
				DeleteCue bool `json:"deleteCue"`
			}

			err := client.Mutate(ctx, `
				mutation DeleteCue($id: ID!) {
					deleteCue(id: $id)
				}
			`, map[string]interface{}{"id": cueID}, &deleteResp)

			require.NoError(t, err)
			assert.True(t, deleteResp.DeleteCue)

			// Verify deletion
			var verifyResp struct {
				Cue *struct {
					ID string `json:"id"`
				} `json:"cue"`
			}

			err = client.Query(ctx, `
				query GetCue($id: ID!) {
					cue(id: $id) {
						id
					}
				}
			`, map[string]interface{}{"id": cueID}, &verifyResp)

			if err == nil {
				assert.Nil(t, verifyResp.Cue, "Deleted cue should not be found")
			}
		})
	})
}

// TestCueOrdering tests reordering cues within a cue list.
func TestCueOrdering(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Cue Ordering Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create scene and cue list
	sceneID := createTestScene(t, client, ctx, projectID, "Ordering Test Scene")

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
			"name":      "Ordering Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Create multiple cues
	cueIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		var cueResp struct {
			CreateCue struct {
				ID string `json:"id"`
			} `json:"createCue"`
		}

		err := client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        "Cue " + string(rune('A'+i)),
				"cueNumber":   float64(i + 1),
				"fadeInTime":  1.0,
				"fadeOutTime": 1.0,
			},
		}, &cueResp)

		require.NoError(t, err)
		cueIDs[i] = cueResp.CreateCue.ID
	}

	// REORDER CUES
	t.Run("ReorderCues", func(t *testing.T) {
		var reorderResp struct {
			ReorderCues bool `json:"reorderCues"`
		}

		// Reverse the order using non-conflicting numbers to avoid unique constraint issues
		// during sequential updates. Use 10, 20, 30 instead of swapping 1, 2, 3
		err := client.Mutate(ctx, `
			mutation ReorderCues($cueListId: ID!, $cueOrders: [CueOrderInput!]!) {
				reorderCues(cueListId: $cueListId, cueOrders: $cueOrders)
			}
		`, map[string]interface{}{
			"cueListId": cueListID,
			"cueOrders": []map[string]interface{}{
				{"cueId": cueIDs[2], "cueNumber": 10.0}, // Cue C first
				{"cueId": cueIDs[1], "cueNumber": 20.0}, // Cue B second
				{"cueId": cueIDs[0], "cueNumber": 30.0}, // Cue A third
			},
		}, &reorderResp)

		require.NoError(t, err)
		assert.True(t, reorderResp.ReorderCues)

		// Verify new order
		var listResp struct {
			CueList struct {
				Cues []struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					CueNumber float64 `json:"cueNumber"`
				} `json:"cues"`
			} `json:"cueList"`
		}

		err = client.Query(ctx, `
			query GetCueList($id: ID!) {
				cueList(id: $id) {
					cues {
						id
						name
						cueNumber
					}
				}
			}
		`, map[string]interface{}{"id": cueListID}, &listResp)

		require.NoError(t, err)

		// Verify order by cue number
		for _, cue := range listResp.CueList.Cues {
			switch cue.ID {
			case cueIDs[2]:
				assert.Equal(t, 10.0, cue.CueNumber)
			case cueIDs[1]:
				assert.Equal(t, 20.0, cue.CueNumber)
			case cueIDs[0]:
				assert.Equal(t, 30.0, cue.CueNumber)
			}
		}
	})
}

// TestBulkCueOperations tests bulk cue updates.
func TestBulkCueOperations(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Bulk Cue Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create scene and cue list
	sceneID := createTestScene(t, client, ctx, projectID, "Bulk Cue Test Scene")

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
			"name":      "Bulk Cue Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Create multiple cues
	cueIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		var cueResp struct {
			CreateCue struct {
				ID string `json:"id"`
			} `json:"createCue"`
		}

		err := client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        "Bulk Cue " + string(rune('A'+i)),
				"cueNumber":   float64(i + 1),
				"fadeInTime":  1.0,
				"fadeOutTime": 1.0,
			},
		}, &cueResp)

		require.NoError(t, err)
		cueIDs[i] = cueResp.CreateCue.ID
	}

	// BULK UPDATE CUES
	t.Run("BulkUpdateCues", func(t *testing.T) {
		var bulkResp struct {
			BulkUpdateCues []struct {
				ID          string  `json:"id"`
				FadeInTime  float64 `json:"fadeInTime"`
				FadeOutTime float64 `json:"fadeOutTime"`
			} `json:"bulkUpdateCues"`
		}

		err := client.Mutate(ctx, `
			mutation BulkUpdateCues($input: BulkCueUpdateInput!) {
				bulkUpdateCues(input: $input) {
					id
					fadeInTime
					fadeOutTime
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueIds":      cueIDs,
				"fadeInTime":  5.0,
				"fadeOutTime": 3.0,
			},
		}, &bulkResp)

		require.NoError(t, err)
		assert.Len(t, bulkResp.BulkUpdateCues, 3)
		for _, cue := range bulkResp.BulkUpdateCues {
			assert.Equal(t, 5.0, cue.FadeInTime)
			assert.Equal(t, 3.0, cue.FadeOutTime)
		}
	})
}

// TestCueListWithSceneDetails tests fetching cue list with scene details.
func TestCueListWithSceneDetails(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Scene Details Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create fixture and scene with values
	definitionID := getOrCreateFixtureDefinition(t, client, ctx)

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
			"name":         "Scene Details Fixture",
			"universe":     1,
			"startChannel": 1,
		},
	}, &fixtureResp)

	require.NoError(t, err)
	fixtureID := fixtureResp.CreateFixtureInstance.ID

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
			"name":      "Scene Details Test Scene",
			"fixtureValues": []map[string]interface{}{
				{"fixtureId": fixtureID, "channels": []map[string]interface{}{
					{"offset": 0, "value": 200},
				}},
			},
		},
	}, &sceneResp)

	require.NoError(t, err)
	sceneID := sceneResp.CreateScene.ID

	// Create cue list with cue
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
			"name":      "Scene Details Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	err = client.Mutate(ctx, `
		mutation CreateCue($input: CreateCueInput!) {
			createCue(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"cueListId":   cueListID,
			"sceneId":     sceneID,
			"name":        "Detailed Cue",
			"cueNumber":   1.0,
			"fadeInTime":  2.0,
			"fadeOutTime": 2.0,
		},
	}, nil)

	require.NoError(t, err)

	// Query cue list with scene details
	var detailsResp struct {
		CueList struct {
			ID   string `json:"id"`
			Cues []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Scene struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					FixtureValues []struct {
						Fixture struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"fixture"`
						Channels []struct {
							Offset int `json:"offset"`
							Value  int `json:"value"`
						} `json:"channels"`
					} `json:"fixtureValues"`
				} `json:"scene"`
			} `json:"cues"`
		} `json:"cueList"`
	}

	err = client.Query(ctx, `
		query GetCueListWithDetails($id: ID!, $includeSceneDetails: Boolean) {
			cueList(id: $id, includeSceneDetails: $includeSceneDetails) {
				id
				cues {
					id
					name
					scene {
						id
						name
						fixtureValues {
							fixture {
								id
								name
							}
							channels {
								offset
								value
							}
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"id":                  cueListID,
		"includeSceneDetails": true,
	}, &detailsResp)

	require.NoError(t, err)
	assert.Len(t, detailsResp.CueList.Cues, 1)
	cue := detailsResp.CueList.Cues[0]
	assert.Equal(t, sceneID, cue.Scene.ID)
	assert.Len(t, cue.Scene.FixtureValues, 1)
	assert.Equal(t, fixtureID, cue.Scene.FixtureValues[0].Fixture.ID)
	assert.Len(t, cue.Scene.FixtureValues[0].Channels, 1)
	assert.Equal(t, 0, cue.Scene.FixtureValues[0].Channels[0].Offset)
	assert.Equal(t, 200, cue.Scene.FixtureValues[0].Channels[0].Value)
}

// TestSearchCues tests searching cues within a cue list.
func TestSearchCues(t *testing.T) {
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
		"input": map[string]interface{}{"name": "Search Cues Test Project"},
	}, &projectResp)

	require.NoError(t, err)
	projectID := projectResp.CreateProject.ID
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
			map[string]interface{}{"id": projectID}, nil)
	}()

	// Create scene and cue list
	sceneID := createTestScene(t, client, ctx, projectID, "Search Test Scene")

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
			"name":      "Search Test List",
		},
	}, &cueListResp)

	require.NoError(t, err)
	cueListID := cueListResp.CreateCueList.ID

	// Create cues with different names
	cueNames := []string{"Opening Scene", "Blackout", "Final Bow", "Scene Change"}
	for i, name := range cueNames {
		err := client.Mutate(ctx, `
			mutation CreateCue($input: CreateCueInput!) {
				createCue(input: $input) { id }
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"cueListId":   cueListID,
				"sceneId":     sceneID,
				"name":        name,
				"cueNumber":   float64(i + 1),
				"fadeInTime":  1.0,
				"fadeOutTime": 1.0,
			},
		}, nil)
		require.NoError(t, err)
	}

	// Search for cues containing "Scene"
	var searchResp struct {
		SearchCues struct {
			Cues []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"cues"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"searchCues"`
	}

	err = client.Query(ctx, `
		query SearchCues($cueListId: ID!, $query: String!) {
			searchCues(cueListId: $cueListId, query: $query) {
				cues {
					id
					name
				}
				pagination {
					total
				}
			}
		}
	`, map[string]interface{}{
		"cueListId": cueListID,
		"query":     "Scene",
	}, &searchResp)

	require.NoError(t, err)
	assert.Equal(t, 2, searchResp.SearchCues.Pagination.Total)
	for _, cue := range searchResp.SearchCues.Cues {
		assert.Contains(t, cue.Name, "Scene")
	}
}
