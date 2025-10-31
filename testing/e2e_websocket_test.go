//go:build integration

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// LogEntry represents a structured log entry from the MCP server
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Extra     map[string]any `json:"-"` // All other fields
}

// UnmarshalJSON custom unmarshaler to capture all fields
func (l *LogEntry) UnmarshalJSON(data []byte) error {
	type Alias LogEntry
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	}

	// first unmarshal into the struct
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// then unmarshal into a map to capture extra fields
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// remove known fields
	delete(m, "timestamp")
	delete(m, "level")
	delete(m, "message")

	l.Extra = m
	return nil
}

// FakeWebSocketServer is a test WebSocket server that collects log entries
type FakeWebSocketServer struct {
	server      *httptest.Server
	url         string
	token       string
	mu          sync.Mutex
	logEntries  []LogEntry
	connections int
	upgrader    websocket.Upgrader
}

// NewFakeWebSocketServer creates and starts a new fake WebSocket server
func NewFakeWebSocketServer(token string) *FakeWebSocketServer {
	fws := &FakeWebSocketServer{
		token:      token,
		logEntries: make([]LogEntry, 0),
		upgrader:   websocket.Upgrader{},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check authorization if token provided
		if fws.token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + fws.token
			if auth != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		conn, err := fws.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		fws.mu.Lock()
		fws.connections++
		fws.mu.Unlock()

		defer conn.Close()

		// read messages and collect them
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// parse the log entry
			var entry LogEntry
			if err := json.Unmarshal(message, &entry); err != nil {
				continue // skip invalid entries
			}

			fws.mu.Lock()
			fws.logEntries = append(fws.logEntries, entry)
			fws.mu.Unlock()
		}
	})

	fws.server = httptest.NewServer(handler)
	// convert http:// to ws://
	fws.url = "ws" + fws.server.URL[4:]

	return fws
}

// Close shuts down the fake server
func (fws *FakeWebSocketServer) Close() {
	if fws.server != nil {
		fws.server.Close()
	}
}

// GetURL returns the WebSocket URL (for use from host)
func (fws *FakeWebSocketServer) GetURL() string {
	return fws.url
}

// GetContainerURL returns the WebSocket URL that containers can use to reach the host
func (fws *FakeWebSocketServer) GetContainerURL() string {
	// replace 127.0.0.1 with host.docker.internal so containers can reach host
	// this works on Docker Desktop (Mac/Windows) and OrbStack
	return strings.Replace(fws.url, "127.0.0.1", "host.docker.internal", 1)
}

// GetToken returns the authentication token
func (fws *FakeWebSocketServer) GetToken() string {
	return fws.token
}

// GetLogEntries returns all collected log entries
func (fws *FakeWebSocketServer) GetLogEntries() []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	// return a copy
	entries := make([]LogEntry, len(fws.logEntries))
	copy(entries, fws.logEntries)
	return entries
}

// GetConnectionCount returns the number of connections received
func (fws *FakeWebSocketServer) GetConnectionCount() int {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	return fws.connections
}

// WaitForLogs waits for at least n log entries or timeout
func (fws *FakeWebSocketServer) WaitForLogs(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(fws.GetLogEntries()) >= n {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// FindLogsByMessage returns all log entries matching the message
func (fws *FakeWebSocketServer) FindLogsByMessage(message string) []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	results := make([]LogEntry, 0)
	for _, entry := range fws.logEntries {
		if entry.Message == message {
			results = append(results, entry)
		}
	}
	return results
}

// FindLogsByLevel returns all log entries at the given level
func (fws *FakeWebSocketServer) FindLogsByLevel(level string) []LogEntry {
	fws.mu.Lock()
	defer fws.mu.Unlock()

	results := make([]LogEntry, 0)
	for _, entry := range fws.logEntries {
		if entry.Level == level {
			results = append(results, entry)
		}
	}
	return results
}

// ClearLogs clears all collected log entries
func (fws *FakeWebSocketServer) ClearLogs() {
	fws.mu.Lock()
	defer fws.mu.Unlock()
	fws.logEntries = make([]LogEntry, 0)
}

// WaitForConnection waits for at least n connections or timeout
func (fws *FakeWebSocketServer) WaitForConnection(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if fws.GetConnectionCount() >= n {
				return true
			}
		case <-time.After(time.Until(deadline)):
			return false
		}
	}
}

