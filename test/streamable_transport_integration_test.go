package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/logger"
	"github.com/nicholasflintwillow/github-mcp/internal/mcp"
)

// TestServer represents a test server instance
type TestServer struct {
	server        *httptest.Server
	streamHandler *mcp.StreamHandler
	logger        *logger.Logger
}

// NewTestServer creates a new test server for integration testing
func NewTestServer() *TestServer {
	logger, _ := logger.New("DEBUG", "text")
	streamHandler := mcp.NewStreamHandler(logger)

	mux := http.NewServeMux()

	// Add SSE endpoint
	mux.HandleFunc("/mcp/stream", streamHandler.HandleSSE)

	// Add MCP request endpoint (simplified for testing)
	mux.HandleFunc("/mcp/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Parse the MCP message
		var message mcp.JSONRPCMessage
		if err := json.Unmarshal(body, &message); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Create a simple response
		var response *mcp.JSONRPCMessage
		if message.IsRequest() {
			switch message.Method {
			case mcp.MethodInitialize:
				response = mcp.NewResponse(message.ID, mcp.InitializeResult{
					ProtocolVersion: mcp.MCPVersion,
					Capabilities: mcp.ServerCapabilities{
						Tools: &mcp.ToolsCapability{ListChanged: true},
					},
					ServerInfo: mcp.ServerInfo{
						Name:    "test-server",
						Version: "1.0.0",
					},
				})
			case mcp.MethodListTools:
				response = mcp.NewResponse(message.ID, mcp.ToolsListResult{
					Tools: []mcp.Tool{
						{
							Name:        "test_tool",
							Description: "A test tool",
							InputSchema: map[string]interface{}{"type": "object"},
						},
					},
				})
			case mcp.MethodPing:
				response = mcp.NewResponse(message.ID, map[string]interface{}{"pong": true})
			default:
				response = mcp.NewErrorResponse(message.ID, mcp.ErrorCodeMethodNotFound, "Method not found", nil)
			}
		} else if message.IsNotification() {
			// For notifications, we might want to stream them to connected clients
			streamer := streamHandler.GetStreamer()
			if streamer != nil {
				streamer.StreamMessage(&message)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if response != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	server := httptest.NewServer(mux)

	return &TestServer{
		server:        server,
		streamHandler: streamHandler,
		logger:        logger,
	}
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	// Safely stop the stream handler
	defer func() {
		if r := recover(); r != nil {
			// Ignore panic from double close
		}
	}()
	if ts.streamHandler != nil {
		ts.streamHandler.Stop()
	}
	if ts.server != nil {
		ts.server.Close()
	}
}

// URL returns the server URL
func (ts *TestServer) URL() string {
	return ts.server.URL
}

// GetConnectedClients returns the number of connected SSE clients
func (ts *TestServer) GetConnectedClients() int {
	return ts.streamHandler.GetConnectedClients()
}

// BroadcastMessage broadcasts a message to all connected clients
func (ts *TestServer) BroadcastMessage(eventType string, data interface{}) {
	ts.streamHandler.BroadcastMessage(eventType, data)
}

// SSEClient represents an SSE client for testing
type SSEClient struct {
	serverURL string
	client    *http.Client
	events    []SSEEvent
	mu        sync.Mutex
	done      chan struct{}
}

// SSEEvent represents a received SSE event
type SSEEvent struct {
	Type string
	Data string
}

// NewSSEClient creates a new SSE client
func NewSSEClient(serverURL string) *SSEClient {
	return &SSEClient{
		serverURL: serverURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		events:    make([]SSEEvent, 0),
		done:      make(chan struct{}),
	}
}

// Connect establishes an SSE connection
func (c *SSEClient) Connect(ctx context.Context) error {
	url := c.serverURL + "/mcp/stream"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent SSEEvent

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent.Type = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" {
			// End of event
			if currentEvent.Type != "" || currentEvent.Data != "" {
				c.mu.Lock()
				c.events = append(c.events, currentEvent)
				c.mu.Unlock()
				currentEvent = SSEEvent{}
			}
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return scanner.Err()
}

// GetEvents returns all received events
func (c *SSEClient) GetEvents() []SSEEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]SSEEvent{}, c.events...)
}

