// edgelogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package edgelogging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// reconnectDelay is the delay between reconnection attempts
	reconnectDelay = 5 * time.Second
	// maxReconnects is the maximum number of reconnection attempts before giving up
	maxReconnects = 5
	// bufferSize is the size of the log buffer channel
	bufferSize = 1000
	// handshakeTimeout is the timeout for WebSocket handshake
	handshakeTimeout = 10 * time.Second
	// readBufferSize is the WebSocket read buffer size in bytes
	readBufferSize = 4096
	// writeBufferSize is the WebSocket write buffer size in bytes
	writeBufferSize = 4096
	// pingInterval is the interval for sending WebSocket ping frames to keep connection alive
	pingInterval = 30 * time.Second
	// pongWait is the timeout for receiving pong responses
	pongWait = 60 * time.Second
)

// Handler implements slog.Handler interface with optional WebSocket logging support.
// By default, it logs to the provided io.Writer (typically stderr).
// When configured with a WebSocket URL, it sends logs to the WebSocket endpoint instead.
type Handler struct {
	// fallback handler for local logging (stderr)
	fallbackHandler slog.Handler

	// WebSocket configuration
	wsEnabled        bool
	wsURL            string
	wsToken          string
	wsConn           *websocket.Conn
	wsBuffer         chan []byte
	wsMu             *sync.Mutex
	wsReconnectDelay time.Duration
	wsMaxReconnects  int

	// handler state for WithAttrs/WithGroup
	attrs  []slog.Attr
	groups []string

	// lifecycle management
	closeOnce *sync.Once
	closed    bool
}

// NewHandler creates a new Handler that logs to the provided io.Writer.
// If EDGE_LOGGING_URL environment variable is set, it will be configured to send logs
// to the WebSocket endpoint instead of the writer.
func NewHandler(out io.Writer, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	h := &Handler{
		fallbackHandler:  slog.NewJSONHandler(out, opts),
		wsMu:             &sync.Mutex{},
		wsReconnectDelay: reconnectDelay,
		wsMaxReconnects:  maxReconnects,
		closeOnce:        &sync.Once{},
	}

	return h
}

// Enabled reports whether the handler handles records at the given level.
// It delegates to the fallback handler's Enabled method.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.fallbackHandler.Enabled(ctx, level)
}

// Handle processes a log record.
// If WebSocket logging is not enabled, it delegates to the fallback handler (stderr).
// If WebSocket logging is enabled, it sends the log to WebSocket asynchronously.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if h.closed {
		return nil
	}

	// if WebSocket is not enabled, use fallback handler (stderr)
	if !h.wsEnabled {
		return h.fallbackHandler.Handle(ctx, r)
	}

	entry := h.buildLogEntry(r)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// try to send to buffer (non-blocking)
	select {
	case h.wsBuffer <- data:
		return nil
	default:
		// buffer is full, drop the message
		return fmt.Errorf("log buffer full, message dropped")
	}
}

// WithAttrs returns a new Handler with the given attributes added.
// It creates a new handler that shares the WebSocket connection but has updated attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	newHandler := &Handler{
		fallbackHandler:  h.fallbackHandler.WithAttrs(attrs),
		wsEnabled:        h.wsEnabled,
		wsURL:            h.wsURL,
		wsToken:          h.wsToken,
		wsConn:           h.wsConn,
		wsBuffer:         h.wsBuffer,
		wsMu:             h.wsMu, // share the mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		attrs:            newAttrs,
		groups:           h.groups,
		closeOnce:        h.closeOnce, // share the closeOnce
		closed:           h.closed,
	}

	return newHandler
}

// WithGroup returns a new Handler with the given group name added.
// Subsequent keys will be qualified by the group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(newGroups)-1] = name

	newHandler := &Handler{
		fallbackHandler:  h.fallbackHandler.WithGroup(name),
		wsEnabled:        h.wsEnabled,
		wsURL:            h.wsURL,
		wsToken:          h.wsToken,
		wsConn:           h.wsConn,
		wsBuffer:         h.wsBuffer,
		wsMu:             h.wsMu, // share the mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		attrs:            h.attrs,
		groups:           newGroups,
		closeOnce:        h.closeOnce, // share the closeOnce
		closed:           h.closed,
	}

	return newHandler
}