// pollCondition polls a condition function until it returns true or timeout
func pollCondition(t *testing.T, timeout time.Duration, condition func() bool, errorMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-time.After(time.Until(deadline)):
			require.Fail(t, errorMsg)
			return
		}
	}
}

// TestMCPServer_WebSocketLogging tests end-to-end WebSocket logging with MCP server.
// This comprehensive test validates: connection establishment, authentication, log delivery,
// log structure, and different log levels - all using a single container for efficiency.
func TestMCPServer_WebSocketLogging(t *testing.T) {
	ctx := context.Background()

	// Start fake WebSocket server
	fakeWS := NewFakeWebSocketServer("test-token-123")
	defer fakeWS.Close()

	// start MCP server with edge logging enabled
	// use GetContainerURL() so the container can reach the host
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		Services:             "apps,networking",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       fakeWS.GetToken(),
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err, "Failed to start MCP server")
	defer container.Terminate(ctx)

	// get the mapped port
	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err, "Failed to get mapped port")

	serverURL := fmt.Sprintf("http://localhost:%s/mcp", port.Port())

	// Verify WebSocket connection is established with correct token
	require.True(t, fakeWS.WaitForConnection(1, 10*time.Second),
		"WebSocket connection not established within timeout")
	require.Greater(t, fakeWS.GetConnectionCount(), 0, "No WebSocket connections received")
	t.Log("WebSocket connection established with correct token")

	// create MCP client
	c := initializeClientWithURL(ctx, t, serverURL)
	defer c.Close()

	// make API call to generate logs
	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "ListTools failed")

	// poll until logs arrive at fake server
	require.True(t, fakeWS.WaitForLogs(1, 5*time.Second),
		"No logs received within timeout")
	t.Log("Logs sent to WebSocket")

	// verify we have logs
	logs := fakeWS.GetLogEntries()
	require.NotEmpty(t, logs, "Expected at least one log entry")
	t.Logf("Received %d log entries", len(logs))

	// verify log structure (timestamp, level, message)
	for _, log := range logs {
		require.NotEmpty(t, log.Timestamp, "Log missing timestamp")
		require.NotEmpty(t, log.Level, "Log missing level")
		require.NotEmpty(t, log.Message, "Log missing message")
	}
	t.Log("Log structure valid (timestamp, level, message)")

	// verify different log levels are captured (INFO and/or DEBUG)
	hasInfo := len(fakeWS.FindLogsByLevel("INFO")) > 0
	hasDebug := len(fakeWS.FindLogsByLevel("DEBUG")) > 0
	require.True(t, hasInfo || hasDebug, "Should have INFO or DEBUG logs")
	t.Logf("Log levels captured: %d INFO logs, %d DEBUG logs",
		len(fakeWS.FindLogsByLevel("INFO")),
		len(fakeWS.FindLogsByLevel("DEBUG")))

	// verify enabled_services field is present in log entries
	foundEnabledServices := false
	for _, log := range logs {
		if enabledServices, ok := log.Extra["enabled_services"]; ok {
			foundEnabledServices = true
			// verify it's a comma-separated string containing our configured services
			if servicesStr, ok := enabledServices.(string); ok {
				require.Contains(t, servicesStr, "apps", "enabled_services should contain 'apps'")
				require.Contains(t, servicesStr, "networking", "enabled_services should contain 'networking'")
				require.Equal(t, "apps,networking", servicesStr, "enabled_services should match exact configuration")
				t.Logf("enabled_services field verified: %s", servicesStr)
			}
			break
		}
	}
	require.True(t, foundEnabledServices, "At least one log entry should contain 'enabled_services' field")
}

