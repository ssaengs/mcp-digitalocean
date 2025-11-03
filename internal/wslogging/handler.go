// wslogging provides a slog.Handler that can optionally send logs to a WebSocket endpoint.
// It is a drop-in replacement for slog.JSONHandler that maintains stderr logging by default,
// but can be configured to send logs to a WebSocket server for centralized log aggregation.
package wslogging

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
	// Set to -1 for unlimited retries (recommended for production)
	// Set to a positive integer (e.g., 5) for limited retries
	maxReconnects = -1
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
	// flushBufferTicker is the interval for polling the buffer during flush
	flushBufferTicker = 100 * time.Millisecond
	// batchInterval is the interval for flushing batched log messages to WebSocket
	batchInterval = 100 * time.Millisecond
	// maxBatchSize is the maximum number of messages to batch before forcing a flush
	maxBatchSize = 50
)

// Handler implements slog.Handler interface with optional WebSocket logging support.
// By default, it logs to the provided io.Writer (typically stderr).
// When configured with a WebSocket URL, it sends logs to both stderr and the WebSocket endpoint.
type Handler struct {
	// standard handler for stderr logging (primary/baseline destination)
	stderrHandler slog.Handler

	// WebSocket configuration
	wsEnabled        bool
	wsURL            string
	wsToken          string
	wsConn           *websocket.Conn
	wsBuffer         chan []byte
	wsFlushTrigger   chan struct{} // signals logWriter to flush batch immediately
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
// If WS_LOGGING_URL environment variable is set, it will be configured to send logs
// to the WebSocket endpoint instead of the writer.
func NewHandler(out io.Writer, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	h := &Handler{
		stderrHandler:    slog.NewJSONHandler(out, opts),
		wsMu:             &sync.Mutex{},
		wsReconnectDelay: reconnectDelay,
		wsMaxReconnects:  maxReconnects,
		closeOnce:        &sync.Once{},
	}

	return h
}

// Enabled reports whether the handler handles records at the given level.
// It delegates to the stderr handler's Enabled method.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stderrHandler.Enabled(ctx, level)
}

// Handle processes a log record.
// Logs are always written to stderr (primary destination).
// If WebSocket logging is enabled, logs are also sent to the WebSocket endpoint asynchronously (complementary destination).
// Both destinations are independent - failure in one does not affect the other.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Check if handler is closed (thread-safe check with mutex)
	h.wsMu.Lock()
	closed := h.closed
	h.wsMu.Unlock()

	if closed {
		return fmt.Errorf("wslogging: handler has been closed and is no longer accepting log messages")
	}

	var stderrErr error

	// log to stderr and WebSocket concurrently (independent operations)
	var wg sync.WaitGroup

	// always write to stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		stderrErr = h.stderrHandler.Handle(ctx, r)
	}()

	// if WebSocket is enabled, also send to WebSocket
	if h.wsEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()

			entry := h.buildLogEntry(r)
			data, err := json.Marshal(entry)
			if err != nil {
				// We use logDiagnostic to stderr here rather than stderrHandler.Handle() because this is infrastructure
				// diagnostic logging about the logging system itself, not application-level logging. Using logDiagnostic keeps this
				// simple, avoids any potential recursion or complexity from trying to log about logging failures, and ensures
				// this diagnostic message will always reach stderr even if there are issues with the handler or its configuration.
				logDiagnostic(os.Stderr, "failed to marshal log entry: %v\n", err)
				return
			}

			// Check again if closed before attempting to send to buffer
			// This prevents panic from writing to a closed channel during shutdown
			h.wsMu.Lock()
			closed := h.closed
			h.wsMu.Unlock()

			if closed {
				return
			}

			// try to send to buffer (non-blocking)
			select {
			case h.wsBuffer <- data:
				// successfully queued for WebSocket transmission
			default:
				// buffer is full, drop the WebSocket message
				// don't set wsErr because this is expected behavior under high load
			}
		}()
	}

	wg.Wait()

	// return stderr error if it failed (primary logging destination)
	// we ignore WebSocket errors as it's a complementary logging destination
	return stderrErr
}

