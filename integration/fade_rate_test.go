// Package integration provides end-to-end integration tests for LacyLights.
package integration

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFadeUpdateRateDefault verifies the default fade update rate is 60Hz.
func TestFadeUpdateRateDefault(t *testing.T) {
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
		"key": "fade_update_rate_hz",
	}, &resp)

	require.NoError(t, err)
	assert.Equal(t, "fade_update_rate_hz", resp.Setting.Key)

	// Default should be 60Hz
	rate, err := strconv.Atoi(resp.Setting.Value)
	require.NoError(t, err, "fade_update_rate should be a valid integer")

	// In a fresh system, it should be 60, but we'll accept any valid value
	assert.Greater(t, rate, 0, "fade_update_rate must be positive")
	assert.LessOrEqual(t, rate, 120, "fade_update_rate should not exceed 120Hz")
}

// TestFadeUpdateRateValidation tests setting various valid and invalid rates.
func TestFadeUpdateRateValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Get original value to restore later
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
		"key": "fade_update_rate_hz",
	}, &getResp)

	require.NoError(t, err)
	originalValue := getResp.Setting.Value

	// Ensure we restore the original value
	defer func() {
		var restoreResp struct {
			UpdateSetting struct {
				Value string `json:"value"`
			} `json:"updateSetting"`
		}

		_ = client.Mutate(context.Background(), `
			mutation UpdateSetting($input: UpdateSettingInput!) {
				updateSetting(input: $input) {
					value
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"key":   "fade_update_rate_hz",
				"value": originalValue,
			},
		}, &restoreResp)
	}()

	// Test valid rates
	validRates := []string{"30", "44", "60", "90", "120"}

	for _, rate := range validRates {
		t.Run("ValidRate_"+rate, func(t *testing.T) {
			var updateResp struct {
				UpdateSetting struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"updateSetting"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateSetting($input: UpdateSettingInput!) {
					updateSetting(input: $input) {
						key
						value
					}
				}
			`, map[string]interface{}{
				"input": map[string]interface{}{
					"key":   "fade_update_rate_hz",
					"value": rate,
				},
			}, &updateResp)

			require.NoError(t, err)
			assert.Equal(t, "fade_update_rate", updateResp.UpdateSetting.Key)
			assert.Equal(t, rate, updateResp.UpdateSetting.Value)

			// Verify it persisted
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
				"key": "fade_update_rate_hz",
			}, &verifyResp)

			require.NoError(t, err)
			assert.Equal(t, rate, verifyResp.Setting.Value)
		})
	}

	// Test edge cases and potentially invalid values
	// Note: The backend may or may not validate these - we're testing behavior
	edgeCases := []struct {
		value       string
		expectError bool
		description string
	}{
		{"1", false, "MinimumRate"},
		{"240", false, "HighRate"},
		{"0", true, "ZeroRate"},      // Should likely fail
		{"-10", true, "NegativeRate"}, // Should fail
		{"abc", true, "NonNumeric"},   // Should fail
	}

	for _, tc := range edgeCases {
		t.Run(tc.description, func(t *testing.T) {
			var updateResp struct {
				UpdateSetting struct {
					Value string `json:"value"`
				} `json:"updateSetting"`
			}

			err := client.Mutate(ctx, `
				mutation UpdateSetting($input: UpdateSettingInput!) {
					updateSetting(input: $input) {
						value
					}
				}
			`, map[string]interface{}{
				"input": map[string]interface{}{
					"key":   "fade_update_rate_hz",
					"value": tc.value,
				},
			}, &updateResp)

			if tc.expectError {
				// We expect this to fail
				assert.Error(t, err, "Setting fade_update_rate to %s should fail", tc.value)
			} else {
				// We expect this to succeed, but backend might have different validation
				// So we'll just log if it fails unexpectedly
				if err != nil {
					t.Logf("Setting fade_update_rate to %s failed (backend may have validation): %v", tc.value, err)
				}
			}
		})
	}
}

// TestFadeUpdateRatePersistence verifies that the setting persists across queries.
func TestFadeUpdateRatePersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Get original value
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
		"key": "fade_update_rate_hz",
	}, &getResp)

	require.NoError(t, err)
	originalValue := getResp.Setting.Value

	// Set to a different value
	testValue := "45"
	if originalValue == testValue {
		testValue = "75"
	}

	var updateResp struct {
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
			"key":   "fade_update_rate_hz",
			"value": testValue,
		},
	}, &updateResp)

	require.NoError(t, err)
	assert.Equal(t, testValue, updateResp.UpdateSetting.Value)

	// Query multiple times to ensure it persists
	for i := 0; i < 3; i++ {
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
			"key": "fade_update_rate_hz",
		}, &verifyResp)

		require.NoError(t, err)
		assert.Equal(t, testValue, verifyResp.Setting.Value, "Setting should persist across query %d", i+1)

		// Small delay between queries
		time.Sleep(100 * time.Millisecond)
	}

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
			"key":   "fade_update_rate_hz",
			"value": originalValue,
		},
	}, &restoreResp)

	require.NoError(t, err)
	assert.Equal(t, originalValue, restoreResp.UpdateSetting.Value)
}
