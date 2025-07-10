package mcp

import (
	"sync"
	"testing"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/logger"
)

// mockStreamHandler implements StreamHandlerInterface for testing
type mockStreamHandler struct {
	broadcastCalls []broadcastCall
	clientCalls    []clientCall
	clientCount    int
	mu             sync.Mutex
}

type broadcastCall struct {
	eventType string
	data      interface{}
}

type clientCall struct {
	clientID  string
	eventType string
	data      interface{}
}

func newMockStreamHandler() *mockStreamHandler {
	return &mockStreamHandler{
		broadcastCalls: make([]broadcastCall, 0),
		clientCalls:    make([]clientCall, 0),
		clientCount:    0,
	}
}

func (m *mockStreamHandler) BroadcastMessage(eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastCalls = append(m.broadcastCalls, broadcastCall{
		eventType: eventType,
		data:      data,
	})
}

func (m *mockStreamHandler) SendToClient(clientID, eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientCalls = append(m.clientCalls, clientCall{
		clientID:  clientID,
		eventType: eventType,
		data:      data,
	})
}

func (m *mockStreamHandler) GetConnectedClients() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clientCount
}

func (m *mockStreamHandler) SetConnectedClients(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientCount = count
}

func (m *mockStreamHandler) GetBroadcastCalls() []broadcastCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]broadcastCall{}, m.broadcastCalls...)
}

func (m *mockStreamHandler) GetClientCalls() []clientCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]clientCall{}, m.clientCalls...)
}

// Helper function to create a test logger
func createTestLogger() *logger.Logger {
	logger, _ := logger.New("DEBUG", "text")
	return logger
}

func TestNewMCPStreamer(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()

	streamer := NewMCPStreamer(logger, handler)

	if streamer == nil {
		t.Fatal("NewMCPStreamer returned nil")
	}

	if streamer.logger == nil {
		t.Error("MCPStreamer logger is nil")
	}

	if streamer.streamHandler == nil {
		t.Error("MCPStreamer streamHandler is nil")
	}
}

func TestStreamMessage_Request(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	// Test MCP request message
	message := NewRequest(1, "tools/list", map[string]interface{}{})

	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_request" {
		t.Errorf("Expected event type 'mcp_request', got '%s'", calls[0].eventType)
	}
}

func TestStreamMessage_Response(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	// Test MCP response message
	message := NewResponse(1, map[string]interface{}{"tools": []interface{}{}})

	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_response" {
		t.Errorf("Expected event type 'mcp_response', got '%s'", calls[0].eventType)
	}
}

func TestStreamMessage_Notification(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	// Test MCP notification message
	message := NewNotification("notifications/initialized", map[string]interface{}{})

	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_notification" {
		t.Errorf("Expected event type 'mcp_notification', got '%s'", calls[0].eventType)
	}
}

func TestStreamMessage_Error(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	// Test MCP error message
	message := NewErrorResponse(1, -32601, "Method not found", nil)

	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_error" {
		t.Errorf("Expected event type 'mcp_error', got '%s'", calls[0].eventType)
	}
}

func TestStreamMessage_NoClients(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(0) // No clients connected
	streamer := NewMCPStreamer(logger, handler)

	// Test message with no clients
	message := NewRequest(1, "tools/list", map[string]interface{}{})

	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Should not broadcast when no clients
	calls := handler.GetBroadcastCalls()
	if len(calls) != 0 {
		t.Errorf("Expected 0 broadcast calls when no clients, got %d", len(calls))
	}
}

func TestStreamMessageToClient_Request(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	streamer := NewMCPStreamer(logger, handler)

	clientID := "test-client-123"
	message := NewRequest(1, "tools/list", map[string]interface{}{})

	err := streamer.StreamMessageToClient(clientID, message)
	if err != nil {
		t.Fatalf("StreamMessageToClient failed: %v", err)
	}

	// Verify client-specific call was made
	calls := handler.GetClientCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 client call, got %d", len(calls))
	}

	if calls[0].clientID != clientID {
		t.Errorf("Expected client ID '%s', got '%s'", clientID, calls[0].clientID)
	}

	if calls[0].eventType != "mcp_request" {
		t.Errorf("Expected event type 'mcp_request', got '%s'", calls[0].eventType)
	}
}