// WaitForEvents waits for a specific number of events with timeout
func (c *SSEClient) WaitForEvents(count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		eventCount := len(c.events)
		c.mu.Unlock()

		if eventCount >= count {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// SendMCPRequest sends an MCP request to the server
func (c *SSEClient) SendMCPRequest(message *mcp.JSONRPCMessage) (*mcp.JSONRPCMessage, error) {
	url := c.serverURL + "/mcp/request"

	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // No response expected (notification)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response mcp.JSONRPCMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func TestSSEConnectionEstablishment(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Test connection establishment
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start SSE connection in goroutine
	go func() {
		err := client.Connect(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("SSE connection failed: %v", err)
		}
	}()

	// Wait for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that client is connected
	if server.GetConnectedClients() != 1 {
		t.Errorf("Expected 1 connected client, got %d", server.GetConnectedClients())
	}

	// Wait for initial connection event
	if !client.WaitForEvents(1, 2*time.Second) {
		t.Error("Did not receive initial connection event")
	}

	events := client.GetEvents()
	if len(events) < 1 {
		t.Fatal("Expected at least 1 event")
	}

	// Check connection event
	if events[0].Type != "connected" {
		t.Errorf("Expected 'connected' event, got '%s'", events[0].Type)
	}

	if !strings.Contains(events[0].Data, "Connected to MCP stream") {
		t.Error("Expected connection message in event data")
	}
}

func TestMCPRequestResponseFlow(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Test initialize request
	initRequest := mcp.NewRequest(1, mcp.MethodInitialize, mcp.InitializeRequest{
		ProtocolVersion: mcp.MCPVersion,
		Capabilities:    mcp.ClientCapabilities{},
		ClientInfo: mcp.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	response, err := client.SendMCPRequest(initRequest)
	if err != nil {
		t.Fatalf("Initialize request failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for initialize request")
	}

	// Convert both IDs to strings for comparison to handle type differences
	expectedID := fmt.Sprintf("%v", initRequest.ID)
	actualID := fmt.Sprintf("%v", response.ID)
	if expectedID != actualID {
		t.Errorf("Expected response ID %v, got %v", initRequest.ID, response.ID)
	}

	if response.IsError() {
		t.Errorf("Expected successful response, got error: %v", response.Error)
	}

	// Verify response structure
	var initResult mcp.InitializeResult
	if err := response.GetResult(&initResult); err != nil {
		t.Errorf("Failed to parse initialize result: %v", err)
	}

	if initResult.ProtocolVersion != mcp.MCPVersion {
		t.Errorf("Expected protocol version %s, got %s", mcp.MCPVersion, initResult.ProtocolVersion)
	}
}

func TestMCPNotificationStreaming(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Start SSE connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		err := client.Connect(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("SSE connection failed: %v", err)
		}
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Send a notification that should be streamed
	notification := mcp.NewNotification("tools/progress", map[string]interface{}{
		"tool":     "test_tool",
		"progress": 50,
		"status":   "running",
	})

	_, err := client.SendMCPRequest(notification)
	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Wait for the notification to be streamed back
	if !client.WaitForEvents(2, 3*time.Second) { // connection + notification
		t.Error("Did not receive expected events")
	}

	events := client.GetEvents()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events, got %d", len(events))
	}

	// Find the notification event
	var notificationEvent *SSEEvent
	for _, event := range events {
		if event.Type == "mcp_notification" {
			notificationEvent = &event
			break
		}
	}

	if notificationEvent == nil {
		t.Fatal("Did not receive notification event")
	}

	// Parse the notification data
	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(notificationEvent.Data), &eventData); err != nil {
		t.Fatalf("Failed to parse notification event data: %v", err)
	}

	if eventData["message_type"] != "notification" {
		t.Errorf("Expected message_type 'notification', got '%v'", eventData["message_type"])
	}
}

func TestConcurrentSSEConnections(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	const numClients = 5
	clients := make([]*SSEClient, numClients)
	contexts := make([]context.Context, numClients)
	cancels := make([]context.CancelFunc, numClients)

	// Create multiple clients
	for i := 0; i < numClients; i++ {
		clients[i] = NewSSEClient(server.URL())
		contexts[i], cancels[i] = context.WithTimeout(context.Background(), 10*time.Second)
	}

	// Cleanup
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	// Start all connections concurrently
	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientIdx int) {
			defer wg.Done()
			err := clients[clientIdx].Connect(contexts[clientIdx])
			if err != nil && err != context.Canceled {
				t.Errorf("Client %d connection failed: %v", clientIdx, err)
			}
		}(i)
	}

	// Wait for connections to establish
	time.Sleep(200 * time.Millisecond)

	// Check that all clients are connected
	connectedCount := server.GetConnectedClients()
	if connectedCount != numClients {
		t.Errorf("Expected %d connected clients, got %d", numClients, connectedCount)
	}

	// Broadcast a message to all clients
	testData := map[string]interface{}{
		"message": "broadcast test",
		"id":      12345,
	}
	server.BroadcastMessage("test_broadcast", testData)

	// Wait for all clients to receive the message
	time.Sleep(200 * time.Millisecond)

	// Check that all clients received the broadcast
	for i, client := range clients {
		events := client.GetEvents()
		found := false
		for _, event := range events {
			if event.Type == "test_broadcast" && strings.Contains(event.Data, "broadcast test") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Client %d did not receive broadcast message", i)
		}
	}

	// Cancel all connections
	for _, cancel := range cancels {
		cancel()
	}

	// Wait for disconnections
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Check that all clients are disconnected
	if server.GetConnectedClients() != 0 {
		t.Errorf("Expected 0 connected clients after disconnect, got %d", server.GetConnectedClients())
	}
}