// ConfigureWebSocket enables WebSocket logging with the given URL and token.
// This method should be called after creating the handler to enable remote logging.
// If url is empty, it returns an error.
func (h *Handler) ConfigureWebSocket(wsURL, token string) error {
	if wsURL == "" {
		return fmt.Errorf("WebSocket URL cannot be empty")
	}

	// validate WebSocket URL
	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	// check scheme is ws or wss
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "ws" && scheme != "wss" {
		return fmt.Errorf("invalid WebSocket URL scheme: must be 'ws' or 'wss', got '%s'", parsedURL.Scheme)
	}

	// warn if no token provided
	if token == "" {
		fmt.Fprintf(os.Stderr, "[edgelogging] WARNING: no authentication token provided - this is a security risk\n")
	}

	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	h.wsURL = wsURL
	h.wsToken = token
	h.wsEnabled = true
	h.wsBuffer = make(chan []byte, bufferSize)

	// log startup diagnostic to stdout
	fmt.Fprintf(os.Stdout, "[edgelogging] configuring WebSocket logging to %s\n", wsURL)

	// start log writer goroutine
	go h.logWriter()

	// start connection manager goroutine
	go h.connectionManager()

	return nil
}

// Close gracefully shuts down the handler and closes the WebSocket connection if open.
// It should be called when the application is shutting down.
// This method is safe to call multiple times.
func (h *Handler) Close() error {
	var err error
	h.closeOnce.Do(func() {
		h.closed = true

		// close the buffer channel if it exists
		if h.wsBuffer != nil {
			close(h.wsBuffer)
		}

		// close WebSocket connection if open
		h.wsMu.Lock()
		defer h.wsMu.Unlock()

		if h.wsConn != nil {
			// send close message
			_ = h.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
			err = h.wsConn.Close()
			h.wsConn = nil
		}
	})
	return err
}

// buildLogEntry constructs a map representing the log entry for WebSocket transmission.
func (h *Handler) buildLogEntry(r slog.Record) map[string]any {
	entry := make(map[string]any)

	if !r.Time.IsZero() {
		entry["timestamp"] = r.Time.Format(time.RFC3339Nano)
	}

	entry["level"] = r.Level.String()
	entry["message"] = r.Message

	// add handler-level persistent attributes (from WithAttrs)
	for _, attr := range h.attrs {
		h.addAttrToMap(entry, attr, h.groups)
	}

	// add record-specific attributes (from this log call)
	r.Attrs(func(attr slog.Attr) bool {
		h.addAttrToMap(entry, attr, h.groups)
		return true
	})

	return entry
}

// addAttrToMap adds an attribute to the map, handling groups and nested attributes.
func (h *Handler) addAttrToMap(entry map[string]any, attr slog.Attr, groups []string) {
	attr.Value = attr.Value.Resolve()

	// ignore empty attributes
	if attr.Equal(slog.Attr{}) {
		return
	}

	key := attr.Key

	// navigate to the correct nested map for groups
	current := entry
	for _, group := range groups {
		if _, exists := current[group]; !exists {
			current[group] = make(map[string]any)
		}
		if nested, ok := current[group].(map[string]any); ok {
			current = nested
		}
	}

	// handle different value kinds
	switch attr.Value.Kind() {
	case slog.KindGroup:
		groupAttrs := attr.Value.Group()
		if len(groupAttrs) == 0 {
			return
		}
		if attr.Key != "" {
			current[key] = make(map[string]any)
			if nested, ok := current[key].(map[string]any); ok {
				for _, ga := range groupAttrs {
					h.addAttrToMap(nested, ga, nil)
				}
			}
		} else {
			// inline group
			for _, ga := range groupAttrs {
				h.addAttrToMap(current, ga, nil)
			}
		}
	default:
		current[key] = attr.Value.Any()
	}
}

