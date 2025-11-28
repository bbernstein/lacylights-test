// Package websocket provides WebSocket client for GraphQL subscription testing.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// graphql-ws protocol messages
	ConnectionInit      = "connection_init"
	ConnectionAck       = "connection_ack"
	Subscribe           = "subscribe"
	Next                = "next"
	Error               = "error"
	Complete            = "complete"
	ConnectionKeepAlive = "ka"
)

// Client is a WebSocket client for GraphQL subscriptions.
type Client struct {
	endpoint string
	conn     *websocket.Conn
	mu       sync.Mutex
	msgID    int
	handlers map[string]chan *Message
}

// Message represents a graphql-ws protocol message.
type Message struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// SubscriptionPayload represents the payload for a subscription.
type SubscriptionPayload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// NewClient creates a new WebSocket client for GraphQL subscriptions.
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = os.Getenv("GRAPHQL_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "http://localhost:4001/graphql"
	}

	// Convert HTTP URL to WebSocket URL
	wsEndpoint := strings.Replace(endpoint, "http://", "ws://", 1)
	wsEndpoint = strings.Replace(wsEndpoint, "https://", "wss://", 1)

	return &Client{
		endpoint: wsEndpoint,
		handlers: make(map[string]chan *Message),
	}
}

// Connect establishes a WebSocket connection and performs the graphql-ws handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	u, err := url.Parse(c.endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Subprotocols:     []string{"graphql-transport-ws"},
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	// Send connection_init
	initMsg := Message{Type: ConnectionInit}
	if err := c.sendMessage(&initMsg); err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to send connection_init: %w", err)
	}

	// Wait for connection_ack
	ackMsg, err := c.readMessage()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to read connection_ack: %w", err)
	}

	if ackMsg.Type != ConnectionAck {
		_ = conn.Close()
		return fmt.Errorf("expected connection_ack, got %s", ackMsg.Type)
	}

	// Start message handler
	go c.handleMessages()

	return nil
}

// Close closes the WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Subscribe starts a subscription and returns a channel for receiving messages.
func (c *Client) Subscribe(ctx context.Context, query string, variables map[string]interface{}) (<-chan *Message, string, error) {
	c.mu.Lock()
	c.msgID++
	id := fmt.Sprintf("sub_%d", c.msgID)
	c.mu.Unlock()

	payload := SubscriptionPayload{
		Query:     query,
		Variables: variables,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := Message{
		ID:      id,
		Type:    Subscribe,
		Payload: payloadBytes,
	}

	ch := make(chan *Message, 100)

	c.mu.Lock()
	c.handlers[id] = ch
	c.mu.Unlock()

	if err := c.sendMessage(&msg); err != nil {
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()
		close(ch)
		return nil, "", fmt.Errorf("failed to send subscribe: %w", err)
	}

	return ch, id, nil
}

// Unsubscribe stops a subscription.
func (c *Client) Unsubscribe(id string) error {
	msg := Message{
		ID:   id,
		Type: Complete,
	}

	c.mu.Lock()
	if ch, ok := c.handlers[id]; ok {
		close(ch)
		delete(c.handlers, id)
	}
	c.mu.Unlock()

	return c.sendMessage(&msg)
}

// CollectMessages collects subscription messages for a duration.
func (c *Client) CollectMessages(ctx context.Context, query string, variables map[string]interface{}, duration time.Duration) ([]json.RawMessage, error) {
	ch, id, err := c.Subscribe(ctx, query, variables)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Unsubscribe(id) }()

	var messages []json.RawMessage
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return messages, ctx.Err()
		case <-timer.C:
			return messages, nil
		case msg, ok := <-ch:
			if !ok {
				return messages, nil
			}
			if msg.Type == Next {
				messages = append(messages, msg.Payload)
			} else if msg.Type == Error {
				return messages, fmt.Errorf("subscription error: %s", string(msg.Payload))
			} else if msg.Type == Complete {
				return messages, nil
			}
		}
	}
}

func (c *Client) sendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) readMessage() (*Message, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (c *Client) handleMessages() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			// Connection closed
			c.mu.Lock()
			for _, ch := range c.handlers {
				close(ch)
			}
			c.handlers = make(map[string]chan *Message)
			c.mu.Unlock()
			return
		}

		// Skip keep-alive messages
		if msg.Type == ConnectionKeepAlive {
			continue
		}

		c.mu.Lock()
		if ch, ok := c.handlers[msg.ID]; ok {
			select {
			case ch <- msg:
			default:
				// Channel full, skip message
			}

			// Remove handler on complete
			if msg.Type == Complete {
				close(ch)
				delete(c.handlers, msg.ID)
			}
		}
		c.mu.Unlock()
	}
}

// DMXOutputMessage represents a dmxOutputChanged subscription message.
type DMXOutputMessage struct {
	DMXOutputChanged struct {
		Universe int   `json:"universe"`
		Channels []int `json:"channels"`
	} `json:"dmxOutputChanged"`
}

// ParseDMXOutputMessage parses a dmxOutputChanged subscription payload.
func ParseDMXOutputMessage(payload json.RawMessage) (*DMXOutputMessage, error) {
	var wrapper struct {
		Data DMXOutputMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}
