package wslogging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestNewHandler tests the handler constructor
func TestNewHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.stderrHandler == nil {
		t.Error("stderrHandler is nil")
	}

	if handler.wsEnabled {
		t.Error("wsEnabled should be false by default")
	}
}

// TestHandler_Enabled tests the Enabled method
func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		testWith slog.Level
		want     bool
	}{
		{
			name:     "debug level allows debug",
			level:    slog.LevelDebug,
			testWith: slog.LevelDebug,
			want:     true,
		},
		{
			name:     "info level blocks debug",
			level:    slog.LevelInfo,
			testWith: slog.LevelDebug,
			want:     false,
		},
		{
			name:     "info level allows info",
			level:    slog.LevelInfo,
			testWith: slog.LevelInfo,
			want:     true,
		},
		{
			name:     "error level blocks info",
			level:    slog.LevelError,
			testWith: slog.LevelInfo,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := NewHandler(&buf, &slog.HandlerOptions{
				Level: tt.level,
			})

			got := handler.Enabled(context.Background(), tt.testWith)
			if got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHandler_Handle_Stderr tests logging to stderr (default mode)
func TestHandler_Handle_Stderr(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("output missing log message: %s", output)
	}

	if !strings.Contains(output, "key") {
		t.Errorf("output missing attribute key: %s", output)
	}

	if !strings.Contains(output, "value") {
		t.Errorf("output missing attribute value: %s", output)
	}
}

// TestHandler_WithAttrs tests attribute addition
func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add attributes
	handler2 := handler.WithAttrs([]slog.Attr{
		slog.String("request_id", "123"),
		slog.Int("user_id", 456),
	})

	logger := slog.New(handler2)
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id") {
		t.Errorf("output missing request_id: %s", output)
	}

	if !strings.Contains(output, "123") {
		t.Errorf("output missing request_id value: %s", output)
	}

	if !strings.Contains(output, "user_id") {
		t.Errorf("output missing user_id: %s", output)
	}
}

// TestHandler_WithGroup tests group nesting
func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// add a group
	handler2 := handler.WithGroup("request")
	logger := slog.New(handler2)
	logger.Info("test message", "method", "GET", "path", "/api")

	output := buf.String()
	if !strings.Contains(output, "request") {
		t.Errorf("output missing group name: %s", output)
	}

	if !strings.Contains(output, "method") {
		t.Errorf("output missing attribute in group: %s", output)
	}
}

// TestHandler_Close tests the Close method
func TestHandler_Close(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// close should not error even without WebSocket
	ctx := context.Background()
	err := handler.Close(ctx)
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// multiple closes should be safe
	err = handler.Close(ctx)
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	// handler should be marked as closed
	if !handler.closed {
		t.Error("handler not marked as closed")
	}
}

// mockWebSocketServer creates a test WebSocket server
func mockWebSocketServer(t *testing.T, token string) (*httptest.Server, chan []byte) {
	t.Helper()

	messages := make(chan []byte, 100)
	upgrader := websocket.Upgrader{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check authorization if token provided
		if token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token
			if auth != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// read messages and send to channel
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			messages <- message
		}
	})

	server := httptest.NewServer(handler)
	return server, messages
}

// TestHandler_WebSocket_SendsLogs tests that logs are successfully sent over WebSocket
func TestHandler_WebSocket_SendsLogs(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	// convert http:// to ws://
	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure WebSocket
	err := handler.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handler)
	logger.Info("websocket test", "key", "value")

	// wait for message to arrive
	select {
	case msg := <-messages:
		var logEntry map[string]any
		if err := json.Unmarshal(msg, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}

		if logEntry["message"] != "websocket test" {
			t.Errorf("message = %v, want 'websocket test'", logEntry["message"])
		}

		if logEntry["level"] != "INFO" {
			t.Errorf("level = %v, want 'INFO'", logEntry["level"])
		}

		if logEntry["key"] != "value" {
			t.Errorf("key = %v, want 'value'", logEntry["key"])
		}

		// check for standard fields
		if _, ok := logEntry["timestamp"]; !ok {
			t.Error("missing timestamp field")
		}

	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WebSocket message")
	}
}

