// Package artnet provides Art-Net packet receiving for DMX capture in tests.
package artnet

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// ArtNetPort is the standard Art-Net UDP port
	ArtNetPort = 6454

	// OpDMX is the Art-Net opcode for DMX data
	OpDMX = 0x5000

	// DMXChannels is the number of channels in a DMX universe
	DMXChannels = 512
)

// Frame represents a captured DMX frame from Art-Net.
type Frame struct {
	Timestamp time.Time
	Universe  int
	Sequence  byte
	Channels  [DMXChannels]byte
}

// Receiver listens for Art-Net packets and captures DMX frames.
type Receiver struct {
	addr   string
	conn   *net.UDPConn
	mu     sync.RWMutex
	frames []Frame
}

// NewReceiver creates a new Art-Net receiver.
// addr should be in the format ":6454" or "0.0.0.0:6454"
func NewReceiver(addr string) *Receiver {
	if addr == "" {
		addr = fmt.Sprintf(":%d", ArtNetPort)
	}
	return &Receiver{
		addr:   addr,
		frames: make([]Frame, 0),
	}
}

// Start begins listening for Art-Net packets.
func (r *Receiver) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", r.addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	r.conn = conn

	go r.receiveLoop()

	return nil
}

// Stop stops the receiver.
func (r *Receiver) Stop() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// CaptureFrames captures Art-Net frames for the specified duration.
func (r *Receiver) CaptureFrames(ctx context.Context, duration time.Duration) ([]Frame, error) {
	if err := r.Start(); err != nil {
		return nil, err
	}
	defer func() { _ = r.Stop() }()

	// Clear any previous frames
	r.mu.Lock()
	r.frames = make([]Frame, 0)
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(duration):
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Frame, len(r.frames))
	copy(result, r.frames)
	return result, nil
}

// GetFrames returns all captured frames.
func (r *Receiver) GetFrames() []Frame {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Frame, len(r.frames))
	copy(result, r.frames)
	return result
}

// ClearFrames clears the captured frames.
func (r *Receiver) ClearFrames() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = make([]Frame, 0)
}

// GetLatestFrame returns the most recent frame for a universe.
func (r *Receiver) GetLatestFrame(universe int) *Frame {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.frames) - 1; i >= 0; i-- {
		if r.frames[i].Universe == universe {
			frame := r.frames[i]
			return &frame
		}
	}
	return nil
}

// GetChannelValue returns the current value of a specific channel.
func (r *Receiver) GetChannelValue(universe, channel int) (byte, bool) {
	frame := r.GetLatestFrame(universe)
	if frame == nil {
		return 0, false
	}
	if channel < 1 || channel > DMXChannels {
		return 0, false
	}
	return frame.Channels[channel-1], true
}

func (r *Receiver) receiveLoop() {
	buf := make([]byte, 1024)

	for {
		if r.conn == nil {
			return
		}

		_ = r.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n < 18 {
			continue // Too short for Art-Net DMX
		}

		frame, ok := parseArtNetPacket(buf[:n])
		if !ok {
			continue
		}

		r.mu.Lock()
		r.frames = append(r.frames, frame)
		r.mu.Unlock()
	}
}

func parseArtNetPacket(data []byte) (Frame, bool) {
	// Check Art-Net header "Art-Net\0"
	if len(data) < 18 {
		return Frame{}, false
	}

	header := string(data[:7])
	if header != "Art-Net" {
		return Frame{}, false
	}

	// Check opcode (little-endian)
	opcode := binary.LittleEndian.Uint16(data[8:10])
	if opcode != OpDMX {
		return Frame{}, false
	}

	// Parse Art-Net DMX packet
	// Offset 10-11: Protocol version (14)
	// Offset 12: Sequence
	// Offset 13: Physical port
	// Offset 14-15: Universe (little-endian)
	// Offset 16-17: Length (big-endian)
	// Offset 18+: DMX data

	sequence := data[12]
	universe := int(binary.LittleEndian.Uint16(data[14:16]))
	length := int(binary.BigEndian.Uint16(data[16:18]))

	if len(data) < 18+length {
		return Frame{}, false
	}

	frame := Frame{
		Timestamp: time.Now(),
		Universe:  universe,
		Sequence:  sequence,
	}

	// Copy DMX data
	copy(frame.Channels[:], data[18:18+length])

	return frame, true
}

// FrameComparator helps compare DMX frames between two sources.
type FrameComparator struct {
	Tolerance int // Maximum allowed difference per channel (default 0 = exact match)
}

// NewFrameComparator creates a new frame comparator.
func NewFrameComparator() *FrameComparator {
	return &FrameComparator{Tolerance: 0}
}

// CompareFrames compares two frames and returns differences.
func (c *FrameComparator) CompareFrames(a, b *Frame) []ChannelDiff {
	if a == nil || b == nil {
		return nil
	}

	var diffs []ChannelDiff

	for i := 0; i < DMXChannels; i++ {
		diff := int(a.Channels[i]) - int(b.Channels[i])
		if diff < 0 {
			diff = -diff
		}
		if diff > c.Tolerance {
			diffs = append(diffs, ChannelDiff{
				Channel:  i + 1,
				ValueA:   a.Channels[i],
				ValueB:   b.Channels[i],
				Diff:     diff,
				Universe: a.Universe,
			})
		}
	}

	return diffs
}

// ChannelDiff represents a difference between two channel values.
type ChannelDiff struct {
	Universe int
	Channel  int
	ValueA   byte
	ValueB   byte
	Diff     int
}

// String returns a human-readable representation of the difference.
func (d ChannelDiff) String() string {
	return fmt.Sprintf("Universe %d Channel %d: %d vs %d (diff: %d)",
		d.Universe, d.Channel, d.ValueA, d.ValueB, d.Diff)
}
