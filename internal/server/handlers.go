package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/nicholasflintwillow/github-mcp/internal/errors"
	"github.com/nicholasflintwillow/github-mcp/internal/mcp"
)

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, errors.Validation("method not allowed"))
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "github-mcp-server",
		"version": "1.0.0",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, errors.Validation("method not allowed"))
		return
	}

	// Check if the server is ready to serve requests
	checks := map[string]string{
		"server": "ok",
		"config": "ok",
	}

	// Check GitHub API connectivity
	ctx := r.Context()
	if err := s.githubClient.ValidateToken(ctx); err != nil {
		checks["github"] = "error"
		s.logger.Warn("GitHub API connectivity check failed", "error", err)
	} else {
		checks["github"] = "ok"
	}

	status := "ready"
	statusCode := http.StatusOK

	// If any check failed, mark as not ready
	for _, check := range checks {
		if check != "ok" {
			status = "not_ready"
			statusCode = http.StatusServiceUnavailable
			break
		}
	}

	response := map[string]interface{}{
		"status":  status,
		"service": "github-mcp-server",
		"checks":  checks,
	}

	s.writeJSONResponse(w, statusCode, response)
}

// handleMCP handles MCP protocol requests
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, errors.Validation("only POST method is allowed for MCP requests"))
		return
	}

	s.logger.Info("MCP request received", "method", r.Method, "path", r.URL.Path)

	// Read request body
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil && err.Error() != "EOF" {
		s.logger.Error("Failed to read MCP request body", "error", err)
		s.writeErrorResponse(w, errors.Validation("failed to read request body"))
		return
	}

	// Process MCP message
	responseData, err := s.mcpHandler.HandleMessage(r.Context(), body)
	if err != nil {
		s.logger.Error("Failed to process MCP message", "error", err)
		s.writeErrorResponse(w, errors.Internal("failed to process MCP message"))
		return
	}

	// If no response (notification), return 204 No Content
	if responseData == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Write MCP response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(responseData); err != nil {
		s.logger.Error("Failed to write MCP response", "error", err)
	}
}

// handleMCPRequest handles MCP protocol requests (new dedicated endpoint)
func (s *Server) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, errors.Validation("only POST method is allowed for MCP requests"))
		return
	}

	s.logger.Info("MCP request received", "method", r.Method, "path", r.URL.Path)

	s.logger.Info("MCP request received", "method", r.Method, "path", r.URL.Path)

	var body []byte
	var msg *mcp.JSONRPCMessage
	var err error

	if r.Method == http.MethodPost {
		body = make([]byte, r.ContentLength)
		if _, err := r.Body.Read(body); err != nil && err != io.EOF {
			s.logger.Error("Failed to read MCP request body", "error", err)
			s.writeErrorResponse(w, errors.Validation("failed to read request body"))
			return
		}
		msg, err = mcp.FromJSON(body)
		if err != nil {
			s.logger.Error("Failed to parse MCP message from POST body", "error", err)
			s.writeErrorResponse(w, errors.Validation("failed to parse MCP message"))
			return
		}
	} else if r.Method == http.MethodGet {
		// If it's a GET request, and no body is expected,
		// we assume it's an implicit tools/list or resources/list request from Roo's client.
		// This is a workaround for Roo's current behavior.
		s.logger.Warn("Received GET request for MCP endpoint; assuming tools/list or resources/list", "path", r.URL.Path)
		// Create a dummy JSON-RPC message for tools/list
		msg = mcp.NewRequest(nil, mcp.MethodListTools, nil)
		// Marshal this dummy message to bytes for HandleMessage
		body, err = json.Marshal(msg)
		if err != nil {
			s.logger.Error("Failed to marshal dummy MCP message", "error", err)
			s.writeErrorResponse(w, errors.Internal("failed to create internal MCP message"))
			return
		}
	} else {
		s.writeErrorResponse(w, errors.Validation("only GET or POST methods are allowed for MCP requests"))
		return
	}

	// Process MCP message
	responseData, err := s.mcpHandler.HandleMessage(r.Context(), body)
	if err != nil {
		s.logger.Error("Failed to process MCP message", "error", err)
		s.writeErrorResponse(w, errors.Internal("failed to process MCP message"))
		return
	}

	// If no response (notification), return 204 No Content
	if responseData == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Write MCP response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(responseData); err != nil {
		s.logger.Error("Failed to write MCP response", "error", err)
	}
}

// handleMCPStream handles SSE connections for streaming MCP messages
func (s *Server) handleMCPStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, errors.Validation("only GET method is allowed for MCP stream"))
		return
	}

	s.logger.Info("MCP stream connection requested", "remoteAddr", r.RemoteAddr)

	// Delegate to the stream handler
	s.streamHandler.HandleSSE(w, r)
}

// handleNotFound handles requests to undefined routes
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	s.logger.Warn("Route not found", "method", r.Method, "path", r.URL.Path)

	err := errors.NotFound("route not found").WithContext("path", r.URL.Path)
	s.writeErrorResponse(w, err)
}

// writeJSONResponse writes a JSON response (improved version)
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
		// Fallback to simple error response
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response (improved version)
func (s *Server) writeErrorResponse(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)

	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"type":    err.Type,
			"message": err.Message,
		},
	}

	if err.Context != nil {
		response["error"].(map[string]interface{})["context"] = err.Context
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		s.logger.Error("Failed to encode error response", "error", encodeErr)
		// Fallback to simple error response
		http.Error(w, err.Message, err.StatusCode)
	}
}