// logWriter reads from the buffer and writes to WebSocket.
// This should be called as a goroutine.
func (h *Handler) logWriter() {
	for data := range h.wsBuffer {
		if h.closed {
			return
		}

		// lock once, write once
		h.wsMu.Lock()
		if h.wsConn != nil {
			err := h.wsConn.WriteMessage(websocket.TextMessage, data)
			h.wsMu.Unlock()

			if err != nil {
				// connection error will be handled by connectionManager
				// which will set wsConn to nil
				continue
			}
		} else {
			h.wsMu.Unlock()
			// no connection available - message is dropped
		}
	}
}

// connectionManager manages the WebSocket connection.
// It handles initial connection, reconnection on failure, and monitors connection health.
// This should be called as a goroutine.
func (h *Handler) connectionManager() {
	var reconnectAttempts int

	for {
		if h.closed {
			return
		}

		// attempt to connect
		conn, err := h.connect()
		if err != nil {
			reconnectAttempts++

			// log connection error to stderr
			fmt.Fprintf(os.Stderr, "[edgelogging] connection failed (attempt %d/%d): %v\n",
				reconnectAttempts, h.wsMaxReconnects, err)

			if reconnectAttempts > h.wsMaxReconnects {
				fmt.Fprintf(os.Stderr, "[edgelogging] max reconnection attempts reached, giving up\n")
				return
			}

			// wait before retrying
			time.Sleep(h.wsReconnectDelay)
			continue
		}

		// connection successful - reset retry counter
		reconnectAttempts = 0

		// log success to stdout
		fmt.Fprintf(os.Stdout, "[edgelogging] WebSocket connection established to %s\n", h.wsURL)

		h.wsMu.Lock()
		h.wsConn = conn
		h.wsMu.Unlock()

		// start read loop to handle pong responses (detects disconnection via read deadline)
		readDone := make(chan struct{})
		go func() {
			h.readLoop(conn)
			close(readDone)
		}()

		// send periodic pings to keep connection alive
		pingTicker := time.NewTicker(pingInterval)
		defer pingTicker.Stop()

		// monitor connection health
	monitorLoop:
		for {
			select {
			case <-pingTicker.C:
				if h.closed {
					break monitorLoop
				}

				// send ping
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				if err != nil {
					// ping failed - connection is lost
					fmt.Fprintf(os.Stderr, "[edgelogging] connection lost: %v\n", err)
					break monitorLoop
				}

			case <-readDone:
				// read loop exited - connection is lost
				fmt.Fprintf(os.Stderr, "[edgelogging] connection lost: read error\n")
				break monitorLoop
			}
		}

		// clean up connection
		h.wsMu.Lock()
		if h.wsConn == conn {
			h.wsConn = nil
		}
		h.wsMu.Unlock()
		_ = conn.Close()

		// wait a bit before attempting to reconnect
		if !h.closed {
			time.Sleep(h.wsReconnectDelay)
		}
	}
}

// connect establishes a new WebSocket connection with authentication.
func (h *Handler) connect() (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		ReadBufferSize:   readBufferSize,
		WriteBufferSize:  writeBufferSize,
	}

	headers := make(map[string][]string)
	if h.wsToken != "" {
		headers["Authorization"] = []string{fmt.Sprintf("Bearer %s", h.wsToken)}
	}

	conn, resp, err := dialer.Dial(h.wsURL, headers)
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	if resp != nil {
		_ = resp.Body.Close()
	}

	return conn, nil
}

// readLoop reads from the WebSocket to handle control frames (pong) and detect disconnections.
// This method blocks until the connection is closed or an error occurs.
func (h *Handler) readLoop(conn *websocket.Conn) {
	// set up pong handler - extends read deadline when pong is received
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// read loop - we don't expect to receive messages, but we need to read to process pongs
	for {
		if h.closed {
			return
		}

		_, _, err := conn.ReadMessage()
		if err != nil {
			// connection closed or error - the connectionManager will handle reconnection
			return
		}
	}
}