// WithAttrs returns a new Handler with the given attributes added.
// It creates a new handler that shares the WebSocket connection but has updated attributes.
//
// Note: We must return a new handler instance to maintain attribute isolation between loggers.
// When logger.With("key", "value") is called, slog creates a derived logger with its own attributes.
// Each derived logger needs its own handler that knows about its specific attributes, but they all
// share the same WebSocket connection, buffer, and mutex for efficient resource usage.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	newHandler := &Handler{
		stderrHandler: h.stderrHandler.WithAttrs(attrs),
		// WebSocket configuration must be copied so derived handlers maintain WS logging capability
		wsEnabled: h.wsEnabled,
		wsURL:     h.wsURL,
		wsToken:   h.wsToken,
		// these are shared across all derived handlers for efficiency
		wsConn:           h.wsConn,         // shared connection
		wsBuffer:         h.wsBuffer,       // shared buffer
		wsFlushTrigger:   h.wsFlushTrigger, // shared flush trigger
		wsMu:             h.wsMu,           // shared mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		// each derived handler has its own attributes and groups
		attrs:     newAttrs,
		groups:    h.groups,
		closeOnce: h.closeOnce, // shared to ensure Close() runs only once
		closed:    h.closed,
	}

	return newHandler
}

// WithGroup returns a new Handler with the given group name added.
// Subsequent keys will be qualified by the group name.
//
// Note: Similar to WithAttrs, we must return a new handler instance to maintain group isolation.
// Each derived logger needs its own handler that knows about its specific group hierarchy.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(newGroups)-1] = name

	newHandler := &Handler{
		stderrHandler: h.stderrHandler.WithGroup(name),
		// WebSocket configuration must be copied so derived handlers maintain WS logging capability
		wsEnabled: h.wsEnabled,
		wsURL:     h.wsURL,
		wsToken:   h.wsToken,
		// these are shared across all derived handlers for efficiency
		wsConn:           h.wsConn,         // shared connection
		wsBuffer:         h.wsBuffer,       // shared buffer
		wsFlushTrigger:   h.wsFlushTrigger, // shared flush trigger
		wsMu:             h.wsMu,           // shared mutex
		wsReconnectDelay: h.wsReconnectDelay,
		wsMaxReconnects:  h.wsMaxReconnects,
		// each derived handler has its own attributes and groups
		attrs:     h.attrs,
		groups:    newGroups,
		closeOnce: h.closeOnce, // shared to ensure Close() runs only once
		closed:    h.closed,
	}

	return newHandler
}

// ConfigureWebSocket configures WebSocket for logging with the given URL and token.
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
		logDiagnostic(os.Stderr, "WARNING: no authentication token provided - this is a security risk\n")
	}

	h.wsMu.Lock()
	defer h.wsMu.Unlock()

	// store configuration in handler fields so WithAttrs() and WithGroup() can copy them
	// when creating derived handlers (required for maintaining WS logging across logger.With() calls)
	h.wsURL = wsURL
	h.wsToken = token
	h.wsEnabled = true
	h.wsBuffer = make(chan []byte, bufferSize)
	h.wsFlushTrigger = make(chan struct{}, 1) // buffered to prevent blocking

	// log startup diagnostic to stdout
	logDiagnostic(os.Stdout, "configuring WebSocket logging to %s\n", wsURL)
	return nil
}

// Start initiates the WebSocket connection manager and log writer goroutines.
// This method should be called after creating the handler and calling ConfigureWebSocket to enable remote logging.
// The provided context controls the lifecycle of the background goroutines - when the context is cancelled,
// the goroutines will gracefully shut down.
func (h *Handler) Start(ctx context.Context) {
	// start log writer goroutine
	go h.logWriter(ctx)

	// start connection manager goroutine
	go h.connectionManager(ctx)
}

