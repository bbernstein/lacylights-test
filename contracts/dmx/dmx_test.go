// Package dmx provides DMX behavior contract tests.
package dmx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/artnet"
	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getArtNetPort returns the Art-Net listening port from env or default.
func getArtNetPort() string {
	port := os.Getenv("ARTNET_LISTEN_PORT")
	if port == "" {
		port = "6454"
	}
	return ":" + port
}

func TestArtNetReceiver(t *testing.T) {
	// This test verifies the Art-Net receiver works
	receiver := artnet.NewReceiver(getArtNetPort())

	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver (port may be in use): %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	// Just verify it starts without error
	assert.NotNil(t, receiver)
}

func TestDMXOutputMatchesQuery(t *testing.T) {
	// This test verifies that the GraphQL DMX output query returns data
	// that can be validated against Art-Net capture (when enabled)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Query DMX output for universe 0
	var resp struct {
		DMXOutput struct {
			Universe int   `json:"universe"`
			Channels []int `json:"channels"`
		} `json:"dmxOutput"`
	}

	err := client.Query(ctx, `
		query {
			dmxOutput(universe: 0) {
				universe
				channels
			}
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.Equal(t, 0, resp.DMXOutput.Universe)
	assert.Len(t, resp.DMXOutput.Channels, 512)
}

func TestSetChannelValue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Set a channel value
	var setResp struct {
		SetChannelValue struct {
			Universe int   `json:"universe"`
			Channels []int `json:"channels"`
		} `json:"setChannelValue"`
	}

	err := client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 0, channel: 1, value: 128) {
				universe
				channels
			}
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.Equal(t, 0, setResp.SetChannelValue.Universe)
	assert.Equal(t, 128, setResp.SetChannelValue.Channels[0])

	// Reset the channel
	err = client.Mutate(ctx, `
		mutation ResetChannel {
			setChannelValue(universe: 0, channel: 1, value: 0) {
				universe
				channels
			}
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.Equal(t, 0, setResp.SetChannelValue.Channels[0])
}

func TestSetMultipleChannels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Set multiple channels
	var setResp struct {
		SetMultipleChannelValues struct {
			Universe int   `json:"universe"`
			Channels []int `json:"channels"`
		} `json:"setMultipleChannelValues"`
	}

	err := client.Mutate(ctx, `
		mutation SetMultiple {
			setMultipleChannelValues(universe: 0, startChannel: 1, values: [100, 150, 200]) {
				universe
				channels
			}
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.Equal(t, 100, setResp.SetMultipleChannelValues.Channels[0])
	assert.Equal(t, 150, setResp.SetMultipleChannelValues.Channels[1])
	assert.Equal(t, 200, setResp.SetMultipleChannelValues.Channels[2])

	// Reset the channels
	err = client.Mutate(ctx, `
		mutation ResetMultiple {
			setMultipleChannelValues(universe: 0, startChannel: 1, values: [0, 0, 0]) {
				universe
				channels
			}
		}
	`, nil, nil)

	require.NoError(t, err)
}

func TestBlackout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// First set some values
	var setResp struct {
		SetChannelValue struct {
			Channels []int `json:"channels"`
		} `json:"setChannelValue"`
	}

	err := client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 0, channel: 1, value: 255) {
				channels
			}
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.Equal(t, 255, setResp.SetChannelValue.Channels[0])

	// Execute blackout
	var blackoutResp struct {
		Blackout bool `json:"blackout"`
	}

	err = client.Mutate(ctx, `
		mutation Blackout {
			blackout
		}
	`, nil, &blackoutResp)

	require.NoError(t, err)
	assert.True(t, blackoutResp.Blackout)

	// Verify channels are zero
	var queryResp struct {
		DMXOutput struct {
			Channels []int `json:"channels"`
		} `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query {
			dmxOutput(universe: 0) {
				channels
			}
		}
	`, nil, &queryResp)

	require.NoError(t, err)
	assert.Equal(t, 0, queryResp.DMXOutput.Channels[0])
}

func TestArtNetCaptureDuringChannelChange(t *testing.T) {
	// This test captures Art-Net packets while changing a channel value
	// to verify DMX output is actually being transmitted

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start Art-Net receiver
	receiver := artnet.NewReceiver(getArtNetPort())
	err := receiver.Start()
	if err != nil {
		t.Skipf("Could not start Art-Net receiver (port may be in use or Art-Net disabled): %v", err)
	}
	defer func() { _ = receiver.Stop() }()

	client := graphql.NewClient("")

	// Clear any previous frames
	receiver.ClearFrames()

	// Set a distinctive value
	var setResp struct {
		SetChannelValue struct {
			Channels []int `json:"channels"`
		} `json:"setChannelValue"`
	}

	err = client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 0, channel: 10, value: 177) {
				channels
			}
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.Equal(t, 177, setResp.SetChannelValue.Channels[9])

	// Wait for Art-Net transmission
	time.Sleep(500 * time.Millisecond)

	// Check captured frames
	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured - Art-Net may not be enabled on server")
	}

	// Find a frame with our value
	found := false
	for _, frame := range frames {
		if frame.Universe == 0 && frame.Channels[9] == 177 {
			found = true
			break
		}
	}

	assert.True(t, found, "Should capture Art-Net frame with channel 10 = 177")

	// Clean up
	_ = client.Mutate(ctx, `
		mutation ResetChannel {
			setChannelValue(universe: 0, channel: 10, value: 0) {
				channels
			}
		}
	`, nil, nil)
}