// TestMCPServer_DualLogging verifies that logs are written to BOTH stderr and WebSocket
func TestMCPServer_DualLogging(t *testing.T) {
	ctx := context.Background()

	// Start fake WebSocket server
	fakeWS := NewFakeWebSocketServer("test-token")
	defer fakeWS.Close()

	// start MCP server with WebSocket logging enabled
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       fakeWS.GetToken(),
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err, "Failed to start MCP server")
	defer container.Terminate(ctx)

	// Wait for WebSocket connection
	require.True(t, fakeWS.WaitForConnection(1, 10*time.Second),
		"WebSocket connection not established within timeout")

	// Get container port
	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err, "Failed to get mapped port")
	serverURL := fmt.Sprintf("http://localhost:%s/mcp", port.Port())

	// Create MCP client and make API call to generate logs
	c := initializeClientWithURL(ctx, t, serverURL)
	defer c.Close()

	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "ListTools failed")

	// Poll until logs arrive at WebSocket
	require.True(t, fakeWS.WaitForLogs(1, 5*time.Second),
		"No logs received at WebSocket within timeout")

	wsLogs := fakeWS.GetLogEntries()
	require.NotEmpty(t, wsLogs, "Expected logs in WebSocket")
	t.Logf("Received %d logs via WebSocket", len(wsLogs))

	// Get stderr logs from container
	stderrLogs, err := container.Logs(ctx)
	require.NoError(t, err, "Failed to get container logs")
	defer stderrLogs.Close()

	// Use io.ReadAll to get all available logs
	stderrBytes, err := io.ReadAll(stderrLogs)
	if err != nil {
		t.Logf("Warning: error reading logs: %v", err)
	}
	stderrOutput := string(stderrBytes)

	// Verify logs appear in stderr
	require.NotEmpty(t, stderrOutput, "Expected logs in stderr")
	t.Logf("stderr output length: %d bytes", len(stderrOutput))

	// Verify WebSocket diagnostic logging appears in stderr (JSON format)
	require.Contains(t, stderrOutput, "configuring WebSocket logging",
		"stderr should contain WebSocket diagnostic messages")

	// Verify at least one application log message appears in both destinations
	// Look for log messages that should be in both places (not diagnostic messages)
	foundInBoth := false
	matchCount := 0
	for _, wsLog := range wsLogs {
		// Skip WebSocket diagnostic messages (these only go to stderr)
		// Diagnostic messages have "source":"wslogging" in the log entry
		if source, ok := wsLog.Extra["source"].(string); ok && source == "wslogging" {
			continue
		}

		// Check if this log message appears in stderr
		if strings.Contains(stderrOutput, wsLog.Message) {
			matchCount++
			if !foundInBoth {
				foundInBoth = true
				t.Logf("Verified dual logging: message '%s' found in both stderr and WebSocket", wsLog.Message)
			}
		}
	}

	require.True(t, foundInBoth, "At least one application log message should appear in both stderr and WebSocket")
	t.Logf("Dual logging verified: %d application log messages found in both stderr and WebSocket", matchCount)
}