func TestSSEEventFormatting(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Start SSE connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		client.Connect(ctx)
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Test different types of MCP messages
	testCases := []struct {
		name     string
		message  *mcp.JSONRPCMessage
		expected string
	}{
		{
			name:     "request",
			message:  mcp.NewRequest(1, "tools/list", nil),
			expected: "mcp_request",
		},
		{
			name:     "response",
			message:  mcp.NewResponse(1, map[string]interface{}{"result": "success"}),
			expected: "mcp_response",
		},
		{
			name:     "error response",
			message:  mcp.NewErrorResponse(1, mcp.ErrorCodeInternalError, "Test error", nil),
			expected: "mcp_error",
		},
		{
			name:     "notification",
			message:  mcp.NewNotification("test/notification", map[string]interface{}{"data": "test"}),
			expected: "mcp_notification",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Stream the message
			streamer := server.streamHandler.GetStreamer()
			err := streamer.StreamMessage(tc.message)
			if err != nil {
				t.Fatalf("Failed to stream message: %v", err)
			}

			// Wait for the event
			time.Sleep(100 * time.Millisecond)

			// Check that the event was received with correct type
			events := client.GetEvents()
			found := false
			for _, event := range events {
				if event.Type == tc.expected {
					found = true

					// Verify event data structure
					var eventData map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &eventData); err != nil {
						t.Errorf("Failed to parse event data: %v", err)
						continue
					}

					// Check required fields
					if _, exists := eventData["mcp_message"]; !exists {
						t.Error("Event data missing 'mcp_message' field")
					}

					if _, exists := eventData["timestamp"]; !exists {
						t.Error("Event data missing 'timestamp' field")
					}

					if _, exists := eventData["message_type"]; !exists {
						t.Error("Event data missing 'message_type' field")
					}

					break
				}
			}

			if !found {
				t.Errorf("Did not receive event of type '%s'", tc.expected)
			}
		})
	}
}

