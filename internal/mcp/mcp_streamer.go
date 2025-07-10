package mcp

import (
	"encoding/json"

	"github.com/nicholasflintwillow/github-mcp/internal/logger"
)

// StreamHandlerInterface defines the interface for stream handler operations
type StreamHandlerInterface interface {
	BroadcastMessage(eventType string, data interface{})
	SendToClient(clientID, eventType string, data interface{})
	GetConnectedClients() int
}

// MCPStreamer handles formatting and pushing MCP messages as SSE events to connected clients
type MCPStreamer struct {
	logger        *logger.Logger
	streamHandler StreamHandlerInterface
}

// NewMCPStreamer creates a new MCPStreamer instance
func NewMCPStreamer(logger *logger.Logger, streamHandler StreamHandlerInterface) *MCPStreamer {
	return &MCPStreamer{
		logger:        logger,
		streamHandler: streamHandler,
	}
}

// StreamMessage sends an MCP message to all connected clients
func (ms *MCPStreamer) StreamMessage(message *JSONRPCMessage) error {
	if ms.streamHandler == nil {
		ms.logger.Warn("No stream handler available for streaming message")
		return nil
	}

	// Check if there are any connected clients
	if ms.streamHandler.GetConnectedClients() == 0 {
		ms.logger.Debug("No connected clients to stream message to")
		return nil
	}

	// Format the message for SSE
	eventData, err := ms.formatMessageForSSE(message)
	if err != nil {
		ms.logger.Error("Failed to format MCP message for SSE", "error", err)
		return err
	}

	// Determine event type based on message type
	eventType := ms.getEventType(message)

	// Broadcast to all connected clients
	ms.streamHandler.BroadcastMessage(eventType, eventData)

	ms.logger.Debug("Streamed MCP message to clients",
		"eventType", eventType,
		"messageMethod", message.Method,
		"messageID", message.ID,
		"clientCount", ms.streamHandler.GetConnectedClients())

	return nil
}

// StreamMessageToClient sends an MCP message to a specific client
func (ms *MCPStreamer) StreamMessageToClient(clientID string, message *JSONRPCMessage) error {
	if ms.streamHandler == nil {
		ms.logger.Warn("No stream handler available for streaming message")
		return nil
	}

	// Format the message for SSE
	eventData, err := ms.formatMessageForSSE(message)
	if err != nil {
		ms.logger.Error("Failed to format MCP message for SSE", "error", err, "clientID", clientID)
		return err
	}

	// Determine event type based on message type
	eventType := ms.getEventType(message)

	// Send to specific client
	ms.streamHandler.SendToClient(clientID, eventType, eventData)

	ms.logger.Debug("Streamed MCP message to specific client",
		"eventType", eventType,
		"messageMethod", message.Method,
		"messageID", message.ID,
		"clientID", clientID)

	return nil
}

// StreamNotification sends a notification message to all connected clients
func (ms *MCPStreamer) StreamNotification(method string, params interface{}) error {
	// Create notification message
	notification := NewNotification(method, params)

	return ms.StreamMessage(notification)
}

// StreamToolProgress sends tool execution progress updates to clients
func (ms *MCPStreamer) StreamToolProgress(toolName string, progress interface{}) error {
	// Create a custom progress notification
	progressData := map[string]interface{}{
		"tool":     toolName,
		"progress": progress,
	}

	return ms.StreamNotification("tools/progress", progressData)
}

// StreamError sends error information to clients
func (ms *MCPStreamer) StreamError(errorCode int, message string, data interface{}) error {
	errorData := map[string]interface{}{
		"code":    errorCode,
		"message": message,
		"data":    data,
	}

	if ms.streamHandler != nil {
		ms.streamHandler.BroadcastMessage("error", errorData)
	}

	return nil
}

// formatMessageForSSE formats an MCP message for SSE transmission
func (ms *MCPStreamer) formatMessageForSSE(message *JSONRPCMessage) (map[string]interface{}, error) {
	// Convert message to map for easier manipulation
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	var messageMap map[string]interface{}
	if err := json.Unmarshal(messageBytes, &messageMap); err != nil {
		return nil, err
	}

	// Add metadata for SSE
	eventData := map[string]interface{}{
		"mcp_message": messageMap,
		"timestamp":   getCurrentTimestamp(),
	}

	// Add message type information
	if message.IsRequest() {
		eventData["message_type"] = "request"
	} else if message.IsResponse() {
		eventData["message_type"] = "response"
	} else if message.IsNotification() {
		eventData["message_type"] = "notification"
	}

	return eventData, nil
}

// getEventType determines the SSE event type based on the MCP message
func (ms *MCPStreamer) getEventType(message *JSONRPCMessage) string {
	if message.IsNotification() {
		return "mcp_notification"
	} else if message.IsResponse() {
		if message.IsError() {
			return "mcp_error"
		}
		return "mcp_response"
	} else if message.IsRequest() {
		return "mcp_request"
	}

	return "mcp_message"
}

// getCurrentTimestamp returns the current Unix timestamp
func getCurrentTimestamp() int64 {
	// This would typically use time.Now().Unix()
	// For now, we'll use a placeholder
	return 1640995200 // 2022-01-01 00:00:00 UTC as placeholder
}

// GetConnectedClientsCount returns the number of connected clients
func (ms *MCPStreamer) GetConnectedClientsCount() int {
	if ms.streamHandler == nil {
		return 0
	}
	return ms.streamHandler.GetConnectedClients()
}

// IsStreamingEnabled returns true if streaming is available
func (ms *MCPStreamer) IsStreamingEnabled() bool {
	return ms.streamHandler != nil && ms.streamHandler.GetConnectedClients() > 0
}
