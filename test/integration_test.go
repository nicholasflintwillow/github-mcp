package test

import (
	"encoding/json"
	"testing"

	"github.com/nicholasflintwillow/github-mcp/internal/config"
	"github.com/nicholasflintwillow/github-mcp/internal/logger"
	"github.com/nicholasflintwillow/github-mcp/internal/mcp"
)

func TestMCPProtocolBasics(t *testing.T) {
	// Test JSON-RPC message creation and parsing
	t.Run("JSONRPCMessage", func(t *testing.T) {
		// Test request creation
		req := mcp.NewRequest(1, "initialize", map[string]interface{}{
			"protocolVersion": "2024-11-05",
		})

		if req.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC version 2.0, got %s", req.JSONRPC)
		}

		if req.ID != 1 {
			t.Errorf("Expected ID 1, got %v", req.ID)
		}

		if req.Method != "initialize" {
			t.Errorf("Expected method 'initialize', got %s", req.Method)
		}

		// Test JSON serialization
		data, err := req.ToJSON()
		if err != nil {
			t.Fatalf("Failed to serialize to JSON: %v", err)
		}

		// Test JSON deserialization
		parsed, err := mcp.FromJSON(data)
		if err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed.Method != req.Method {
			t.Errorf("Parsed method doesn't match: expected %s, got %s", req.Method, parsed.Method)
		}
	})

	t.Run("InitializeRequest", func(t *testing.T) {
		initReq := mcp.InitializeRequest{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcp.ClientCapabilities{
				Experimental: map[string]interface{}{},
			},
			ClientInfo: mcp.ClientInfo{
				Name:    "test-client",
				Version: "1.0.0",
			},
		}

		// Test JSON marshaling
		data, err := json.Marshal(initReq)
		if err != nil {
			t.Fatalf("Failed to marshal InitializeRequest: %v", err)
		}

		// Test JSON unmarshaling
		var parsed mcp.InitializeRequest
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal InitializeRequest: %v", err)
		}

		if parsed.ClientInfo.Name != initReq.ClientInfo.Name {
			t.Errorf("Client name doesn't match: expected %s, got %s", initReq.ClientInfo.Name, parsed.ClientInfo.Name)
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := &config.Config{
			Port:                  8080,
			Host:                  "localhost",
			GitHubToken:           "test_token",
			LogLevel:              "INFO",
			LogFormat:             "json",
			CacheTTL:              60,
			MaxConcurrentRequests: 100,
		}

		if err := cfg.Validate(); err != nil {
			t.Errorf("Valid config should not return error: %v", err)
		}
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		cfg := &config.Config{
			Port:        0, // Invalid port
			Host:        "localhost",
			GitHubToken: "", // Missing token
			LogLevel:    "INVALID",
			LogFormat:   "json",
		}

		if err := cfg.Validate(); err == nil {
			t.Error("Invalid config should return error")
		}
	})
}

func TestLoggerInitialization(t *testing.T) {
	t.Run("JSONLogger", func(t *testing.T) {
		logger, err := logger.New("INFO", "json")
		if err != nil {
			t.Fatalf("Failed to create JSON logger: %v", err)
		}

		if logger == nil {
			t.Error("Logger should not be nil")
		}
	})

	t.Run("TextLogger", func(t *testing.T) {
		logger, err := logger.New("DEBUG", "text")
		if err != nil {
			t.Fatalf("Failed to create text logger: %v", err)
		}

		if logger == nil {
			t.Error("Logger should not be nil")
		}
	})
}

func TestMCPHandlerBasics(t *testing.T) {
	// Test MCP handler creation (without GitHub client for now)
	t.Run("HandlerCreation", func(t *testing.T) {
		// This test would require a GitHub client, so we'll skip the actual handler creation
		// and just test that the MCP protocol structures work correctly

		// Test tool definition
		tool := mcp.Tool{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}

		if tool.Name != "test_tool" {
			t.Errorf("Tool name doesn't match: expected test_tool, got %s", tool.Name)
		}

		// Test resource definition
		resource := mcp.Resource{
			URI:         "test://resource",
			Name:        "Test Resource",
			Description: "A test resource",
			MimeType:    "application/json",
		}

		if resource.URI != "test://resource" {
			t.Errorf("Resource URI doesn't match: expected test://resource, got %s", resource.URI)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("MCPErrors", func(t *testing.T) {
		// Test MCP error response creation
		errorResp := mcp.NewErrorResponse(1, mcp.ErrorCodeMethodNotFound, "Method not found", nil)

		if errorResp.Error == nil {
			t.Error("Error response should have error field")
		}

		if errorResp.Error.Code != mcp.ErrorCodeMethodNotFound {
			t.Errorf("Error code doesn't match: expected %d, got %d", mcp.ErrorCodeMethodNotFound, errorResp.Error.Code)
		}

		if errorResp.Error.Message != "Method not found" {
			t.Errorf("Error message doesn't match: expected 'Method not found', got %s", errorResp.Error.Message)
		}
	})
}

// Benchmark tests
func BenchmarkJSONRPCMessageCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req := mcp.NewRequest(i, "test_method", map[string]interface{}{
			"param1": "value1",
			"param2": i,
		})
		_, _ = req.ToJSON()
	}
}

func BenchmarkJSONRPCMessageParsing(b *testing.B) {
	req := mcp.NewRequest(1, "test_method", map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	})
	data, _ := req.ToJSON()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mcp.FromJSON(data)
	}
}