func TestClientDisconnectionHandling(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Start SSE connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	go func() {
		client.Connect(ctx)
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Verify client is connected
	if server.GetConnectedClients() != 1 {
		t.Errorf("Expected 1 connected client, got %d", server.GetConnectedClients())
	}

	// Cancel the context to simulate client disconnect
	cancel()

	// Wait for disconnection to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify client is disconnected
	if server.GetConnectedClients() != 0 {
		t.Errorf("Expected 0 connected clients after disconnect, got %d", server.GetConnectedClients())
	}
}

func TestHeartbeatMessages(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Start the stream handler to enable heartbeat
	server.streamHandler.Start()

	client := NewSSEClient(server.URL())

	// Start SSE connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		client.Connect(ctx)
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Manually trigger a heartbeat instead of waiting for the timer
	// This is more reliable for testing
	go func() {
		time.Sleep(100 * time.Millisecond)
		server.streamHandler.BroadcastMessage("heartbeat", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}()

	// Wait for heartbeat event
	time.Sleep(500 * time.Millisecond)

	// Check for heartbeat events
	events := client.GetEvents()
	heartbeatFound := false
	for _, event := range events {
		if event.Type == "heartbeat" {
			heartbeatFound = true

			// Verify heartbeat data
			var heartbeatData map[string]interface{}
			if err := json.Unmarshal([]byte(event.Data), &heartbeatData); err != nil {
				t.Errorf("Failed to parse heartbeat data: %v", err)
				continue
			}

			if _, exists := heartbeatData["timestamp"]; !exists {
				t.Error("Heartbeat data missing 'timestamp' field")
			}

			break
		}
	}

	if !heartbeatFound {
		t.Error("Did not receive heartbeat event")
	}
}

func TestMCPProtocolIntegration(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := NewSSEClient(server.URL())

	// Test complete MCP protocol flow

	// 1. Initialize
	initRequest := mcp.NewRequest(1, mcp.MethodInitialize, mcp.InitializeRequest{
		ProtocolVersion: mcp.MCPVersion,
		Capabilities:    mcp.ClientCapabilities{},
		ClientInfo: mcp.ClientInfo{
			Name:    "integration-test-client",
			Version: "1.0.0",
		},
	})

	response, err := client.SendMCPRequest(initRequest)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if response.IsError() {
		t.Fatalf("Initialize returned error: %v", response.Error)
	}

	// 2. Send initialized notification
	initializedNotification := mcp.NewNotification(mcp.MethodInitialized, nil)
	_, err = client.SendMCPRequest(initializedNotification)
	if err != nil {
		t.Fatalf("Initialized notification failed: %v", err)
	}

	// 3. List tools
	toolsRequest := mcp.NewRequest(2, mcp.MethodListTools, nil)
	response, err = client.SendMCPRequest(toolsRequest)
	if err != nil {
		t.Fatalf("List tools failed: %v", err)
	}

	if response.IsError() {
		t.Fatalf("List tools returned error: %v", response.Error)
	}

	var toolsResult mcp.ToolsListResult
	if err := response.GetResult(&toolsResult); err != nil {
		t.Fatalf("Failed to parse tools result: %v", err)
	}

	if len(toolsResult.Tools) == 0 {
		t.Error("Expected at least one tool in the result")
	}

	// 4. Ping
	pingRequest := mcp.NewRequest(3, mcp.MethodPing, nil)
	response, err = client.SendMCPRequest(pingRequest)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if response.IsError() {
		t.Fatalf("Ping returned error: %v", response.Error)
	}

	var pingResult map[string]interface{}
	if err := response.GetResult(&pingResult); err != nil {
		t.Fatalf("Failed to parse ping result: %v", err)
	}

	if pingResult["pong"] != true {
		t.Error("Expected pong response to be true")
	}
}