// TestHandler_DualLogging tests that logs appear in BOTH stderr and WebSocket when WebSocket is enabled
func TestHandler_DualLogging(t *testing.T) {
	server, messages := mockWebSocketServer(t, "test-token")
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure WebSocket
	err := handler.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handler)
	logger.Info("dual logging test", "destination", "both")

	// verify the log appears in stderr (buf)
	stderrOutput := buf.String()
	if !strings.Contains(stderrOutput, "dual logging test") {
		t.Errorf("stderr missing log message: %s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "destination") {
		t.Errorf("stderr missing attribute key: %s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "both") {
		t.Errorf("stderr missing attribute value: %s", stderrOutput)
	}

	// verify the log also appears in WebSocket
	select {
	case msg := <-messages:
		var logEntry map[string]any
		if err := json.Unmarshal(msg, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal WebSocket log entry: %v", err)
		}

		if logEntry["message"] != "dual logging test" {
			t.Errorf("WebSocket message = %v, want 'dual logging test'", logEntry["message"])
		}

		if logEntry["level"] != "INFO" {
			t.Errorf("WebSocket level = %v, want 'INFO'", logEntry["level"])
		}

		if logEntry["destination"] != "both" {
			t.Errorf("WebSocket destination = %v, want 'both'", logEntry["destination"])
		}

	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for WebSocket message")
	}
}

// TestHandler_WebSocket_WithAuth tests WebSocket with authentication
func TestHandler_WebSocket_WithAuth(t *testing.T) {
	server, messages := mockWebSocketServer(t, "secret-token")
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure with correct token
	err := handler.ConfigureWebSocket(wsURL, "secret-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// wait for connection
	time.Sleep(100 * time.Millisecond)

	// send a log message
	logger := slog.New(handler)
	logger.Info("auth test")

	// should receive message
	select {
	case <-messages:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for authenticated WebSocket message")
	}
}

// TestHandler_ConfigureWebSocket_EmptyURL tests error on empty URL
func TestHandler_ConfigureWebSocket_EmptyURL(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	err := handler.ConfigureWebSocket("", "token")
	if err == nil {
		t.Error("ConfigureWebSocket() with empty URL should return error")
	}
}

// httpToWebSocketURL converts an HTTP test server URL to a WebSocket URL
func httpToWebSocketURL(httpURL string) string {
	// httptest.NewServer returns http:// URLs, but we need ws:// for WebSocket
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

// waitForCondition polls a condition function until it returns true or timeout
func waitForCondition(t *testing.T, timeout time.Duration, checkInterval time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(checkInterval)
	}
	t.Fatal(message)
}

// TestHandler_WebSocket_Reconnection tests reconnection logic
func TestHandler_WebSocket_Reconnection(t *testing.T) {
	// create a channel to track connection attempts
	connectionCount := 0
	var mu sync.Mutex
	messages := make(chan []byte, 10)

	// create server handler
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connectionCount++
		mu.Unlock()

		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// read messages
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			messages <- msg
		}
	})

	server1 := httptest.NewServer(handler1)
	wsURL := httpToWebSocketURL(server1.URL)

	var buf bytes.Buffer
	h := NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// configure WebSocket
	err := h.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	h.Start(ctx)
	defer h.Close(context.Background())

	// wait for initial connection (poll until connected)
	waitForCondition(t, 5*time.Second, 50*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return connectionCount >= 1
	}, "timeout waiting for initial connection")

	// send a log to verify connection works
	logger := slog.New(h)
	logger.Info("before disconnect")

	// wait for message to arrive
	select {
	case <-messages:
		// good, message received
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first message")
	}

	// close the server to simulate disconnection
	server1.Close()

	// start a new server
	server2 := httptest.NewServer(handler1)
	defer server2.Close()

	// reconfigure with new URL (simulates reconnection to new endpoint)
	wsURL2 := httpToWebSocketURL(server2.URL)
	h.Close(context.Background())

	h2 := NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	err = h2.ConfigureWebSocket(wsURL2, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error on reconnect: %v", err)
	}

	// start WebSocket with background context
	ctx2 := context.Background()
	h2.Start(ctx2)
	defer h2.Close(context.Background())

	// wait for reconnection (poll until second connection established)
	waitForCondition(t, 5*time.Second, 50*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return connectionCount >= 2
	}, "timeout waiting for reconnection")

	// send another log
	logger2 := slog.New(h2)
	logger2.Info("after reconnect")

	// wait for message after reconnect
	select {
	case <-messages:
		// good, message received after reconnect
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message after reconnect")
	}

	mu.Lock()
	finalConnections := connectionCount
	mu.Unlock()

	// should have 2 connections total (initial + reconnect)
	if finalConnections != 2 {
		t.Fatalf("expected 2 total connections, got %d", finalConnections)
	}
}