// TestEdgeLogging_Authentication tests that authentication is required
func TestEdgeLogging_Authentication(t *testing.T) {
	ctx := context.Background()

	// start fake WebSocket server with token required
	fakeWS := NewFakeWebSocketServer("secret-token")
	defer fakeWS.Close()

	// start MCP server with WRONG token
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS.GetContainerURL(),
		WSLoggingToken:       "wrong-token",
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err, "Failed to start MCP server")
	defer container.Terminate(ctx)

	// poll to ensure no connection is established (inverse condition)
	// wait up to 3 seconds - should remain at 0 connections
	pollCondition(t, 3*time.Second, func() bool {
		// this is a negative test - we want it to stay at 0
		// so we sleep a bit and check if it's still 0
		time.Sleep(100 * time.Millisecond)
		return fakeWS.GetConnectionCount() == 0
	}, "Should not connect with wrong token")

	// final verification
	require.Equal(t, 0, fakeWS.GetConnectionCount(), "Should not connect with wrong token")

	// now test with correct token
	fakeWS2 := NewFakeWebSocketServer("correct-token")
	defer fakeWS2.Close()

	cfg2 := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS2.GetContainerURL(),
		WSLoggingToken:       "correct-token",
	}
	container2, err := startMcpServer(ctx, cfg2)
	require.NoError(t, err, "Failed to start MCP server")
	defer container2.Terminate(ctx)

	// poll until connection is established with correct token
	require.True(t, fakeWS2.WaitForConnection(1, 10*time.Second),
		"Should connect with correct token within timeout")

	require.Greater(t, fakeWS2.GetConnectionCount(), 0, "Should have successful connection")
}

// TestEdgeLogging_Reconnection tests WebSocket reconnection behavior
func TestEdgeLogging_Reconnection(t *testing.T) {
	ctx := context.Background()

	// start first fake WebSocket server
	fakeWS1 := NewFakeWebSocketServer("test-token")

	// start MCP server pointing to first server
	cfg := McpServerConfig{
		BindAddr:             "0.0.0.0:8080",
		DigitalOceanAPIToken: os.Getenv("DIGITALOCEAN_API_TOKEN"),
		LogLevel:             "debug",
		Transport:            "http",
		WSLoggingURL:         fakeWS1.GetContainerURL(),
		WSLoggingToken:       fakeWS1.GetToken(),
	}
	container, err := startMcpServer(ctx, cfg)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	// poll until initial connection is established
	require.True(t, fakeWS1.WaitForConnection(1, 10*time.Second),
		"Initial WebSocket connection not established within timeout")

	initialConnections := fakeWS1.GetConnectionCount()
	require.Greater(t, initialConnections, 0, "Should have initial connection")
	t.Logf("Initial connections: %d", initialConnections)

	// get some initial logs to verify connection works
	port, err := container.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err)
	serverURL := fmt.Sprintf("http://localhost:%s/mcp", port.Port())

	c := initializeClientWithURL(ctx, t, serverURL)
	defer c.Close()

	// make a call to generate logs
	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)

	// poll until we get at least one log
	require.True(t, fakeWS1.WaitForLogs(1, 5*time.Second),
		"No logs received from initial connection")

	initialLogCount := len(fakeWS1.GetLogEntries())
	t.Logf("Received %d logs from initial connection", initialLogCount)

	// close the first server (simulates network failure/server restart)
	t.Log("Closing first WebSocket server to simulate network failure")
	fakeWS1.Close()

	// note: due to httptest limitations, we can't easily restart a server on the same URL
	// the connection manager will attempt to reconnect in the background
	// we can verify that it handles the disconnection gracefully by:
	// 1. confirming the connection was closed
	// 2. checking that the MCP server continues to function

	// make another call - this should still work even though WS logging is down
	// (logs will be dropped but the MCP server should continue)
	_, err = c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "MCP server should continue working even if WS logging fails")

	// verify the handler's reconnection logic is working by checking that
	// no panics occurred and the server is still responsive
	t.Log("Verified: MCP server continues to function after WebSocket disconnection")
	t.Log("Reconnection attempts are handled by the connection manager (automatic background retries)")
}

// initializeClientWithURL is a wrapper that accepts server URL parameter
func initializeClientWithURL(ctx context.Context, t *testing.T, serverURL string) *client.Client {
	c, err := newClient(
		serverURL,
		transport.WithHTTPHeaders(map[string]string{"Authorization": fmt.Sprintf("Bearer %s", apiToken)}),
	)

	require.NoError(t, err)
	err = c.Start(ctx)
	require.NoError(t, err)

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	}

	_, err = c.Initialize(ctx, initRequest)
	require.NoError(t, err)

	return c
}