// Close gracefully shuts down the handler and closes the WebSocket connection if open.
// It accepts a context to control the timeout for flushing remaining buffered messages.
// The flush operation polls the buffer every 100ms and completes once the buffer is empty.
//
// Shutdown sequence to prevent race conditions:
// 1. Mark handler as closed (prevents new messages from being queued)
// 2. Flush any messages already in the buffer (with timeout)
// 3. Close the buffer channel (signals logWriter goroutine to exit)
// 4. Close the WebSocket connection (cleanup network resources)
//
// This ordering ensures that no messages are lost during shutdown and prevents panics
// from writing to a closed channel.
//
// This method is safe to call multiple times.
func (h *Handler) Close(ctx context.Context) error {
	var err error
	h.closeOnce.Do(func() {
		// Step 1: Mark as closed to stop accepting new messages
		// Must be done first to prevent race where new messages arrive after flush starts.
		// Protected by mutex to ensure thread-safe access from Handle() and other goroutines.
		// We hold this lock for the entire closure process to make it atomic.
		h.wsMu.Lock()
		defer h.wsMu.Unlock()
		h.closed = true

		// Step 2: Flush remaining buffered messages before tearing down
		// At this point no new messages can be queued (closed=true), so we can safely
		// flush everything that's already in the buffer.
		if h.wsBuffer != nil && h.wsConn != nil {
			err = h.flushBuffer(ctx)
		}

		// Step 3: Close the buffer channel
		// This signals the logWriter goroutine to exit after it drains any remaining messages.
		// Safe to do now because no new messages can be added (closed=true).
		if h.wsBuffer != nil {
			close(h.wsBuffer)
		}

		// Step 4: Close WebSocket connection
		// Final cleanup of network resources after all messages are flushed.
		if h.wsConn != nil {
			// send close message to gracefully shutdown WebSocket
			if closeMsgWriteErr := h.wsConn.WriteMessage(websocket.CloseMessage, []byte{}); closeMsgWriteErr != nil {
				err = fmt.Errorf("failed to send close message: %w", closeMsgWriteErr)
				h.wsConn = nil
				return
			}

			// close the underlying connection
			if connCloseErr := h.wsConn.Close(); connCloseErr != nil {
				err = connCloseErr
				h.wsConn = nil
				return
			}
			h.wsConn = nil
		}
	})
	return err
}

// flushBuffer attempts to flush all buffered messages before shutdown.
// It triggers an immediate flush in the logWriter goroutine and then waits for the buffer to drain.
// The buffer is polled every 100ms and the function returns once it's empty.
// If the provided context has a deadline, it will respect that timeout.
func (h *Handler) flushBuffer(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// trigger immediate flush in logWriter goroutine (non-blocking send)
	select {
	case h.wsFlushTrigger <- struct{}{}:
		// flush triggered
	default:
		// channel full, flush already pending
	}

	ticker := time.NewTicker(flushBufferTicker)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// timeout or cancellation - stop attempting to flush
			remaining := len(h.wsBuffer)
			if remaining > 0 {
				return fmt.Errorf("flush timeout: %d messages remaining in buffer", remaining)
			}
			return ctx.Err()
		case <-ticker.C:
			// check if buffer is empty
			if len(h.wsBuffer) == 0 {
				return nil
			}
			// continue polling
		}
	}
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

// logWriter reads from the buffer and writes to WebSocket with time-based batching.
// Messages are accumulated for up to batchInterval (100ms) or until maxBatchSize is reached,
// then flushed as a batch to reduce websocket write overhead.
// This should be called as a goroutine.
// It respects the provided context and will exit gracefully when the context is cancelled.
func (h *Handler) logWriter(ctx context.Context) {
	batch := make([][]byte, 0, maxBatchSize)
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	// flushBatch writes accumulated messages to the websocket
	flushBatch := func() {
		if len(batch) == 0 {
			return
		}

		// Check if handler is closed (thread-safe check)
		h.wsMu.Lock()
		closed := h.closed
		conn := h.wsConn
		h.wsMu.Unlock()

		if closed {
			return
		}

		// write to WebSocket if connection is available
		if conn != nil {
			// write each message in the batch
			for _, data := range batch {
				h.wsMu.Lock()
				err := h.wsConn.WriteMessage(websocket.TextMessage, data)
				h.wsMu.Unlock()

				if err != nil {
					// connection error will be handled by connectionManager
					// which will set wsConn to nil
					break
				}
			}
		}

		// clear the batch
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// context cancelled - flush remaining messages before shutdown
			flushBatch()
			return

		case data, ok := <-h.wsBuffer:
			if !ok {
				// channel closed - flush remaining messages before shutdown
				flushBatch()
				return
			}

			// add message to batch
			batch = append(batch, data)

			// flush if batch is full
			if len(batch) >= maxBatchSize {
				flushBatch()
			}

		case <-ticker.C:
			// periodic flush
			flushBatch()

		case <-h.wsFlushTrigger:
			// immediate flush requested (e.g., during shutdown)
			flushBatch()
		}
	}
}