// TestHandler_WebSocket_BufferFull tests behavior when buffer is full
func TestHandler_WebSocket_BufferFull(t *testing.T) {
	// create handler but don't start server
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// configure with invalid URL so connection fails
	_ = handler.ConfigureWebSocket("ws://localhost:1/invalid", "token")

	// start WebSocket with background context
	ctx := context.Background()
	handler.Start(ctx)
	defer handler.Close(context.Background())

	// send many messages rapidly
	logger := slog.New(handler)
	for i := 0; i < 2000; i++ {
		logger.Info("flood test", "i", i)
	}

	// should not panic or block
	// messages should be dropped when buffer is full
}

// TestBuildLogEntry tests the log entry construction
func TestBuildLogEntry(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// create a record
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("key1", "value1"),
		slog.Int("key2", 123),
	)

	entry := handler.buildLogEntry(record)

	if entry["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", entry["message"])
	}

	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want 'INFO'", entry["level"])
	}

	if entry["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", entry["key1"])
	}

	if v, ok := entry["key2"].(int64); !ok || v != 123 {
		t.Errorf("key2 = %v (type %T), want 123", entry["key2"], entry["key2"])
	}
}

// TestConfigureWebSocket_URLValidation tests URL validation in ConfigureWebSocket
func TestConfigureWebSocket_URLValidation(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		token     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid ws URL",
			url:       "ws://localhost:8080",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "valid wss URL",
			url:       "wss://logs.example.com/stream",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "valid wss URL with path",
			url:       "wss://logs.example.com:9000/path/to/logs",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "empty URL",
			url:       "",
			token:     "test-token",
			wantError: true,
			errorMsg:  "WebSocket URL cannot be empty",
		},
		{
			name:      "invalid scheme - http",
			url:       "http://example.com",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme: must be 'ws' or 'wss', got 'http'",
		},
		{
			name:      "invalid scheme - https",
			url:       "https://example.com",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme: must be 'ws' or 'wss', got 'https'",
		},
		{
			name:      "invalid URL format",
			url:       "not-a-url",
			token:     "test-token",
			wantError: true,
			errorMsg:  "invalid WebSocket URL scheme",
		},
		{
			name:      "valid URL with empty token",
			url:       "ws://localhost:8080",
			token:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := NewHandler(&buf, nil)

			err := handler.ConfigureWebSocket(tt.url, tt.token)

			if tt.wantError {
				if err == nil {
					t.Errorf("ConfigureWebSocket() error = nil, want error containing '%s'", tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ConfigureWebSocket() error = %v, want error containing '%s'", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ConfigureWebSocket() unexpected error = %v", err)
					return
				}
				// clean up
				handler.Close(context.Background())
			}
		})
	}
}

// TestHandler_WebSocket_PingPong tests that the handler sends pings and handles pongs
func TestHandler_WebSocket_PingPong(t *testing.T) {
	var mu sync.Mutex
	pingCount := 0
	pongCount := 0
	messages := make(chan []byte, 10)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// set up ping handler to count pings and respond with pongs
		conn.SetPingHandler(func(appData string) error {
			mu.Lock()
			pingCount++
			mu.Unlock()

			// automatically send pong back (gorilla/websocket does this by default)
			err := conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
			if err != nil {
				return err
			}

			mu.Lock()
			pongCount++
			mu.Unlock()

			return nil
		})

		// set read deadline to keep connection alive
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// read messages continuously (required to process ping frames)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// reset read deadline on each message
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// send message to channel
			select {
			case messages <- message:
			default:
				// channel full, skip
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := httpToWebSocketURL(server.URL)

	var buf bytes.Buffer
	h := NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	err := h.ConfigureWebSocket(wsURL, "test-token")
	if err != nil {
		t.Fatalf("ConfigureWebSocket() error: %v", err)
	}

	// start WebSocket with background context
	ctx := context.Background()
	h.Start(ctx)
	defer h.Close(context.Background())

	// wait a moment for connection to be fully established
	time.Sleep(100 * time.Millisecond)

	// send a log message to establish connection
	logger := slog.New(h)
	logger.Info("test message")

	// wait for message
	select {
	case msg := <-messages:
		t.Logf("received message: %s", string(msg))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// wait for at least 1 ping (pingInterval is 30s, but we'll wait up to 35s)
	waitForCondition(t, 35*time.Second, 100*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return pingCount >= 1
	}, "timeout waiting for ping")

	mu.Lock()
	finalPingCount := pingCount
	finalPongCount := pongCount
	mu.Unlock()

	if finalPingCount < 1 {
		t.Errorf("expected at least 1 ping, got %d", finalPingCount)
	}

	if finalPongCount < 1 {
		t.Errorf("expected at least 1 pong, got %d", finalPongCount)
	}

	if finalPingCount != finalPongCount {
		t.Errorf("ping count (%d) should equal pong count (%d)", finalPingCount, finalPongCount)
	}
}
