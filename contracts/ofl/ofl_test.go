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
			IsImporting         bool    `json:"isImporting"`
			Phase               string  `json:"phase"`
			TotalFixtures       int     `json:"totalFixtures"`
			ImportedCount       int     `json:"importedCount"`
			FailedCount         int     `json:"failedCount"`
			SkippedCount        int     `json:"skippedCount"`
			PercentComplete     float64 `json:"percentComplete"`
			OFLVersion          *string `json:"oflVersion"`
			UsingBundledData    bool    `json:"usingBundledData"`
			CurrentManufacturer *string `json:"currentManufacturer"`
			CurrentFixture      *string `json:"currentFixture"`
			ErrorMessage        *string `json:"errorMessage"`
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
			CurrentFixtureCount int `json:"currentFixtureCount"`
			OFLFixtureCount     int `json:"oflFixtureCount"`
			NewFixtureCount     int `json:"newFixtureCount"`
			ChangedFixtureCount int `json:"changedFixtureCount"`
			ChangedInUseCount   int `json:"changedInUseCount"`
			FixtureUpdates      []struct {
				FixtureKey    string  `json:"fixtureKey"`
				Manufacturer  string  `json:"manufacturer"`
				Model         string  `json:"model"`
				ChangeType    string  `json:"changeType"`
				IsInUse       bool    `json:"isInUse"`
				InstanceCount int     `json:"instanceCount"`
				CurrentHash   *string `json:"currentHash"`
				NewHash       string  `json:"newHash"`
			} `json:"fixtureUpdates"`
			OFLVersion string `json:"oflVersion"`
			CheckedAt  string `json:"checkedAt"`
		} `json:"checkOFLUpdates"`
	}

	err := client.Query(ctx, `
		query {
			checkOFLUpdates {
				currentFixtureCount
				oflFixtureCount
				newFixtureCount
				changedFixtureCount
				changedInUseCount
				fixtureUpdates {
					fixtureKey
					manufacturer
					model
					changeType
					isInUse
					instanceCount
					currentHash
					newHash
				}
				oflVersion
				checkedAt
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// Counts should be non-negative
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.CurrentFixtureCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.OFLFixtureCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.NewFixtureCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.ChangedFixtureCount, 0)
	assert.GreaterOrEqual(t, resp.CheckOFLUpdates.ChangedInUseCount, 0)

	// fixtureUpdates changeType should be valid
	for _, fixture := range resp.CheckOFLUpdates.FixtureUpdates {
		assert.Contains(t, []string{"NEW", "UPDATED", "UNCHANGED"}, fixture.ChangeType,
			"ChangeType should be NEW, UPDATED, or UNCHANGED")
		assert.NotEmpty(t, fixture.Manufacturer)
		assert.NotEmpty(t, fixture.Model)
		assert.NotEmpty(t, fixture.FixtureKey)
		assert.NotEmpty(t, fixture.NewHash)
	}

	// OFLVersion and CheckedAt should be present
	assert.NotEmpty(t, resp.CheckOFLUpdates.OFLVersion)
	assert.NotEmpty(t, resp.CheckOFLUpdates.CheckedAt)
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
			Success      bool   `json:"success"`
			ErrorMessage string `json:"errorMessage"`
			OFLVersion   string `json:"oflVersion"`
			Stats        struct {
				TotalProcessed    int     `json:"totalProcessed"`
				SuccessfulImports int     `json:"successfulImports"`
				FailedImports     int     `json:"failedImports"`
				SkippedDuplicates int     `json:"skippedDuplicates"`
				UpdatedFixtures   int     `json:"updatedFixtures"`
				DurationSeconds   float64 `json:"durationSeconds"`
			} `json:"stats"`
		} `json:"triggerOFLImport"`
	}

	err = client.Mutate(ctx, `
		mutation TriggerOFLImport($options: OFLImportOptionsInput) {
			triggerOFLImport(options: $options) {
				success
				errorMessage
				oflVersion
				stats {
					totalProcessed
					successfulImports
					failedImports
					skippedDuplicates
					updatedFixtures
					durationSeconds
				}
			}
		}
	`, map[string]interface{}{
		"options": map[string]interface{}{
			"preferBundled": true,
		},
	}, &triggerResp)

	require.NoError(t, err)

	// triggerOFLImport is synchronous and returns final result
	if triggerResp.TriggerOFLImport.Success {
		t.Logf("Import completed: processed=%d, successful=%d, failed=%d, skipped=%d, duration=%.1fs",
			triggerResp.TriggerOFLImport.Stats.TotalProcessed,
			triggerResp.TriggerOFLImport.Stats.SuccessfulImports,
			triggerResp.TriggerOFLImport.Stats.FailedImports,
			triggerResp.TriggerOFLImport.Stats.SkippedDuplicates,
			triggerResp.TriggerOFLImport.Stats.DurationSeconds)
		assert.GreaterOrEqual(t, triggerResp.TriggerOFLImport.Stats.TotalProcessed, 0)
	} else {
		t.Logf("Import failed: %s", triggerResp.TriggerOFLImport.ErrorMessage)
	}

	assert.NotEmpty(t, triggerResp.TriggerOFLImport.OFLVersion)
}

// TestCancelOFLImport tests cancelling an OFL import.
func TestCancelOFLImport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// cancelOFLImport returns Boolean! (not an object)
	var resp struct {
		CancelOFLImport bool `json:"cancelOFLImport"`
	}

	err := client.Mutate(ctx, `
		mutation {
			cancelOFLImport
		}
	`, nil, &resp)

	require.NoError(t, err)
	// Result is just a boolean - true if cancelled, false if no import was running
	t.Logf("Cancel result: %v", resp.CancelOFLImport)
}

// TestOFLImportedFixturesHaveFadeBehavior tests that newly created fixtures have correct FadeBehavior.
// Note: This test creates its own fixture definition with specific channel types to verify
// that the FadeBehavior auto-detection works correctly. Existing fixtures in the database
// may have been imported before FadeBehavior was implemented, so we don't test those.
func TestOFLImportedFixturesHaveFadeBehavior(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Create a fixture definition with channel types that should have specific FadeBehavior
	var createResp struct {
		CreateFixtureDefinition struct {
			ID       string `json:"id"`
			Channels []struct {
				Name         string `json:"name"`
				Type         string `json:"type"`
				FadeBehavior string `json:"fadeBehavior"`
				IsDiscrete   bool   `json:"isDiscrete"`
			} `json:"channels"`
		} `json:"createFixtureDefinition"`
	}

	err := client.Mutate(ctx, `
		mutation CreateFixtureDefinition($input: CreateFixtureDefinitionInput!) {
			createFixtureDefinition(input: $input) {
				id
				channels {
					name
					type
					fadeBehavior
					isDiscrete
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"manufacturer": "Test OFL FadeBehavior",
			"model":        "Auto Detection Test",
			"type":         "LED_PAR",
			"channels": []map[string]interface{}{
				// Channels that should FADE
				{"name": "Dimmer", "type": "INTENSITY", "offset": 0, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Red", "type": "RED", "offset": 1, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Green", "type": "GREEN", "offset": 2, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				{"name": "Blue", "type": "BLUE", "offset": 3, "minValue": 0, "maxValue": 255, "defaultValue": 0},
				// Channels that should SNAP (when isDiscrete is set)
				{"name": "Strobe", "type": "OTHER", "offset": 4, "minValue": 0, "maxValue": 255, "defaultValue": 0, "isDiscrete": true},
				{"name": "Color Macro", "type": "OTHER", "offset": 5, "minValue": 0, "maxValue": 255, "defaultValue": 0, "isDiscrete": true},
			},
		},
	}, &createResp)

	require.NoError(t, err)
	require.NotEmpty(t, createResp.CreateFixtureDefinition.ID)

	// Cleanup
	defer func() {
		_ = client.Mutate(ctx, `mutation DeleteFixtureDefinition($id: ID!) { deleteFixtureDefinition(id: $id) }`,
			map[string]interface{}{"id": createResp.CreateFixtureDefinition.ID}, nil)
	}()

	// Build a map of channels by name for easier testing
	channelMap := make(map[string]struct {
		Type         string
		FadeBehavior string
		IsDiscrete   bool
	})
	for _, ch := range createResp.CreateFixtureDefinition.Channels {
		channelMap[ch.Name] = struct {
			Type         string
			FadeBehavior string
			IsDiscrete   bool
		}{ch.Type, ch.FadeBehavior, ch.IsDiscrete}
	}

	// Check that continuous channels have FADE behavior
	for _, name := range []string{"Dimmer", "Red", "Green", "Blue"} {
		ch, ok := channelMap[name]
		require.True(t, ok, "Channel %s should exist", name)
		assert.Containsf(t, []string{"FADE", "SNAP", "SNAP_END"}, ch.FadeBehavior,
			"Channel %s should have valid FadeBehavior", name)
		// Continuous channels typically default to FADE
		assert.Equal(t, "FADE", ch.FadeBehavior,
			"Continuous channel %s should have FADE behavior", name)
	}

	// Check that discrete channels have SNAP behavior
	for _, name := range []string{"Strobe", "Color Macro"} {
		ch, ok := channelMap[name]
		require.True(t, ok, "Channel %s should exist", name)
		assert.Containsf(t, []string{"FADE", "SNAP", "SNAP_END"}, ch.FadeBehavior,
			"Channel %s should have valid FadeBehavior", name)
		// Discrete channels should have SNAP or SNAP_END behavior
		if ch.IsDiscrete {
			assert.Containsf(t, []string{"SNAP", "SNAP_END"}, ch.FadeBehavior,
				"Discrete channel %s should have SNAP or SNAP_END behavior", name)
		}
	}

	t.Logf("FadeBehavior auto-detection verified for %d channels", len(createResp.CreateFixtureDefinition.Channels))
}
