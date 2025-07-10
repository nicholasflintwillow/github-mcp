package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/logger"
)

// ClientConnection represents an active SSE client connection
type ClientConnection struct {
	ID       string
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Done     chan struct{}
	LastSeen time.Time
}

// StreamHandler manages SSE connections and handles streaming MCP messages to clients
type StreamHandler struct {
	logger     *logger.Logger
	clients    map[string]*ClientConnection
	clientsMux sync.RWMutex
	streamer   *MCPStreamer
	heartbeat  time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewStreamHandler creates a new StreamHandler instance
func NewStreamHandler(logger *logger.Logger) *StreamHandler {
	sh := &StreamHandler{
		logger:    logger,
		clients:   make(map[string]*ClientConnection),
		heartbeat: 30 * time.Second, // Send heartbeat every 30 seconds
		stopCh:    make(chan struct{}),
	}

	// Create MCPStreamer with reference to this handler
	sh.streamer = NewMCPStreamer(logger, sh)

	return sh
}

// Start begins the background processes for the stream handler
func (sh *StreamHandler) Start() {
	sh.wg.Add(1)
	go sh.heartbeatLoop()
}

// Stop gracefully stops the stream handler
func (sh *StreamHandler) Stop() {
	close(sh.stopCh)
	sh.wg.Wait()

	// Close all client connections
	sh.clientsMux.Lock()
	defer sh.clientsMux.Unlock()

	for _, client := range sh.clients {
		close(client.Done)
	}
	sh.clients = make(map[string]*ClientConnection)
}

// HandleSSE handles incoming SSE connection requests
func (sh *StreamHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if the response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		sh.logger.Error("SSE not supported: ResponseWriter does not support flushing")
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Generate unique client ID
	clientID := sh.generateClientID()

	// Create client connection
	client := &ClientConnection{
		ID:       clientID,
		Writer:   w,
		Flusher:  flusher,
		Done:     make(chan struct{}),
		LastSeen: time.Now(),
	}

	// Register client
	sh.addClient(client)
	defer sh.removeClient(clientID)

	sh.logger.Info("SSE client connected", "clientID", clientID, "remoteAddr", r.RemoteAddr)

	// Send initial connection event
	sh.sendEvent(client, "connected", map[string]interface{}{
		"clientId": clientID,
		"message":  "Connected to MCP stream",
	})

	// Keep connection alive until client disconnects or context is cancelled
	select {
	case <-r.Context().Done():
		sh.logger.Info("SSE client disconnected (context cancelled)", "clientID", clientID)
	case <-client.Done:
		sh.logger.Info("SSE client disconnected (connection closed)", "clientID", clientID)
	}
}

// GetStreamer returns the MCPStreamer instance
func (sh *StreamHandler) GetStreamer() *MCPStreamer {
	return sh.streamer
}

// BroadcastMessage sends a message to all connected clients
func (sh *StreamHandler) BroadcastMessage(eventType string, data interface{}) {
	sh.clientsMux.RLock()
	clients := make([]*ClientConnection, 0, len(sh.clients))
	for _, client := range sh.clients {
		clients = append(clients, client)
	}
	sh.clientsMux.RUnlock()

	for _, client := range clients {
		sh.sendEvent(client, eventType, data)
	}
}

// SendToClient sends a message to a specific client
func (sh *StreamHandler) SendToClient(clientID, eventType string, data interface{}) {
	sh.clientsMux.RLock()
	client, exists := sh.clients[clientID]
	sh.clientsMux.RUnlock()

	if !exists {
		sh.logger.Warn("Attempted to send message to non-existent client", "clientID", clientID)
		return
	}

	sh.sendEvent(client, eventType, data)
}

// GetConnectedClients returns the number of connected clients
func (sh *StreamHandler) GetConnectedClients() int {
	sh.clientsMux.RLock()
	defer sh.clientsMux.RUnlock()
	return len(sh.clients)
}

// addClient adds a new client connection
func (sh *StreamHandler) addClient(client *ClientConnection) {
	sh.clientsMux.Lock()
	defer sh.clientsMux.Unlock()
	sh.clients[client.ID] = client
}

// removeClient removes a client connection
func (sh *StreamHandler) removeClient(clientID string) {
	sh.clientsMux.Lock()
	defer sh.clientsMux.Unlock()
	delete(sh.clients, clientID)
}

// sendEvent sends an SSE event to a specific client
func (sh *StreamHandler) sendEvent(client *ClientConnection, eventType string, data interface{}) {
	// Check if client connection is still active
	select {
	case <-client.Done:
		return // Client connection is closed
	default:
	}

	// Format SSE event
	event := formatSSEEvent(eventType, data)

	// Write event to client
	if _, err := fmt.Fprint(client.Writer, event); err != nil {
		sh.logger.Error("Failed to write SSE event to client", "clientID", client.ID, "error", err)
		close(client.Done)
		return
	}

	// Flush the response
	client.Flusher.Flush()

	// Update last seen time
	client.LastSeen = time.Now()
}

// heartbeatLoop sends periodic heartbeat messages to keep connections alive
func (sh *StreamHandler) heartbeatLoop() {
	defer sh.wg.Done()

	ticker := time.NewTicker(sh.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-sh.stopCh:
			return
		case <-ticker.C:
			sh.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends heartbeat to all connected clients
func (sh *StreamHandler) sendHeartbeat() {
	sh.clientsMux.RLock()
	clients := make([]*ClientConnection, 0, len(sh.clients))
	for _, client := range sh.clients {
		clients = append(clients, client)
	}
	sh.clientsMux.RUnlock()

	for _, client := range clients {
		// Send heartbeat event
		sh.sendEvent(client, "heartbeat", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}

	sh.logger.Debug("Sent heartbeat to clients", "count", len(clients))
}

// generateClientID generates a unique client ID
func (sh *StreamHandler) generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// formatSSEEvent formats data as an SSE event
func formatSSEEvent(eventType string, data interface{}) string {
	// Convert data to JSON string
	jsonData, err := json.Marshal(data)
	if err != nil {
		jsonData = []byte(fmt.Sprintf(`{"error": "failed to marshal data: %v"}`, err))
	}

	// Format as SSE event
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
}