func TestStreamMessageToClient_Response(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	streamer := NewMCPStreamer(logger, handler)

	clientID := "test-client-456"
	message := NewResponse(1, map[string]interface{}{"tools": []interface{}{}})

	err := streamer.StreamMessageToClient(clientID, message)
	if err != nil {
		t.Fatalf("StreamMessageToClient failed: %v", err)
	}

	// Verify client-specific call was made
	calls := handler.GetClientCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 client call, got %d", len(calls))
	}

	if calls[0].clientID != clientID {
		t.Errorf("Expected client ID '%s', got '%s'", clientID, calls[0].clientID)
	}

	if calls[0].eventType != "mcp_response" {
		t.Errorf("Expected event type 'mcp_response', got '%s'", calls[0].eventType)
	}
}

func TestStreamNotification(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	err := streamer.StreamNotification("test/notification", map[string]interface{}{"data": "test"})
	if err != nil {
		t.Fatalf("StreamNotification failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_notification" {
		t.Errorf("Expected event type 'mcp_notification', got '%s'", calls[0].eventType)
	}
}

func TestStreamToolProgress(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	err := streamer.StreamToolProgress("test-tool", map[string]interface{}{"percent": 50})
	if err != nil {
		t.Fatalf("StreamToolProgress failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "mcp_notification" {
		t.Errorf("Expected event type 'mcp_notification', got '%s'", calls[0].eventType)
	}
}

func TestStreamError(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	err := streamer.StreamError(-32601, "Method not found", nil)
	if err != nil {
		t.Fatalf("StreamError failed: %v", err)
	}

	// Verify broadcast was called
	calls := handler.GetBroadcastCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
	}

	if calls[0].eventType != "error" {
		t.Errorf("Expected event type 'error', got '%s'", calls[0].eventType)
	}
}

func TestConcurrentStreaming(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	const numGoroutines = 10
	const messagesPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines streaming messages concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				message := NewRequest(
					routineID*messagesPerGoroutine+j,
					"test/method",
					map[string]interface{}{"routine": routineID, "message": j},
				)

				err := streamer.StreamMessage(message)
				if err != nil {
					t.Errorf("StreamMessage failed in goroutine %d: %v", routineID, err)
				}

				// Small delay to increase chance of race conditions
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all messages were broadcast
	calls := handler.GetBroadcastCalls()
	expectedCalls := numGoroutines * messagesPerGoroutine
	if len(calls) != expectedCalls {
		t.Errorf("Expected %d broadcast calls, got %d", expectedCalls, len(calls))
	}

	// Verify all calls have the correct event type
	for i, call := range calls {
		if call.eventType != "mcp_request" {
			t.Errorf("Call %d: expected event type 'mcp_request', got '%s'", i, call.eventType)
		}
	}
}

func TestEventTypeDetection(t *testing.T) {
	logger := createTestLogger()
	handler := newMockStreamHandler()
	handler.SetConnectedClients(1) // Set at least one client
	streamer := NewMCPStreamer(logger, handler)

	testCases := []struct {
		name          string
		message       *JSONRPCMessage
		expectedEvent string
	}{
		{
			name:          "Request with method and id",
			message:       NewRequest(1, "tools/list", nil),
			expectedEvent: "mcp_request",
		},
		{
			name:          "Response with result",
			message:       NewResponse(1, map[string]interface{}{}),
			expectedEvent: "mcp_response",
		},
		{
			name:          "Response with error",
			message:       NewErrorResponse(1, -1, "error", nil),
			expectedEvent: "mcp_error",
		},
		{
			name:          "Notification with method but no id",
			message:       NewNotification("notifications/initialized", nil),
			expectedEvent: "mcp_notification",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear previous calls
			handler.broadcastCalls = nil

			err := streamer.StreamMessage(tc.message)
			if err != nil {
				t.Fatalf("StreamMessage failed: %v", err)
			}

			calls := handler.GetBroadcastCalls()
			if len(calls) != 1 {
				t.Fatalf("Expected 1 broadcast call, got %d", len(calls))
			}

			if calls[0].eventType != tc.expectedEvent {
				t.Errorf("Expected event type '%s', got '%s'", tc.expectedEvent, calls[0].eventType)
			}
		})
	}
}

func TestNilStreamHandler(t *testing.T) {
	logger := createTestLogger()
	streamer := NewMCPStreamer(logger, nil)

	message := NewRequest(1, "tools/list", nil)

	// Should not panic with nil stream handler
	err := streamer.StreamMessage(message)
	if err != nil {
		t.Fatalf("StreamMessage failed with nil handler: %v", err)
	}

	// Should not panic with nil stream handler
	err = streamer.StreamMessageToClient("test", message)
	if err != nil {
		t.Fatalf("StreamMessageToClient failed with nil handler: %v", err)
	}
}