// connectionManager manages the WebSocket connection.
// It handles initial connection, reconnection on failure, and monitors connection health.
// This should be called as a goroutine.
// It respects the provided context and will exit gracefully when the context is cancelled.
func (h *Handler) connectionManager(ctx context.Context) {
	var reconnectAttempts int

	for {
		select {
		case <-ctx.Done():
			// context cancelled - graceful shutdown
			return
		default:
		}

		// Check if handler is closed (thread-safe check)
		h.wsMu.Lock()
		closed := h.closed
		h.wsMu.Unlock()

		if closed {
			return
		}

		// attempt to connect
		conn, err := h.connect()
		if err != nil {
			reconnectAttempts++

			// log connection error to stderr
			// if maxReconnects is -1, show "unlimited" instead of a number
			if h.wsMaxReconnects == -1 {
				logDiagnostic(os.Stderr, "connection failed (attempt %d/unlimited): %v\n",
					reconnectAttempts, err)
			} else {
				logDiagnostic(os.Stderr, "connection failed (attempt %d/%d): %v\n",
					reconnectAttempts, h.wsMaxReconnects, err)
			}

			// check if we've exceeded max reconnects (only if not unlimited)
			if h.wsMaxReconnects != -1 && reconnectAttempts > h.wsMaxReconnects {
				logDiagnostic(os.Stderr, "max reconnection attempts reached, giving up\n")
				return
			}

			// wait before retrying, but also check context
			select {
			case <-ctx.Done():
				return
			case <-time.After(h.wsReconnectDelay):
				continue
			}
		}

		// connection successful - reset retry counter
		reconnectAttempts = 0

		// log success to stdout
		logDiagnostic(os.Stdout, "WebSocket connection established to %s\n", h.wsURL)

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
			case <-ctx.Done():
				// context cancelled - graceful shutdown
				break monitorLoop

			case <-pingTicker.C:
				// Check if handler is closed (thread-safe check)
				h.wsMu.Lock()
				closed := h.closed
				h.wsMu.Unlock()

				if closed {
					break monitorLoop
				}

				// send ping
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				if err != nil {
					// ping failed - connection is lost
					logDiagnostic(os.Stderr, "connection lost: %v\n", err)
					break monitorLoop
				}

			case <-readDone:
				// read loop exited - connection is lost
				logDiagnostic(os.Stderr, "connection lost: read error\n")
				break monitorLoop
			}
		}

		// clean up connection
		h.wsMu.Lock()
		if h.wsConn == conn {
			h.wsConn = nil
		}
		h.wsMu.Unlock()
		if err := conn.Close(); err != nil {
			logDiagnostic(os.Stderr, "failed to close connection: %v\n", err)
		}

		// wait a bit before attempting to reconnect, but also check context
		h.wsMu.Lock()
		closed = h.closed
		h.wsMu.Unlock()

		if !closed {
			select {
			case <-ctx.Done():
				return
			case <-time.After(h.wsReconnectDelay):
				// continue to next iteration
			}
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
			if closeErr := resp.Body.Close(); closeErr != nil {
				logDiagnostic(os.Stderr, "failed to close response body after dial error: %v\n", closeErr)
			}
		}
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logDiagnostic(os.Stderr, "failed to close response body: %v\n", closeErr)
		}
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
		// Check if handler is closed (thread-safe check)
		h.wsMu.Lock()
		closed := h.closed
		h.wsMu.Unlock()

		if closed {
			return
		}

		_, _, err := conn.ReadMessage()
		if err != nil {
			// connection closed or error - the connectionManager will handle reconnection
			logDiagnostic(os.Stderr, "read loop error: %v\n", err)
			return
		}
	}
}

// logDiagnostic writes a diagnostic message about the wslogging infrastructure itself.
// These messages are written directly to stdout/stderr rather than through the slog handler to avoid recursion
// or complexity from trying to log about logging failures.
// The messages are formatted as JSON to maintain consistency with application logs.
// The log level is determined by the writer: ERROR for stderr, INFO for stdout.
func logDiagnostic(w io.Writer, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// remove trailing newline if present, as JSON encoding will add its own
	message = strings.TrimSuffix(message, "\n")

	// Determine log level based on the writer
	level := "INFO"
	if w == os.Stderr {
		level = "ERROR"
	}

	entry := map[string]any{
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
		"level":  level,
		"msg":    message,
		"source": "wslogging",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		fmt.Fprintf(w, "[wslogging] %s\n", message)
		return
	}

	fmt.Fprintf(w, "%s\n", data)
}
