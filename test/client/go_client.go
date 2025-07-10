package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/mcp"
)

const (
	defaultServerURL = "http://localhost:8080"
	streamEndpoint   = "/mcp/stream"
	requestEndpoint  = "/mcp/request"
)

type Client struct {
	serverURL  string
	httpClient *http.Client
}

func NewClient(serverURL string) *Client {
	if serverURL == "" {
		serverURL = defaultServerURL
	}

	return &Client{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// connectSSE establishes an SSE connection to the stream endpoint
func (c *Client) connectSSE(ctx context.Context) error {
	url := c.serverURL + streamEndpoint
	fmt.Printf("Connecting to SSE stream at: %s\n", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSE connection failed with status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ SSE connection established successfully")
	fmt.Println("📡 Listening for streamed messages...")

	// Read SSE events
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Handle SSE event format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "" {
				fmt.Printf("📨 Received SSE message: %s\n", data)

				// Try to parse as MCP message
				var msg mcp.JSONRPCMessage
				if err := json.Unmarshal([]byte(data), &msg); err == nil {
					fmt.Printf("   📋 Parsed MCP message - Method: %s, ID: %v\n", msg.Method, msg.ID)
				}
			}
		} else if strings.HasPrefix(line, "event: ") {
			event := strings.TrimPrefix(line, "event: ")
			fmt.Printf("📡 Event type: %s\n", event)
		} else if line == "" {
			// Empty line indicates end of event
			fmt.Println("---")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading SSE stream: %w", err)
	}

	return nil
}

// sendRequest sends an HTTP POST request to the request endpoint
func (c *Client) sendRequest(message *mcp.JSONRPCMessage) (*mcp.JSONRPCMessage, error) {
	url := c.serverURL + requestEndpoint

	// Marshal the message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	fmt.Printf("📤 Sending HTTP POST request to: %s\n", url)
	fmt.Printf("   📋 Message: %s\n", string(data))

	// Create and send the request
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("📥 Received HTTP response (status %d): %s\n", resp.StatusCode, string(respBody))

	// Handle different response types
	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("   ✅ Request processed successfully (no response expected)")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse MCP response
	var response mcp.JSONRPCMessage
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse MCP response: %w", err)
	}

	fmt.Printf("   ✅ Parsed MCP response - ID: %v, Error: %v\n", response.ID, response.Error != nil)
	return &response, nil
}

// createSampleMessages creates sample MCP messages for testing
func createSampleMessages() []*mcp.JSONRPCMessage {
	return []*mcp.JSONRPCMessage{
		// Initialize request
		mcp.NewRequest(1, mcp.MethodInitialize, mcp.InitializeRequest{
			ProtocolVersion: mcp.MCPVersion,
			Capabilities: mcp.ClientCapabilities{
				Experimental: map[string]interface{}{},
			},
			ClientInfo: mcp.ClientInfo{
				Name:    "test-client",
				Version: "1.0.0",
			},
		}),

		// List tools request
		mcp.NewRequest(2, mcp.MethodListTools, nil),

		// List resources request
		mcp.NewRequest(3, mcp.MethodListResources, nil),

		// Ping request
		mcp.NewRequest(4, mcp.MethodPing, nil),

		// Initialized notification
		mcp.NewNotification(mcp.MethodInitialized, nil),
	}
}

func main() {
	// Parse command line arguments
	serverURL := defaultServerURL
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	client := NewClient(serverURL)

	fmt.Println("🚀 MCP Streamable HTTP Transport Test Client")
	fmt.Printf("🌐 Server URL: %s\n", serverURL)
	fmt.Println(strings.Repeat("=", 50))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Test HTTP POST requests first
	fmt.Println("\n🧪 Testing HTTP POST requests...")
	messages := createSampleMessages()

	for i, msg := range messages {
		fmt.Printf("\n--- Test %d ---\n", i+1)

		response, err := client.sendRequest(msg)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else if response != nil {
			fmt.Printf("✅ Success: Received response\n")
		} else {
			fmt.Printf("✅ Success: No response expected\n")
		}

		// Small delay between requests
		time.Sleep(500 * time.Millisecond)
	}

	// Test SSE connection
	fmt.Println("\n🧪 Testing SSE connection...")
	fmt.Println("Press Ctrl+C to stop...")

	go func() {
		if err := client.connectSSE(ctx); err != nil {
			fmt.Printf("❌ SSE Error: %v\n", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\n🛑 Shutting down client...")
	cancel()

	// Give some time for cleanup
	time.Sleep(1 * time.Second)
	fmt.Println("👋 Client stopped")
}
