package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockResponseWriter implements http.ResponseWriter and http.Flusher for testing
type mockResponseWriter struct {
	headers    http.Header
	statusCode int
	body       []byte
	flushed    bool
	mu         sync.Mutex
}

// Ensure mockResponseWriter implements http.Flusher
var _ http.Flusher = (*mockResponseWriter)(nil)

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers:    make(http.Header),
		statusCode: 200,
		body:       make([]byte, 0),
		flushed:    false,
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.body = append(m.body, data...)
	return len(data), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCode = statusCode
}

func (m *mockResponseWriter) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushed = true
}

func (m *mockResponseWriter) GetBody() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.body)
}

func (m *mockResponseWriter) GetStatusCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusCode
}

func (m *mockResponseWriter) WasFlushed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.flushed
}

func TestNewStreamHandler(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	if sh == nil {
		t.Fatal("NewStreamHandler returned nil")
	}

	if sh.GetConnectedClients() != 0 {
		t.Errorf("Expected 0 connected clients, got %d", sh.GetConnectedClients())
	}
}

func TestHandleSSE(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create a mock response writer
	w := newMockResponseWriter()

	// Create a test request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/mcp/stream", nil)
	req = req.WithContext(ctx)

	// Handle the SSE connection in a goroutine since it blocks
	go sh.HandleSSE(w, req)

	// Give some time for the headers to be set and connection to be established
	time.Sleep(50 * time.Millisecond)

	// Verify headers were set correctly
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", w.Header().Get("Content-Type"))
	}

	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got '%s'", w.Header().Get("Cache-Control"))
	}

	if w.Header().Get("Connection") != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got '%s'", w.Header().Get("Connection"))
	}

	// Verify response was flushed
	if !w.WasFlushed() {
		t.Error("Expected response to be flushed")
	}

	// Verify client was registered
	if sh.GetConnectedClients() != 1 {
		t.Errorf("Expected 1 connected client, got %d", sh.GetConnectedClients())
	}
}

func TestBroadcastMessage(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create multiple mock connections
	w1 := newMockResponseWriter()
	w2 := newMockResponseWriter()

	req1 := httptest.NewRequest("GET", "/mcp/stream", nil)
	req2 := httptest.NewRequest("GET", "/mcp/stream", nil)

	// Start connections in goroutines
	go sh.HandleSSE(w1, req1)
	go sh.HandleSSE(w2, req2)

	// Wait for connections to be established
	time.Sleep(50 * time.Millisecond)

	// Verify both clients are connected
	if sh.GetConnectedClients() != 2 {
		t.Errorf("Expected 2 connected clients, got %d", sh.GetConnectedClients())
	}

	// Broadcast a message
	testData := map[string]interface{}{
		"message": "test broadcast",
		"id":      123,
	}

	sh.BroadcastMessage("test", testData)

	// Give some time for the message to be sent
	time.Sleep(50 * time.Millisecond)

	// Verify both writers received the message
	body1 := w1.GetBody()
	body2 := w2.GetBody()

	if !strings.Contains(body1, "event: test") {
		t.Error("Expected first client to receive event type 'test'")
	}

	if !strings.Contains(body2, "event: test") {
		t.Error("Expected second client to receive event type 'test'")
	}

	if !strings.Contains(body1, "test broadcast") {
		t.Error("Expected first client to receive message content")
	}

	if !strings.Contains(body2, "test broadcast") {
		t.Error("Expected second client to receive message content")
	}
}

func TestSendToClient(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create mock connections
	w1 := newMockResponseWriter()
	w2 := newMockResponseWriter()

	req1 := httptest.NewRequest("GET", "/mcp/stream", nil)
	req2 := httptest.NewRequest("GET", "/mcp/stream", nil)

	// Start connections in goroutines
	go sh.HandleSSE(w1, req1)
	go sh.HandleSSE(w2, req2)

	// Wait for connections to be established
	time.Sleep(50 * time.Millisecond)

	// Get client IDs (we'll need to extract them from the connections)
	// For this test, we'll send to the first client we can find
	testData := map[string]interface{}{
		"message": "targeted message",
		"id":      456,
	}

	// Since we can't easily get the client ID in this test setup,
	// we'll test the method exists and doesn't panic
	sh.SendToClient("non-existent-client", "test", testData)

	// Give some time for processing
	time.Sleep(50 * time.Millisecond)

	// The message shouldn't appear in either client since the ID doesn't exist
	body1 := w1.GetBody()
	body2 := w2.GetBody()

	if strings.Contains(body1, "targeted message") {
		t.Error("Message should not have been sent to first client with non-existent ID")
	}

	if strings.Contains(body2, "targeted message") {
		t.Error("Message should not have been sent to second client with non-existent ID")
	}
}

