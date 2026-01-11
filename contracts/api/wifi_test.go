// Package api provides API contract tests for WiFi AP Mode functionality.
//
// These tests validate the GraphQL API contracts for WiFi configuration and
// AP (Access Point) mode management. On development machines (macOS) where
// WiFi hardware access is not available, the tests verify the API returns
// appropriate "not available" responses.
//
// On Raspberry Pi with WiFi support, these tests would validate actual WiFi
// operations including AP mode switching, network scanning, and connections.
package api

import (
	"context"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WiFiMode enum values matching GraphQL schema
const (
	WiFiModeClient     = "CLIENT"
	WiFiModeAP         = "AP"
	WiFiModeDisabled   = "DISABLED"
	WiFiModeConnecting = "CONNECTING"
	WiFiModeStartingAP = "STARTING_AP"
)

// TestWiFiStatusQuery tests the wifiStatus query.
// On dev machines, available should be false since there's no WiFi hardware.
// On RPi, this would return actual WiFi status.
func TestWiFiStatusQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		WifiStatus struct {
			Available   bool    `json:"available"`
			Enabled     bool    `json:"enabled"`
			Connected   bool    `json:"connected"`
			SSID        *string `json:"ssid"`
			Mode        *string `json:"mode"`
			IPAddress   *string `json:"ipAddress"`
			MACAddress  *string `json:"macAddress"`
			APConfig    *struct {
				SSID             string `json:"ssid"`
				IPAddress        string `json:"ipAddress"`
				Channel          int    `json:"channel"`
				ClientCount      int    `json:"clientCount"`
				TimeoutMinutes   int    `json:"timeoutMinutes"`
				MinutesRemaining *int   `json:"minutesRemaining"`
			} `json:"apConfig"`
			ConnectedClients []struct {
				MACAddress  string `json:"macAddress"`
				IPAddress   string `json:"ipAddress"`
				Hostname    string `json:"hostname"`
				ConnectedAt string `json:"connectedAt"`
			} `json:"connectedClients"`
		} `json:"wifiStatus"`
	}

	err := client.Query(ctx, `
		query {
			wifiStatus {
				available
				enabled
				connected
				ssid
				mode
				ipAddress
				macAddress
				apConfig {
					ssid
					ipAddress
					channel
					clientCount
					timeoutMinutes
					minutesRemaining
				}
				connectedClients {
					macAddress
					ipAddress
					hostname
					connectedAt
				}
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// The query should succeed regardless of WiFi availability
	// On dev machines, available will be false
	// On RPi, available should be true
	t.Logf("WiFi available: %v", resp.WifiStatus.Available)
	t.Logf("WiFi enabled: %v", resp.WifiStatus.Enabled)
	t.Logf("WiFi mode: %v", resp.WifiStatus.Mode)

	if !resp.WifiStatus.Available {
		// On dev machines, WiFi is not available - verify proper handling
		assert.False(t, resp.WifiStatus.Enabled, "Should not be enabled when not available")
		assert.False(t, resp.WifiStatus.Connected, "Should not be connected when not available")
		assert.Nil(t, resp.WifiStatus.APConfig, "Should have no AP config when not available")
	}
}

// TestWiFiModeQuery tests the wifiMode query.
// Returns the current WiFi mode as a string enum.
func TestWiFiModeQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		WifiMode string `json:"wifiMode"`
	}

	err := client.Query(ctx, `
		query {
			wifiMode
		}
	`, nil, &resp)

	require.NoError(t, err)

	// Mode should be one of the valid enum values
	validModes := []string{WiFiModeClient, WiFiModeAP, WiFiModeDisabled, WiFiModeConnecting, WiFiModeStartingAP}
	assert.Contains(t, validModes, resp.WifiMode, "Mode should be a valid WiFiMode enum value")
	t.Logf("Current WiFi mode: %s", resp.WifiMode)
}

// TestAPConfigQuery tests the apConfig query.
// Returns AP configuration (null when not in AP mode or WiFi unavailable).
func TestAPConfigQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		APConfig *struct {
			SSID             string `json:"ssid"`
			IPAddress        string `json:"ipAddress"`
			Channel          int    `json:"channel"`
			ClientCount      int    `json:"clientCount"`
			TimeoutMinutes   int    `json:"timeoutMinutes"`
			MinutesRemaining *int   `json:"minutesRemaining"`
		} `json:"apConfig"`
	}

	err := client.Query(ctx, `
		query {
			apConfig {
				ssid
				ipAddress
				channel
				clientCount
				timeoutMinutes
				minutesRemaining
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// AP config may be nil if not in AP mode or WiFi unavailable
	if resp.APConfig != nil {
		t.Logf("AP SSID: %s", resp.APConfig.SSID)
		t.Logf("AP IP: %s", resp.APConfig.IPAddress)
		t.Logf("AP Channel: %d", resp.APConfig.Channel)
		t.Logf("AP Clients: %d", resp.APConfig.ClientCount)
		assert.NotEmpty(t, resp.APConfig.SSID, "SSID should not be empty when AP config exists")
		assert.NotEmpty(t, resp.APConfig.IPAddress, "IP address should not be empty when AP config exists")
	} else {
		t.Log("AP config is nil (not in AP mode or WiFi unavailable)")
	}
}

// TestAPClientsQuery tests the apClients query.
// Returns list of connected clients (empty when not in AP mode).
func TestAPClientsQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		APClients []struct {
			MACAddress  string `json:"macAddress"`
			IPAddress   string `json:"ipAddress"`
			Hostname    string `json:"hostname"`
			ConnectedAt string `json:"connectedAt"`
		} `json:"apClients"`
	}

	err := client.Query(ctx, `
		query {
			apClients {
				macAddress
				ipAddress
				hostname
				connectedAt
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// Should return an array (possibly empty)
	assert.NotNil(t, resp.APClients, "APClients should never be nil")
	t.Logf("Connected clients count: %d", len(resp.APClients))
}

// TestWiFiNetworksQuery tests the wifiNetworks query.
// Returns available WiFi networks (empty on dev machines).
func TestWiFiNetworksQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		WifiNetworks []struct {
			SSID           string `json:"ssid"`
			SignalStrength int    `json:"signalStrength"`
			Frequency      string `json:"frequency"`
			Security       string `json:"security"`
			InUse          bool   `json:"inUse"`
			Saved          bool   `json:"saved"`
		} `json:"wifiNetworks"`
	}

	err := client.Query(ctx, `
		query {
			wifiNetworks {
				ssid
				signalStrength
				frequency
				security
				inUse
				saved
			}
		}
	`, nil, &resp)

	// On systems without nmcli (CI/dev machines), this may return a GraphQL error
	// The test validates either: proper error handling OR valid response structure
	if err != nil {
		// Expected error on systems without WiFi/nmcli - should indicate unavailability
		t.Logf("WiFi networks query returned error (expected on CI/dev): %v", err)
		assert.Contains(t, err.Error(), "nmcli", "Error should indicate nmcli dependency")
		return
	}

	// Should return an array (possibly empty on dev machines)
	assert.NotNil(t, resp.WifiNetworks, "WifiNetworks should never be nil")
	t.Logf("Available networks count: %d", len(resp.WifiNetworks))

	for _, network := range resp.WifiNetworks {
		assert.NotEmpty(t, network.SSID, "Network SSID should not be empty")
		assert.GreaterOrEqual(t, network.SignalStrength, 0, "Signal strength should be >= 0")
		assert.LessOrEqual(t, network.SignalStrength, 100, "Signal strength should be <= 100")
	}
}

// TestStartAPModeMutation tests the startAPMode mutation.
// On dev machines, this should fail gracefully since WiFi is not available.
func TestStartAPModeMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		StartAPMode struct {
			Success bool    `json:"success"`
			Message string  `json:"message"`
			Mode    string  `json:"mode"`
		} `json:"startAPMode"`
	}

	err := client.Mutate(ctx, `
		mutation {
			startAPMode {
				success
				message
				mode
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	// On dev machines, should fail gracefully with success=false
	// On RPi, would actually start AP mode
	t.Logf("StartAPMode success: %v", resp.StartAPMode.Success)
	t.Logf("StartAPMode message: %s", resp.StartAPMode.Message)
	t.Logf("StartAPMode mode: %s", resp.StartAPMode.Mode)

	if resp.StartAPMode.Success {
		// If successful, mode should be AP or STARTING_AP
		assert.Contains(t, []string{WiFiModeAP, WiFiModeStartingAP}, resp.StartAPMode.Mode)

		// Clean up: stop AP mode
		t.Log("Cleaning up: stopping AP mode")
		var stopResp struct {
			StopAPMode struct {
				Success bool `json:"success"`
			} `json:"stopAPMode"`
		}
		_ = client.Mutate(ctx, `mutation { stopAPMode { success } }`, nil, &stopResp)
	} else {
		// Should have an informative error message
		assert.NotEmpty(t, resp.StartAPMode.Message, "Should have error message when not successful")
		t.Logf("Expected failure on dev machine: %s", resp.StartAPMode.Message)
	}
}

// TestStopAPModeMutation tests the stopAPMode mutation.
// Should handle being called when not in AP mode gracefully.
func TestStopAPModeMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		StopAPMode struct {
			Success bool    `json:"success"`
			Message string  `json:"message"`
			Mode    string  `json:"mode"`
		} `json:"stopAPMode"`
	}

	err := client.Mutate(ctx, `
		mutation {
			stopAPMode {
				success
				message
				mode
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	t.Logf("StopAPMode success: %v", resp.StopAPMode.Success)
	t.Logf("StopAPMode message: %s", resp.StopAPMode.Message)
	t.Logf("StopAPMode mode: %s", resp.StopAPMode.Mode)

	// Should handle gracefully regardless of current state
	// Mode should be a valid enum value
	validModes := []string{WiFiModeClient, WiFiModeAP, WiFiModeDisabled, WiFiModeConnecting, WiFiModeStartingAP}
	assert.Contains(t, validModes, resp.StopAPMode.Mode, "Mode should be a valid WiFiMode enum value")
}

// TestStopAPModeWithSSIDMutation tests the stopAPMode mutation with connectToSSID parameter.
// This allows stopping AP mode and immediately connecting to a saved network.
func TestStopAPModeWithSSIDMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Test with a non-existent SSID - should fail gracefully
	var resp struct {
		StopAPMode struct {
			Success bool    `json:"success"`
			Message string  `json:"message"`
			Mode    string  `json:"mode"`
		} `json:"stopAPMode"`
	}

	err := client.Mutate(ctx, `
		mutation StopAPMode($connectToSSID: String) {
			stopAPMode(connectToSSID: $connectToSSID) {
				success
				message
				mode
			}
		}
	`, map[string]interface{}{
		"connectToSSID": "NonExistentNetwork_12345",
	}, &resp)

	require.NoError(t, err)

	t.Logf("StopAPMode with SSID success: %v", resp.StopAPMode.Success)
	t.Logf("StopAPMode with SSID message: %s", resp.StopAPMode.Message)

	// Should handle gracefully - may fail but shouldn't error
}

// TestResetAPTimeoutMutation tests the resetAPTimeout mutation.
// Resets the 30-minute AP mode timeout back to full duration.
func TestResetAPTimeoutMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		ResetAPTimeout bool `json:"resetAPTimeout"`
	}

	err := client.Mutate(ctx, `
		mutation {
			resetAPTimeout
		}
	`, nil, &resp)

	require.NoError(t, err)

	// Should return a boolean - true if successful, false otherwise
	t.Logf("ResetAPTimeout result: %v", resp.ResetAPTimeout)
}

// TestConnectWiFiMutation tests the connectWiFi mutation.
// On dev machines, this should fail gracefully.
func TestConnectWiFiMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		ConnectWiFi struct {
			Success   bool    `json:"success"`
			Message   string  `json:"message"`
			Connected bool    `json:"connected"`
		} `json:"connectWiFi"`
	}

	err := client.Mutate(ctx, `
		mutation ConnectWiFi($ssid: String!, $password: String) {
			connectWiFi(ssid: $ssid, password: $password) {
				success
				message
				connected
			}
		}
	`, map[string]interface{}{
		"ssid":     "TestNetwork",
		"password": "testpassword",
	}, &resp)

	require.NoError(t, err)

	t.Logf("ConnectWiFi success: %v", resp.ConnectWiFi.Success)
	t.Logf("ConnectWiFi message: %s", resp.ConnectWiFi.Message)

	// On dev machines, should fail gracefully
	// On RPi with WiFi, would attempt actual connection
}

// TestDisconnectWiFiMutation tests the disconnectWiFi mutation.
func TestDisconnectWiFiMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		DisconnectWiFi struct {
			Success   bool    `json:"success"`
			Message   string  `json:"message"`
			Connected bool    `json:"connected"`
		} `json:"disconnectWiFi"`
	}

	err := client.Mutate(ctx, `
		mutation {
			disconnectWiFi {
				success
				message
				connected
			}
		}
	`, nil, &resp)

	require.NoError(t, err)

	t.Logf("DisconnectWiFi success: %v", resp.DisconnectWiFi.Success)
	t.Logf("DisconnectWiFi message: %s", resp.DisconnectWiFi.Message)
}

// TestSetWiFiEnabledMutation tests the setWiFiEnabled mutation.
func TestSetWiFiEnabledMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		SetWiFiEnabled struct {
			Available bool `json:"available"`
			Enabled   bool `json:"enabled"`
			Connected bool `json:"connected"`
		} `json:"setWiFiEnabled"`
	}

	err := client.Mutate(ctx, `
		mutation SetWiFiEnabled($enabled: Boolean!) {
			setWiFiEnabled(enabled: $enabled) {
				available
				enabled
				connected
			}
		}
	`, map[string]interface{}{
		"enabled": true,
	}, &resp)

	// On systems without nmcli (CI/dev machines), this may return a GraphQL error
	// The test validates either: proper error handling OR valid response structure
	if err != nil {
		// Expected error on systems without WiFi/nmcli - should indicate unavailability
		t.Logf("SetWiFiEnabled returned error (expected on CI/dev): %v", err)
		assert.Contains(t, err.Error(), "nmcli", "Error should indicate nmcli dependency")
		return
	}

	t.Logf("SetWiFiEnabled available: %v", resp.SetWiFiEnabled.Available)
	t.Logf("SetWiFiEnabled enabled: %v", resp.SetWiFiEnabled.Enabled)

	// On dev machines, available should be false and enabled may not change
}

// TestForgetWiFiNetworkMutation tests the forgetWiFiNetwork mutation.
func TestForgetWiFiNetworkMutation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	var resp struct {
		ForgetWiFiNetwork bool `json:"forgetWiFiNetwork"`
	}

	err := client.Mutate(ctx, `
		mutation ForgetWiFiNetwork($ssid: String!) {
			forgetWiFiNetwork(ssid: $ssid)
		}
	`, map[string]interface{}{
		"ssid": "NonExistentNetwork_12345",
	}, &resp)

	require.NoError(t, err)

	// Should return a boolean indicating success
	t.Logf("ForgetWiFiNetwork result: %v", resp.ForgetWiFiNetwork)
}
