// Package ofl provides contract tests for OFL (Open Fixture Library) import functionality.
package ofl

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOFLImportStatus tests querying the OFL import status.
func TestOFLImportStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		OFLImportStatus struct {
			IsImporting       bool    `json:"isImporting"`
			Phase             string  `json:"phase"`
			TotalFixtures     int     `json:"totalFixtures"`
			ImportedCount     int     `json:"importedCount"`
			FailedCount       int     `json:"failedCount"`
			SkippedCount      int     `json:"skippedCount"`
			PercentComplete   float64 `json:"percentComplete"`
			OFLVersion        *string `json:"oflVersion"`
			UsingBundledData  bool    `json:"usingBundledData"`
			CurrentManufacturer *string `json:"currentManufacturer"`
			CurrentFixture    *string `json:"currentFixture"`
			ErrorMessage      *string `json:"errorMessage"`
		} `json:"oflImportStatus"`
	}

	err := client.Query(ctx, `
		query {
			oflImportStatus {
				isImporting
				phase
				totalFixtures
				importedCount
				failedCount
				skippedCount
				percentComplete
				oflVersion
				usingBundledData
				currentManufacturer
				currentFixture
				errorMessage
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// When not importing, phase should be IDLE or COMPLETE
	if !resp.OFLImportStatus.IsImporting {
		assert.Contains(t, []string{"IDLE", "COMPLETE", "FAILED", "CANCELLED"}, resp.OFLImportStatus.Phase,
			"When not importing, phase should be IDLE, COMPLETE, FAILED, or CANCELLED")
	}

	// Percent complete should be 0-100
	assert.GreaterOrEqual(t, resp.OFLImportStatus.PercentComplete, float64(0))
	assert.LessOrEqual(t, resp.OFLImportStatus.PercentComplete, float64(100))
}

// TestCheckOFLUpdates tests checking for OFL fixture updates.
func TestCheckOFLUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		CheckOFLUpdates struct {
			HasUpdates       bool `json:"hasUpdates"`
			NewFixtureCount  int  `json:"newFixtureCount"`
			UpdatedFixtureCount int `json:"updatedFixtureCount"`
			InUseFixtureCount int `json:"inUseFixtureCount"`
			Stats struct {
				TotalInOFL    int `json:"totalInOFL"`
				TotalInDB     int `json:"totalInDB"`
				NewCount      int `json:"newCount"`
				UpdatedCount  int `json:"updatedCount"`
				UnchangedCount int `json:"unchangedCount"`
				InUseCount    int `json:"inUseCount"`
			} `json:"stats"`
			UpdatedFixtures []struct {
				Manufacturer string `json:"manufacturer"`
				Model        string `json:"model"`
				ChangeType   string `json:"changeType"`
				IsInUse      bool   `json:"isInUse"`
				InstanceCount int   `json:"instanceCount"`
			} `json:"updatedFixtures"`
		} `json:"checkOFLUpdates"`
	}

	err := client.Query(ctx, `
		query {
			checkOFLUpdates {
				hasUpdates
				newFixtureCount
				updatedFixtureCount
				inUseFixtureCount
				stats {
					totalInOFL
					totalInDB
					newCount
					updatedCount
					unchangedCount
					inUseCount
				}
				updatedFixtures {
					manufacturer
					model
					changeType
					isInUse
					instanceCount
				}
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// Stats should be non-negative
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.Stats.TotalInOFL, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.Stats.TotalInDB, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.Stats.NewCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.Stats.UpdatedCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.Stats.UnchangedCount, 0)

	// HasUpdates should match counts
	if resp.CheckOFLUpdates.NewFixtureCount > 0 || resp.CheckOFLUpdates.UpdatedFixtureCount > 0 {
		assert.True(t, resp.CheckOFLUpdates.HasUpdates, "HasUpdates should be true when there are new or updated fixtures")
	}

	// updatedFixtures changeType should be valid
	for _, fixture := range resp.CheckOFLUpdates.UpdatedFixtures {
		assert.Contains(t, []string{"NEW", "UPDATED", "UNCHANGED"}, fixture.ChangeType,
			"ChangeType should be NEW, UPDATED, or UNCHANGED")
		assert.NotEmpty(t, fixture.Manufacturer)
		assert.NotEmpty(t, fixture.Model)
	}
}

// TestTriggerOFLImport tests triggering an OFL import.
// Note: This test may take a while to complete depending on the number of fixtures.
func TestTriggerOFLImport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping OFL import test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := graphql.NewClient("")

	// First check current status
	var statusResp struct {
		OFLImportStatus struct {
			IsImporting bool   `json:"isImporting"`
			Phase       string `json:"phase"`
		} `json:"oflImportStatus"`
	}

	err := client.Query(ctx, `
		query { oflImportStatus { isImporting phase } }
	`, nil, &statusResp)
	require.NoError(t, err)

	// If already importing, skip
	if statusResp.OFLImportStatus.IsImporting {
		t.Skip("OFL import already in progress")
	}

	// Trigger import with preferBundled to use embedded data (faster)
	var triggerResp struct {
		TriggerOFLImport struct {
			Success      bool    `json:"success"`
			Message      string  `json:"message"`
			ImportID     *string `json:"importId"`
			AlreadyRunning bool  `json:"alreadyRunning"`
		} `json:"triggerOFLImport"`
	}

	err = client.Mutate(ctx, `
		mutation TriggerOFLImport($options: OFLImportOptionsInput) {
			triggerOFLImport(options: $options) {
				success
				message
				importId
				alreadyRunning
			}
		}
	`, map[string]interface{}{
		"options": map[string]interface{}{
			"preferBundled": true,
		},
	}, &triggerResp)

	require.NoError(t, err)

	if triggerResp.TriggerOFLImport.AlreadyRunning {
		t.Log("Import was already running")
		return
	}

	assert.True(t, triggerResp.TriggerOFLImport.Success, "Import should start successfully")
	assert.NotEmpty(t, triggerResp.TriggerOFLImport.Message)

	// Poll for completion (max 3 minutes)
	pollCtx, pollCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer pollCancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			t.Fatal("Timed out waiting for OFL import to complete")
		case <-ticker.C:
			var pollResp struct {
				OFLImportStatus struct {
					IsImporting     bool    `json:"isImporting"`
					Phase           string  `json:"phase"`
					PercentComplete float64 `json:"percentComplete"`
					ImportedCount   int     `json:"importedCount"`
					FailedCount     int     `json:"failedCount"`
					ErrorMessage    *string `json:"errorMessage"`
				} `json:"oflImportStatus"`
			}

			err := client.Query(pollCtx, `
				query {
					oflImportStatus {
						isImporting
						phase
						percentComplete
						importedCount
						failedCount
						errorMessage
					}
				}
			`, nil, &pollResp)

			if err != nil {
				t.Logf("Poll error: %v", err)
				continue
			}

			t.Logf("Import status: phase=%s, progress=%.1f%%, imported=%d, failed=%d",
				pollResp.OFLImportStatus.Phase,
				pollResp.OFLImportStatus.PercentComplete,
				pollResp.OFLImportStatus.ImportedCount,
				pollResp.OFLImportStatus.FailedCount)

			if !pollResp.OFLImportStatus.IsImporting {
				// Import completed - check final phase
				switch pollResp.OFLImportStatus.Phase {
				case "FAILED":
					errMsg := "unknown error"
					if pollResp.OFLImportStatus.ErrorMessage != nil {
						errMsg = *pollResp.OFLImportStatus.ErrorMessage
					}
					t.Fatalf("OFL import failed: %s", errMsg)
				case "CANCELLED":
					t.Fatal("OFL import was cancelled unexpectedly")
				case "COMPLETE":
					assert.Greater(t, pollResp.OFLImportStatus.ImportedCount, 0, "Should have imported some fixtures")
					t.Logf("Import completed: %d fixtures imported, %d failed",
						pollResp.OFLImportStatus.ImportedCount,
						pollResp.OFLImportStatus.FailedCount)
					return
				default:
					t.Fatalf("OFL import ended in unexpected phase: %s", pollResp.OFLImportStatus.Phase)
				}
			}
		}
	}
}