func TestConcurrentConnections(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	const numConnections = 10
	var wg sync.WaitGroup
	wg.Add(numConnections)

	// Create multiple concurrent connections
	for i := 0; i < numConnections; i++ {
		go func(id int) {
			defer wg.Done()

			w := newMockResponseWriter()
			req := httptest.NewRequest("GET", "/mcp/stream", nil)

			// Simulate a short-lived connection
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			req = req.WithContext(ctx)

			sh.HandleSSE(w, req)
		}(i)
	}

	// Wait a bit for connections to establish
	time.Sleep(50 * time.Millisecond)

	// Check that we have the expected number of connections
	connectedClients := sh.GetConnectedClients()
	if connectedClients != numConnections {
		t.Errorf("Expected %d connected clients, got %d", numConnections, connectedClients)
	}

	// Wait for connections to timeout and close
	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Verify connections were cleaned up
	finalClients := sh.GetConnectedClients()
	if finalClients != 0 {
		t.Errorf("Expected 0 connected clients after timeout, got %d", finalClients)
	}
}

func TestHeartbeat(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create a mock connection
	w := newMockResponseWriter()
	req := httptest.NewRequest("GET", "/mcp/stream", nil)

	// Start connection in goroutine
	go sh.HandleSSE(w, req)

	// Wait for connection to be established
	time.Sleep(50 * time.Millisecond)

	// Wait for at least one heartbeat (they occur every 30 seconds by default, but we'll check for the initial setup)
	time.Sleep(100 * time.Millisecond)

	// Verify the connection is still active
	if sh.GetConnectedClients() != 1 {
		t.Errorf("Expected 1 connected client after heartbeat, got %d", sh.GetConnectedClients())
	}

	// Check that some data was written (initial connection message)
	body := w.GetBody()
	if len(body) == 0 {
		t.Error("Expected some data to be written to the connection")
	}
}

func TestConnectionCleanup(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create a connection with a short timeout
	w := newMockResponseWriter()
	req := httptest.NewRequest("GET", "/mcp/stream", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	// Start connection
	go sh.HandleSSE(w, req)

	// Wait for connection to establish
	time.Sleep(25 * time.Millisecond)

	// Verify connection is active
	if sh.GetConnectedClients() != 1 {
		t.Errorf("Expected 1 connected client, got %d", sh.GetConnectedClients())
	}

	// Wait for context to timeout
	time.Sleep(100 * time.Millisecond)

	// Verify connection was cleaned up
	if sh.GetConnectedClients() != 0 {
		t.Errorf("Expected 0 connected clients after cleanup, got %d", sh.GetConnectedClients())
	}
}

func TestBroadcastToNoClients(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Broadcast message when no clients are connected
	testData := map[string]interface{}{
		"message": "no recipients",
	}

	// This should not panic
	sh.BroadcastMessage("test", testData)

	// Verify no clients are connected
	if sh.GetConnectedClients() != 0 {
		t.Errorf("Expected 0 connected clients, got %d", sh.GetConnectedClients())
	}
}

func TestMultipleBroadcasts(t *testing.T) {
	logger := createTestLogger()
	sh := NewStreamHandler(logger)

	// Create a mock connection
	w := newMockResponseWriter()
	req := httptest.NewRequest("GET", "/mcp/stream", nil)

	// Start connection
	go sh.HandleSSE(w, req)

	// Wait for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Send multiple messages
	for i := 0; i < 5; i++ {
		testData := map[string]interface{}{
			"message": fmt.Sprintf("broadcast %d", i),
			"id":      i,
		}
		sh.BroadcastMessage("test", testData)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all messages to be sent
	time.Sleep(100 * time.Millisecond)

	// Verify all messages were received
	body := w.GetBody()
	for i := 0; i < 5; i++ {
		expectedMessage := fmt.Sprintf("broadcast %d", i)
		if !strings.Contains(body, expectedMessage) {
			t.Errorf("Expected to find message '%s' in response body", expectedMessage)
		}
	}
}
