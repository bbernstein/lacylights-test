// Package settings provides contract tests for system settings.
package settings

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFadeUpdateRateQuery tests querying the fade_update_rate setting.
//
// GraphQL Schema Assumptions:
// - Query: setting(key: String!): Setting
// - Setting type has: key: String!, value: String!
// - Default fade_update_rate is "60" (60Hz)
func TestFadeUpdateRateQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		Setting struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"setting"`
	}

	err := client.Query(ctx, `
		query GetSetting($key: String!) {
			setting(key: $key) {
				key
				value
			}
		}
	`, map[string]interface{}{
		"key": "fade_update_rate",
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, "fade_update_rate", resp.Setting.Key)

	// Default should be 60Hz
	assert.NotEmpty(t, resp.Setting.Value, "fade_update_rate should have a value")

	// Value should be parseable as a number (we'll validate range in integration tests)
	// For contract tests, we just verify the structure is correct
}

// TestFadeUpdateRateMutation tests updating the fade_update_rate setting.
//
// GraphQL Schema Assumptions:
// - Mutation: updateSetting(input: UpdateSettingInput!): Setting
// - UpdateSettingInput has: key: String!, value: String!
// - Setting type has: key: String!, value: String!
func TestFadeUpdateRateMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// First, get the current value to restore later
	var getResp struct {
		Setting struct {
			Value string `json:"value"`
		} `json:"setting"`
	}

	err := client.Query(ctx, `
		query GetSetting($key: String!) {
			setting(key: $key) {
				value
			}
		}
	`, map[string]interface{}{
		"key": "fade_update_rate",
	}, &getResp)

	require.NoError(t, err)
	originalValue := getResp.Setting.Value

	// Update to a new value
	var updateResp struct {
		UpdateSetting struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"updateSetting"`
	}

	err = client.Mutate(ctx, `
		mutation UpdateSetting($input: UpdateSettingInput!) {
			updateSetting(input: $input) {
				key
				value
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":   "fade_update_rate",
			"value": "30",
		},
	}, &updateResp)

	require.NoError(t, err)
	assert.Equal(t, "fade_update_rate", updateResp.UpdateSetting.Key)
	assert.Equal(t, "30", updateResp.UpdateSetting.Value)

	// Verify the change persisted by reading it back
	var verifyResp struct {
		Setting struct {
			Value string `json:"value"`
		} `json:"setting"`
	}

	err = client.Query(ctx, `
		query GetSetting($key: String!) {
			setting(key: $key) {
				value
			}
		}
	`, map[string]interface{}{
		"key": "fade_update_rate",
	}, &verifyResp)

	require.NoError(t, err)
	assert.Equal(t, "30", verifyResp.Setting.Value)

	// Restore original value
	var restoreResp struct {
		UpdateSetting struct {
			Value string `json:"value"`
		} `json:"updateSetting"`
	}

	err = client.Mutate(ctx, `
		mutation UpdateSetting($input: UpdateSettingInput!) {
			updateSetting(input: $input) {
				value
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"key":   "fade_update_rate",
			"value": originalValue,
		},
	}, &restoreResp)

	require.NoError(t, err)
	assert.Equal(t, originalValue, restoreResp.UpdateSetting.Value)
}

// TestAllSettingsQuery tests querying all settings at once.
//
// GraphQL Schema Assumptions:
// - Query: settings: [Setting!]!
// - Setting type has: key: String!, value: String!
func TestAllSettingsQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		Settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"settings"`
	}

	err := client.Query(ctx, `
		query {
			settings {
				key
				value
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.NotNil(t, resp.Settings)

	// Verify fade_update_rate is in the list
	found := false
	for _, setting := range resp.Settings {
		if setting.Key == "fade_update_rate" {
			found = true
			assert.NotEmpty(t, setting.Value)
			break
		}
	}
	assert.True(t, found, "fade_update_rate should be in settings list")
}