// TestCancelOFLImport tests cancelling an OFL import.
func TestCancelOFLImport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Cancel should work whether or not an import is running
	var resp struct {
		CancelOFLImport struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		} `json:"cancelOFLImport"`
	}

	err := client.Mutate(ctx, `
		mutation {
			cancelOFLImport {
				success
				message
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	// Note: success may be false if no import was running, but the mutation should complete without error
	assert.NotEmpty(t, resp.CancelOFLImport.Message)
}

// TestOFLImportedFixturesHaveFadeBehavior tests that fixtures imported from OFL have correct FadeBehavior.
func TestOFLImportedFixturesHaveFadeBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Query fixture definitions with channels to check FadeBehavior
	var resp struct {
		FixtureDefinitions []struct {
			ID           string `json:"id"`
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
			Type         string `json:"type"`
			OFLSourceHash *string `json:"oflSourceHash"`
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
				oflSourceHash
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

	// Find fixtures that have OFL source hash (imported from OFL)
	oflFixtureCount := 0
	for _, def := range resp.FixtureDefinitions {
		if def.OFLSourceHash != nil && *def.OFLSourceHash != "" {
			oflFixtureCount++

			// Check that channels have valid FadeBehavior
			for _, ch := range def.Channels {
				assert.Containsf(t, []string{"FADE", "SNAP", "SNAP_END"}, ch.FadeBehavior,
					"Channel %s in %s/%s should have valid FadeBehavior", ch.Name, def.Manufacturer, def.Model)

				// Discrete channels should have non-fade behavior (SNAP or SNAP_END)
				if ch.IsDiscrete {
					assert.Containsf(t, []string{"SNAP", "SNAP_END"}, ch.FadeBehavior,
						"Discrete channel %s in %s/%s should have SNAP or SNAP_END behavior", ch.Name, def.Manufacturer, def.Model)
				}

				// Certain channel types should have non-fade behavior (SNAP or SNAP_END)
				discreteTypeSet := map[string]bool{
					"STROBE": true, "COLOR_MACRO": true, "GOBO": true,
					"PRISM": true, "EFFECT_SPEED": true, "SHUTTER": true,
				}
				if discreteTypeSet[ch.Type] {
					assert.Containsf(t, []string{"SNAP", "SNAP_END"}, ch.FadeBehavior,
						"Channel type %s (%s in %s/%s) should have SNAP or SNAP_END behavior", ch.Type, ch.Name, def.Manufacturer, def.Model)
				}
			}
		}
	}

	t.Logf("Found %d OFL-imported fixture definitions", oflFixtureCount)
}
