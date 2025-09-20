package kick

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"kick-bot/internal/dashboard"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketMessage represents a generic websocket message
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// HandshakeData represents the handshake message data
type HandshakeData struct {
	Message struct {
		ChannelID int `json:"channelId"`
	} `json:"message"`
}

// ConnectionHandler manages a single websocket connection to Kick.com
type ConnectionHandler struct {
	index     int
	channelID int
	token     string
	proxyURL  string
	logger    *logrus.Logger
	conn      *websocket.Conn
}

// NewConnectionHandler creates a new websocket connection handler
func NewConnectionHandler(index, channelID int, token, proxyURL string, logger *logrus.Logger) *ConnectionHandler {
	return &ConnectionHandler{
		index:     index,
		channelID: channelID,
		token:     token,
		proxyURL:  proxyURL,
		logger:    logger,
	}
}

// Start begins the websocket connection and message loop
func (ch *ConnectionHandler) Start(ctx context.Context) error {
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ch.logger.Infof("[%d] Starting connection attempt %d/%d", ch.index, attempt, maxRetries)

		if err := ch.connect(); err != nil {
			ch.logger.WithError(err).Errorf("[%d] Connection attempt %d failed", ch.index, attempt)

			// Wait before retrying
			retryDelay := time.Duration(4+rand.Intn(5)) * time.Second
			ch.logger.Infof("[%d] Retrying in %v...", ch.index, retryDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}

		// If connection successful, start message loop
		return ch.messageLoop(ctx)
	}

	return fmt.Errorf("failed to establish connection after %d attempts", maxRetries)
}

// StartWithDashboard begins the websocket connection with dashboard updates
func (ch *ConnectionHandler) StartWithDashboard(ctx context.Context, dash *dashboard.Dashboard) error {
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Update dashboard with current attempt
		dash.UpdateConnection(ch.index, dashboard.StatusConnecting, attempt, "")

		if err := ch.connect(); err != nil {
			// Update dashboard with retry status
			dash.UpdateConnection(ch.index, dashboard.StatusRetrying, attempt, err.Error())

			// Wait before retrying
			retryDelay := time.Duration(4+rand.Intn(5)) * time.Second

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}

		// Connection successful, update dashboard
		dash.UpdateConnection(ch.index, dashboard.StatusConnected, attempt, "")

		// Start message loop
		return ch.messageLoopWithDashboard(ctx, dash)
	}

	return fmt.Errorf("failed to establish connection after %d attempts", maxRetries)
}

// connect establishes the websocket connection
func (ch *ConnectionHandler) connect() error {
	// Create websocket URL
	wsURL := fmt.Sprintf("wss://websockets.kick.com/viewer/v1/connect?token=%s", ch.token)

	// Parse proxy URL
	proxyURL, err := url.Parse(ch.proxyURL)
	if err != nil {
		return fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	// Create dialer with proxy
	dialer := &websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: 30 * time.Second,
	}

	// Set proxy if provided
	if proxyURL != nil {
		dialer.Proxy = func(*http.Request) (*url.URL, error) {
			return proxyURL, nil
		}
	}

	// Establish websocket connection
	ch.logger.Debugf("[%d] Connecting to %s via proxy %s", ch.index, wsURL, ch.proxyURL)

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	ch.conn = conn
	ch.logger.Infof("[%d] WebSocket connection established", ch.index)
	return nil
}

// messageLoop handles the ping/handshake message cycle
func (ch *ConnectionHandler) messageLoop(ctx context.Context) error {
	defer func() {
		if ch.conn != nil {
			ch.conn.Close()
			ch.logger.Infof("[%d] WebSocket connection closed", ch.index)
		}
	}()

	counter := 0

	for {
		select {
		case <-ctx.Done():
			ch.logger.Infof("[%d] Context cancelled, stopping message loop", ch.index)
			return ctx.Err()
		default:
		}

		counter++

		var message WebSocketMessage

		if counter%2 == 0 {
			// Send ping message
			message = WebSocketMessage{
				Type: "ping",
			}
			ch.logger.Debugf("[%d] Sending ping", ch.index)
		} else {
			// Send handshake message
			handshakeData := HandshakeData{}
			handshakeData.Message.ChannelID = ch.channelID

			message = WebSocketMessage{
				Type: "channel_handshake",
				Data: handshakeData,
			}
			ch.logger.Debugf("[%d] Sending handshake for channel %d", ch.index, ch.channelID)
		}

		// Send message
		if err := ch.conn.WriteJSON(message); err != nil {
			ch.logger.WithError(err).Errorf("[%d] Failed to send message", ch.index)
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Calculate random delay (11-18 seconds)
		delay := time.Duration(11+rand.Intn(8)) * time.Second
		ch.logger.Debugf("[%d] Waiting %v before next message", ch.index, delay)

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			ch.logger.Infof("[%d] Context cancelled during delay", ch.index)
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next iteration
		}
	}
}

// messageLoopWithDashboard handles the ping/handshake message cycle with dashboard updates
func (ch *ConnectionHandler) messageLoopWithDashboard(ctx context.Context, dash *dashboard.Dashboard) error {
	defer func() {
		if ch.conn != nil {
			ch.conn.Close()
		}
	}()

	counter := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		counter++

		var message WebSocketMessage

		if counter%2 == 0 {
			// Send ping message
			message = WebSocketMessage{
				Type: "ping",
			}
		} else {
			// Send handshake message
			handshakeData := HandshakeData{}
			handshakeData.Message.ChannelID = ch.channelID

			message = WebSocketMessage{
				Type: "channel_handshake",
				Data: handshakeData,
			}
		}

		// Send message
		if err := ch.conn.WriteJSON(message); err != nil {
			// Update dashboard with error
			dash.UpdateConnection(ch.index, dashboard.StatusFailed, 1, "Message send failed: "+err.Error())
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Calculate random delay (11-18 seconds)
		delay := time.Duration(11+rand.Intn(8)) * time.Second

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next iteration
		}
	}
}
