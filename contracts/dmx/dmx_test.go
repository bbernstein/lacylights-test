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

// skipDMXTests returns true if SKIP_DMX_TESTS or SKIP_FADE_TESTS is set
// DMX tests require Art-Net output to be functioning
func skipDMXTests(t *testing.T) {
	if os.Getenv("SKIP_DMX_TESTS") != "" || os.Getenv("SKIP_FADE_TESTS") != "" {
		t.Skip("Skipping DMX test: SKIP_DMX_TESTS or SKIP_FADE_TESTS is set")
	}
}

// getArtNetPort returns the Art-Net listening address from env or default.
func getArtNetPort() string {
	port := os.Getenv("ARTNET_LISTEN_PORT")
	if port == "" {
		port = "6454"
	}
	// If running locally with localhost broadcast, bind to localhost
	if os.Getenv("ARTNET_BROADCAST") == "127.0.0.1" {
		return "127.0.0.1:" + port
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

	// Query DMX output for universe 1 - returns [Int!]! (512 channel values)
	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	var resp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err := client.Query(ctx, `
		query {
			dmxOutput(universe: 1)
		}
	`, nil, &resp)

	require.NoError(t, err)
	assert.Len(t, resp.DMXOutput, 512)
}

func TestSetChannelValue(t *testing.T) {
	skipDMXTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Set a channel value - returns Boolean
	var setResp struct {
		SetChannelValue bool `json:"setChannelValue"`
	}

	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	err := client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 1, channel: 1, value: 128)
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.True(t, setResp.SetChannelValue)

	// Verify the value was set by querying DMX output
	var dmxResp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 1) }
	`, nil, &dmxResp)
	require.NoError(t, err)
	assert.Equal(t, 128, dmxResp.DMXOutput[0])

	// Reset the channel
	err = client.Mutate(ctx, `
		mutation ResetChannel {
			setChannelValue(universe: 1, channel: 1, value: 0)
		}
	`, nil, &setResp)

	require.NoError(t, err)

	// Verify reset
	err = client.Query(ctx, `
		query { dmxOutput(universe: 1) }
	`, nil, &dmxResp)
	require.NoError(t, err)
	assert.Equal(t, 0, dmxResp.DMXOutput[0])
}

func TestSetMultipleChannels(t *testing.T) {
	skipDMXTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Set multiple channels using individual calls
	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	var setResp struct {
		C1 bool `json:"c1"`
		C2 bool `json:"c2"`
		C3 bool `json:"c3"`
	}

	err := client.Mutate(ctx, `
		mutation SetMultiple {
			c1: setChannelValue(universe: 1, channel: 1, value: 100)
			c2: setChannelValue(universe: 1, channel: 2, value: 150)
			c3: setChannelValue(universe: 1, channel: 3, value: 200)
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.True(t, setResp.C1)
	assert.True(t, setResp.C2)
	assert.True(t, setResp.C3)

	// Verify the values were set
	var dmxResp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 1) }
	`, nil, &dmxResp)
	require.NoError(t, err)
	assert.Equal(t, 100, dmxResp.DMXOutput[0])
	assert.Equal(t, 150, dmxResp.DMXOutput[1])
	assert.Equal(t, 200, dmxResp.DMXOutput[2])

	// Reset the channels
	err = client.Mutate(ctx, `
		mutation ResetMultiple {
			c1: setChannelValue(universe: 1, channel: 1, value: 0)
			c2: setChannelValue(universe: 1, channel: 2, value: 0)
			c3: setChannelValue(universe: 1, channel: 3, value: 0)
		}
	`, nil, nil)

	require.NoError(t, err)
}

func TestBlackout(t *testing.T) {
	skipDMXTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// First set some values
	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	var setResp struct {
		SetChannelValue bool `json:"setChannelValue"`
	}

	err := client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 1, channel: 1, value: 255)
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.True(t, setResp.SetChannelValue)

	// Verify the value was set
	var dmxResp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 1) }
	`, nil, &dmxResp)
	require.NoError(t, err)
	assert.Equal(t, 255, dmxResp.DMXOutput[0])

	// Execute fadeToBlack (instant with 0 second fade)
	var fadeToBlackResp struct {
		FadeToBlack bool `json:"fadeToBlack"`
	}

	err = client.Mutate(ctx, `
		mutation FadeToBlack {
			fadeToBlack(fadeOutTime: 0)
		}
	`, nil, &fadeToBlackResp)

	require.NoError(t, err)
	assert.True(t, fadeToBlackResp.FadeToBlack)

	// Wait for fade engine to process (runs at 40Hz = 25ms interval)
	time.Sleep(100 * time.Millisecond)

	// Verify channels are zero
	var queryResp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query {
			dmxOutput(universe: 1)
		}
	`, nil, &queryResp)

	require.NoError(t, err)
	assert.Equal(t, 0, queryResp.DMXOutput[0])
}

func TestArtNetCaptureDuringChannelChange(t *testing.T) {
	skipDMXTests(t)

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
	// DMX universes are 1-indexed (standard convention: 1-4, not 0-3)
	var setResp struct {
		SetChannelValue bool `json:"setChannelValue"`
	}

	err = client.Mutate(ctx, `
		mutation SetChannel {
			setChannelValue(universe: 1, channel: 10, value: 177)
		}
	`, nil, &setResp)

	require.NoError(t, err)
	assert.True(t, setResp.SetChannelValue)

	// Verify the value was set
	var dmxResp struct {
		DMXOutput []int `json:"dmxOutput"`
	}

	err = client.Query(ctx, `
		query { dmxOutput(universe: 1) }
	`, nil, &dmxResp)
	require.NoError(t, err)
	assert.Equal(t, 177, dmxResp.DMXOutput[9])

	// Wait for Art-Net transmission
	time.Sleep(500 * time.Millisecond)

	// Check captured frames
	frames := receiver.GetFrames()
	if len(frames) == 0 {
		t.Skip("No Art-Net frames captured - Art-Net may not be enabled on server")
	}

	// Find a frame with our value
	// Note: Art-Net uses 0-indexed universe numbers in the protocol
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
			setChannelValue(universe: 1, channel: 10, value: 0)
		}
	`, nil, nil)
}
