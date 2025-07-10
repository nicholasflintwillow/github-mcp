package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nicholasflintwillow/github-mcp/internal/client"
	"github.com/nicholasflintwillow/github-mcp/internal/errors"
	"github.com/nicholasflintwillow/github-mcp/internal/logger"
)

// Handler handles MCP protocol requests
type Handler struct {
	githubClient *client.GitHubClient
	logger       *logger.Logger
	initialized  bool
	tools        []Tool
	resources    []Resource
	streamer     *MCPStreamer
}

// NewHandler creates a new MCP handler
func NewHandler(githubClient *client.GitHubClient, logger *logger.Logger) *Handler {
	h := &Handler{
		githubClient: githubClient,
		logger:       logger,
		initialized:  false,
	}

	// Initialize tools and resources
	h.initializeTools()
	h.initializeResources()

	return h
}

// SetStreamer sets the MCP streamer for this handler
func (h *Handler) SetStreamer(streamer *MCPStreamer) {
	h.streamer = streamer
}

// HandleMessage processes an MCP message
func (h *Handler) HandleMessage(ctx context.Context, data []byte) ([]byte, error) {
	// Parse the JSON-RPC message
	msg, err := FromJSON(data)
	if err != nil {
		h.logger.Error("Failed to parse MCP message", "error", err)
		errorResp := NewErrorResponse(nil, ErrorCodeParseError, "Parse error", nil)
		return errorResp.ToJSON()
	}

	h.logger.Debug("Received MCP message", "method", msg.Method, "id", msg.ID)

	// Handle the message based on type
	if msg.IsRequest() {
		return h.handleRequest(ctx, msg)
	} else if msg.IsNotification() {
		return h.handleNotification(ctx, msg)
	} else {
		h.logger.Warn("Received unexpected message type", "message", string(data))
		errorResp := NewErrorResponse(msg.ID, ErrorCodeInvalidRequest, "Invalid request", nil)
		return errorResp.ToJSON()
	}
}

// handleRequest handles JSON-RPC requests
func (h *Handler) handleRequest(ctx context.Context, msg *JSONRPCMessage) ([]byte, error) {
	var response *JSONRPCMessage

	switch msg.Method {
	case MethodInitialize:
		response = h.handleInitialize(msg)
	case MethodListTools:
		response = h.handleListTools(msg)
	case MethodCallTool:
		response = h.handleCallTool(ctx, msg)
	case MethodListResources:
		response = h.handleListResources(msg)
	case MethodReadResource:
		response = h.handleReadResource(ctx, msg)
	case MethodListResourceTemplates:
		response = h.handleListResourceTemplates(msg)
	case MethodPing:
		response = h.handlePing(msg)
	default:
		response = NewErrorResponse(msg.ID, ErrorCodeMethodNotFound, fmt.Sprintf("Method not found: %s", msg.Method), nil)
	}

	return response.ToJSON()
}

// handleNotification handles JSON-RPC notifications
func (h *Handler) handleNotification(ctx context.Context, msg *JSONRPCMessage) ([]byte, error) {
	switch msg.Method {
	case MethodInitialized:
		h.handleInitialized(msg)
	default:
		h.logger.Warn("Unknown notification method", "method", msg.Method)
	}

	// Notifications don't require a response
	return nil, nil
}

// handleInitialize handles the initialize request
func (h *Handler) handleInitialize(msg *JSONRPCMessage) *JSONRPCMessage {
	var req InitializeRequest
	if err := msg.GetParams(&req); err != nil {
		h.logger.Error("Failed to parse initialize request", "error", err)
		return NewErrorResponse(msg.ID, ErrorCodeInvalidParams, "Invalid params", nil)
	}

	h.logger.Info("Initializing MCP server", "client", req.ClientInfo.Name, "version", req.ClientInfo.Version)

	// Create initialize result
	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "github-mcp-server",
			Version: "1.0.0",
		},
		Instructions: "GitHub MCP Server - Provides access to GitHub API through MCP protocol",
	}

	return NewResponse(msg.ID, result)
}

// handleInitialized handles the initialized notification
func (h *Handler) handleInitialized(msg *JSONRPCMessage) {
	h.initialized = true
	h.logger.Info("MCP server initialized successfully")
}

// handleListTools handles the tools/list request
func (h *Handler) handleListTools(msg *JSONRPCMessage) *JSONRPCMessage {
	if !h.initialized {
		return NewErrorResponse(msg.ID, ErrorCodeInternalError, "Server not initialized", nil)
	}

	result := ToolsListResult{
		Tools: h.tools,
	}

	return NewResponse(msg.ID, result)
}

// handleCallTool handles the tools/call request
func (h *Handler) handleCallTool(ctx context.Context, msg *JSONRPCMessage) *JSONRPCMessage {
	if !h.initialized {
		return NewErrorResponse(msg.ID, ErrorCodeInternalError, "Server not initialized", nil)
	}

	var req CallToolRequest
	if err := msg.GetParams(&req); err != nil {
		h.logger.Error("Failed to parse call tool request", "error", err)
		return NewErrorResponse(msg.ID, ErrorCodeInvalidParams, "Invalid params", nil)
	}

	h.logger.Info("Calling tool", "name", req.Name)

	// Stream tool execution start notification if streaming is enabled
	if h.streamer != nil && h.streamer.IsStreamingEnabled() {
		h.streamer.StreamToolProgress(req.Name, map[string]interface{}{
			"status": "started",
			"toolId": msg.ID,
		})
	}

	// Find the tool
	var tool *Tool
	for _, t := range h.tools {
		if t.Name == req.Name {
			tool = &t
			break
		}
	}

	if tool == nil {
		errorResp := NewErrorResponse(msg.ID, ErrorCodeToolNotFound, fmt.Sprintf("Tool not found: %s", req.Name), nil)
		// Stream error if streaming is enabled
		if h.streamer != nil && h.streamer.IsStreamingEnabled() {
			h.streamer.StreamMessage(errorResp)
		}
		return errorResp
	}

	// Execute the tool
	result, err := h.executeTool(ctx, req.Name, req.Arguments)
	if err != nil {
		h.logger.Error("Tool execution failed", "tool", req.Name, "error", err)
		errorResp := NewErrorResponse(msg.ID, ErrorCodeInvalidTool, fmt.Sprintf("Tool execution failed: %v", err), nil)
		// Stream error if streaming is enabled
		if h.streamer != nil && h.streamer.IsStreamingEnabled() {
			h.streamer.StreamMessage(errorResp)
		}
		return errorResp
	}

	// Stream tool execution completion notification if streaming is enabled
	if h.streamer != nil && h.streamer.IsStreamingEnabled() {
		h.streamer.StreamToolProgress(req.Name, map[string]interface{}{
			"status": "completed",
			"toolId": msg.ID,
		})
	}

	response := NewResponse(msg.ID, result)

	// Stream successful response if streaming is enabled
	if h.streamer != nil && h.streamer.IsStreamingEnabled() {
		h.streamer.StreamMessage(response)
	}

	return response
}

// handleListResources handles the resources/list request
func (h *Handler) handleListResources(msg *JSONRPCMessage) *JSONRPCMessage {
	if !h.initialized {
		return NewErrorResponse(msg.ID, ErrorCodeInternalError, "Server not initialized", nil)
	}

	result := ResourcesListResult{
		Resources: h.resources,
	}

	return NewResponse(msg.ID, result)
}

// handleReadResource handles the resources/read request
func (h *Handler) handleReadResource(ctx context.Context, msg *JSONRPCMessage) *JSONRPCMessage {
	if !h.initialized {
		return NewErrorResponse(msg.ID, ErrorCodeInternalError, "Server not initialized", nil)
	}

	var req ReadResourceRequest
	if err := msg.GetParams(&req); err != nil {
		h.logger.Error("Failed to parse read resource request", "error", err)
		return NewErrorResponse(msg.ID, ErrorCodeInvalidParams, "Invalid params", nil)
	}

	h.logger.Info("Reading resource", "uri", req.URI)

	// Execute the resource read
	result, err := h.readResource(ctx, req.URI)
	if err != nil {
		h.logger.Error("Resource read failed", "uri", req.URI, "error", err)
		return NewErrorResponse(msg.ID, ErrorCodeResourceNotFound, fmt.Sprintf("Resource read failed: %v", err), nil)
	}

	return NewResponse(msg.ID, result)
}

// handleListResourceTemplates handles the resources/templates/list request
func (h *Handler) handleListResourceTemplates(msg *JSONRPCMessage) *JSONRPCMessage {
	if !h.initialized {
		return NewErrorResponse(msg.ID, ErrorCodeInternalError, "Server not initialized", nil)
	}

	// For now, return empty list - will be implemented in later tasks
	result := ResourceTemplatesListResult{
		ResourceTemplates: []ResourceTemplate{},
	}

	return NewResponse(msg.ID, result)
}

// handlePing handles the ping request
func (h *Handler) handlePing(msg *JSONRPCMessage) *JSONRPCMessage {
	return NewResponse(msg.ID, map[string]string{"status": "pong"})
}

// initializeTools initializes the available tools
func (h *Handler) initializeTools() {
	// GitHub Users API tools
	h.tools = []Tool{
		{
			Name:        "get_user",
			Description: "Get information about a GitHub user by username",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "get_authenticated_user",
			Description: "Get information about the authenticated user",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "update_authenticated_user",
			Description: "Update the authenticated user's profile",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The new name of the user",
					},
					"email": map[string]interface{}{
						"type":        "string",
						"description": "The publicly visible email address of the user",
					},
					"blog": map[string]interface{}{
						"type":        "string",
						"description": "The new blog URL of the user",
					},
					"company": map[string]interface{}{
						"type":        "string",
						"description": "The new company of the user",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The new location of the user",
					},
					"hireable": map[string]interface{}{
						"type":        "boolean",
						"description": "The new hiring availability of the user",
					},
					"bio": map[string]interface{}{
						"type":        "string",
						"description": "The new short biography of the user",
					},
					"twitter_username": map[string]interface{}{
						"type":        "string",
						"description": "The new Twitter username of the user",
					},
				},
			},
		},
		{
			Name:        "list_users",
			Description: "List all GitHub users",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"since": map[string]interface{}{
						"type":        "integer",
						"description": "A user ID. Only return users with an ID greater than this ID",
						"minimum":     0,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
			},
		},
		{
			Name:        "list_user_followers",
			Description: "List followers of a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "list_user_following",
			Description: "List users followed by a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "check_user_following",
			Description: "Check if the authenticated user follows another user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username to check",
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "follow_user",
			Description: "Follow a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username to follow",
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "unfollow_user",
			Description: "Unfollow a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username to unfollow",
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "list_repositories",
			Description: "List repositories for a user or organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Repository owner (username or organization)",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Repository type (all, owner, member)",
						"enum":        []string{"all", "owner", "member"},
						"default":     "owner",
					},
				},
				"required": []string{"owner"},
			},
		},
		// GitHub Organizations API tools
		{
			Name:        "get_organization",
			Description: "Get information about a GitHub organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
				},
				"required": []string{"org"},
			},
		},
		{
			Name:        "update_organization",
			Description: "Update an organization's profile",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The organization's display name",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The organization's description",
					},
					"company": map[string]interface{}{
						"type":        "string",
						"description": "The organization's company name",
					},
					"blog": map[string]interface{}{
						"type":        "string",
						"description": "The organization's blog URL",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The organization's location",
					},
					"email": map[string]interface{}{
						"type":        "string",
						"description": "The organization's email",
					},
					"twitter_username": map[string]interface{}{
						"type":        "string",
						"description": "The organization's Twitter username",
					},
					"billing_email": map[string]interface{}{
						"type":        "string",
						"description": "The organization's billing email",
					},
					"has_organization_projects": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether organization projects are enabled",
					},
					"has_repository_projects": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether repository projects are enabled",
					},
					"default_repository_permission": map[string]interface{}{
						"type":        "string",
						"description": "Default permission level members have for organization repositories",
						"enum":        []string{"read", "write", "admin", "none"},
					},
					"members_can_create_repositories": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether members can create repositories",
					},
				},
				"required": []string{"org"},
			},
		},
		{
			Name:        "list_organizations",
			Description: "List all GitHub organizations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"since": map[string]interface{}{
						"type":        "integer",
						"description": "An organization ID. Only return organizations with an ID greater than this ID",
						"minimum":     0,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
			},
		},
		{
			Name:        "list_user_organizations",
			Description: "List organizations for a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"username"},
			},
		},
		{
			Name:        "list_authenticated_user_organizations",
			Description: "List organizations for the authenticated user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
			},
		},
		{
			Name:        "list_organization_members",
			Description: "List members of an organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Filter members returned in the list",
						"enum":        []string{"2fa_disabled", "all"},
						"default":     "all",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "Filter members returned by their role",
						"enum":        []string{"all", "admin", "member"},
						"default":     "all",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"org"},
			},
		},
		{
			Name:        "check_organization_membership",
			Description: "Check if a user is a member of an organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username to check",
					},
				},
				"required": []string{"org", "username"},
			},
		},
		{
			Name:        "check_public_organization_membership",
			Description: "Check if a user is a public member of an organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username to check",
					},
				},
				"required": []string{"org", "username"},
			},
		},
		// GitHub Teams API tools
		{
			Name:        "list_teams",
			Description: "List teams in an organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"org"},
			},
		},
		{
			Name:        "get_team",
			Description: "Get a team by organization and team slug",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
				},
				"required": []string{"org", "team_slug"},
			},
		},
		{
			Name:        "create_team",
			Description: "Create a new team in an organization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the team",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The description of the team",
					},
					"privacy": map[string]interface{}{
						"type":        "string",
						"description": "The level of privacy this team should have",
						"enum":        []string{"secret", "closed"},
						"default":     "secret",
					},
					"permission": map[string]interface{}{
						"type":        "string",
						"description": "The permission that new repositories will be added to the team with",
						"enum":        []string{"pull", "push", "admin"},
						"default":     "pull",
					},
					"parent_team_id": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of a team to set as the parent team",
					},
				},
				"required": []string{"org", "name"},
			},
		},
		{
			Name:        "update_team",
			Description: "Update a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the team",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The description of the team",
					},
					"privacy": map[string]interface{}{
						"type":        "string",
						"description": "The level of privacy this team should have",
						"enum":        []string{"secret", "closed"},
					},
					"permission": map[string]interface{}{
						"type":        "string",
						"description": "The permission that new repositories will be added to the team with",
						"enum":        []string{"pull", "push", "admin"},
					},
					"parent_team_id": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of a team to set as the parent team",
					},
				},
				"required": []string{"org", "team_slug"},
			},
		},
		{
			Name:        "delete_team",
			Description: "Delete a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
				},
				"required": []string{"org", "team_slug"},
			},
		},
		{
			Name:        "list_team_members",
			Description: "List members of a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "Filter members returned by their role",
						"enum":        []string{"member", "maintainer", "all"},
						"default":     "all",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"org", "team_slug"},
			},
		},
		{
			Name:        "get_team_membership",
			Description: "Get team membership for a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
				},
				"required": []string{"org", "team_slug", "username"},
			},
		},
		{
			Name:        "add_team_membership",
			Description: "Add or update team membership for a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "The role to give the user in the team",
						"enum":        []string{"member", "maintainer"},
						"default":     "member",
					},
				},
				"required": []string{"org", "team_slug", "username"},
			},
		},
		{
			Name:        "remove_team_membership",
			Description: "Remove a user from a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username",
					},
				},
				"required": []string{"org", "team_slug", "username"},
			},
		},
		{
			Name:        "list_team_repositories",
			Description: "List repositories for a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"description": "Page number of the results to fetch",
						"minimum":     1,
						"default":     1,
					},
					"per_page": map[string]interface{}{
						"type":        "integer",
						"description": "The number of results per page (max 100)",
						"minimum":     1,
						"maximum":     100,
						"default":     30,
					},
				},
				"required": []string{"org", "team_slug"},
			},
		},
		{
			Name:        "check_team_repository",
			Description: "Check if a team has access to a repository",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Repository owner",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "Repository name",
					},
				},
				"required": []string{"org", "team_slug", "owner", "repo"},
			},
		},
		{
			Name:        "add_team_repository",
			Description: "Add a repository to a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Repository owner",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "Repository name",
					},
					"permission": map[string]interface{}{
						"type":        "string",
						"description": "The permission to grant the team on this repository",
						"enum":        []string{"pull", "triage", "push", "maintain", "admin"},
						"default":     "pull",
					},
				},
				"required": []string{"org", "team_slug", "owner", "repo"},
			},
		},
		{
			Name:        "remove_team_repository",
			Description: "Remove a repository from a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"org": map[string]interface{}{
						"type":        "string",
						"description": "Organization name",
					},
					"team_slug": map[string]interface{}{
						"type":        "string",
						"description": "Team slug",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Repository owner",
					},
					"repo": map[string]interface{}{
						"type":        "string",
						"description": "Repository name",
					},
				},
				"required": []string{"org", "team_slug", "owner", "repo"},
			},
		},
	}
}

// initializeResources initializes the available resources
func (h *Handler) initializeResources() {
	// Basic resources - will be expanded in later tasks
	h.resources = []Resource{
		{
			URI:         "github://user/{username}",
			Name:        "GitHub User",
			Description: "GitHub user information",
			MimeType:    "application/json",
		},
		{
			URI:         "github://repos/{owner}",
			Name:        "GitHub Repositories",
			Description: "List of GitHub repositories",
			MimeType:    "application/json",
		},
		// Organization resources
		{
			URI:         "github://org/{org}",
			Name:        "GitHub Organization",
			Description: "GitHub organization information",
			MimeType:    "application/json",
		},
		{
			URI:         "github://org/{org}/members",
			Name:        "GitHub Organization Members",
			Description: "List of GitHub organization members",
			MimeType:    "application/json",
		},
		{
			URI:         "github://organizations",
			Name:        "GitHub Organizations",
			Description: "List of all GitHub organizations",
			MimeType:    "application/json",
		},
		{
			URI:         "github://user/{username}/orgs",
			Name:        "User Organizations",
			Description: "List of organizations for a specific user",
			MimeType:    "application/json",
		},
	}
}

// executeTool executes a tool with the given arguments
func (h *Handler) executeTool(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	switch toolName {
	case "get_user":
		return h.executeGetUser(ctx, args)
	case "get_authenticated_user":
		return h.executeGetAuthenticatedUser(ctx, args)
	case "update_authenticated_user":
		return h.executeUpdateAuthenticatedUser(ctx, args)
	case "list_users":
		return h.executeListUsers(ctx, args)
	case "list_user_followers":
		return h.executeListUserFollowers(ctx, args)
	case "list_user_following":
		return h.executeListUserFollowing(ctx, args)
	case "check_user_following":
		return h.executeCheckUserFollowing(ctx, args)
	case "follow_user":
		return h.executeFollowUser(ctx, args)
	case "unfollow_user":
		return h.executeUnfollowUser(ctx, args)
	case "list_repositories":
		return h.executeListRepositories(ctx, args)
	// Organization tools
	case "get_organization":
		return h.executeGetOrganization(ctx, args)
	case "update_organization":
		return h.executeUpdateOrganization(ctx, args)
	case "list_organizations":
		return h.executeListOrganizations(ctx, args)
	case "list_user_organizations":
		return h.executeListUserOrganizations(ctx, args)
	case "list_authenticated_user_organizations":
		return h.executeListAuthenticatedUserOrganizations(ctx, args)
	case "list_organization_members":
		return h.executeListOrganizationMembers(ctx, args)
	case "check_organization_membership":
		return h.executeCheckOrganizationMembership(ctx, args)
	case "check_public_organization_membership":
		return h.executeCheckPublicOrganizationMembership(ctx, args)
	// Team tools
	case "list_teams":
		return h.executeListTeams(ctx, args)
	case "get_team":
		return h.executeGetTeam(ctx, args)
	case "create_team":
		return h.executeCreateTeam(ctx, args)
	case "update_team":
		return h.executeUpdateTeam(ctx, args)
	case "delete_team":
		return h.executeDeleteTeam(ctx, args)
	case "list_team_members":
		return h.executeListTeamMembers(ctx, args)
	case "get_team_membership":
		return h.executeGetTeamMembership(ctx, args)
	case "add_team_membership":
		return h.executeAddTeamMembership(ctx, args)
	case "remove_team_membership":
		return h.executeRemoveTeamMembership(ctx, args)
	case "list_team_repositories":
		return h.executeListTeamRepositories(ctx, args)
	case "check_team_repository":
		return h.executeCheckTeamRepository(ctx, args)
	case "add_team_repository":
		return h.executeAddTeamRepository(ctx, args)
	case "remove_team_repository":
		return h.executeRemoveTeamRepository(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// executeGetUser executes the get_user tool
func (h *Handler) executeGetUser(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the new client function
	user, err := h.githubClient.GetUser(ctx, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting user %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting user data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("User information for %s:\n%s", username, string(userJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListRepositories executes the list_repositories tool
func (h *Handler) executeListRepositories(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, errors.Validation("owner is required and must be a string")
	}

	repoType := "owner"
	if t, ok := args["type"].(string); ok {
		repoType = t
	}

	// Make GitHub API request
	endpoint := fmt.Sprintf("/users/%s/repos", owner)
	params := map[string]string{
		"type": repoType,
	}

	resp, err := h.githubClient.Get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	// Format response
	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Repositories for %s (type: %s):\n%s", owner, repoType, string(resp.Body)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeGetAuthenticatedUser executes the get_authenticated_user tool
func (h *Handler) executeGetAuthenticatedUser(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	// Make GitHub API request using the new client function
	user, err := h.githubClient.GetAuthenticatedUser(ctx)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting authenticated user: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting user data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Authenticated user information:\n%s", string(userJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeUpdateAuthenticatedUser executes the update_authenticated_user tool
func (h *Handler) executeUpdateAuthenticatedUser(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	// Build updates map from args
	updates := make(map[string]interface{})

	// Copy valid fields from args to updates
	validFields := []string{"name", "email", "blog", "company", "location", "hireable", "bio", "twitter_username"}
	for _, field := range validFields {
		if value, exists := args[field]; exists {
			updates[field] = value
		}
	}

	if len(updates) == 0 {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "No valid fields provided for update",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the new client function
	user, err := h.githubClient.UpdateAuthenticatedUser(ctx, updates)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error updating authenticated user: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting user data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Updated user information:\n%s", string(userJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListUsers executes the list_users tool
func (h *Handler) executeListUsers(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	var since int64
	var perPage int

	if s, ok := args["since"].(float64); ok {
		since = int64(s)
	}
	if p, ok := args["per_page"].(float64); ok {
		perPage = int(p)
	}

	// Make GitHub API request using the new client function
	users, err := h.githubClient.ListUsers(ctx, since, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing users: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	usersJSON, err := json.Marshal(users)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting users data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Users list (since: %d, per_page: %d):\n%s", since, perPage, string(usersJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListUserFollowers executes the list_user_followers tool
func (h *Handler) executeListUserFollowers(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the new client function
	followers, err := h.githubClient.ListUserFollowers(ctx, username, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing followers for %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	followersJSON, err := json.Marshal(followers)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting followers data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Followers for %s (page: %d, per_page: %d):\n%s", username, page, perPage, string(followersJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListUserFollowing executes the list_user_following tool
func (h *Handler) executeListUserFollowing(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the new client function
	following, err := h.githubClient.ListUserFollowing(ctx, username, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing following for %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	followingJSON, err := json.Marshal(following)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting following data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Following for %s (page: %d, per_page: %d):\n%s", username, page, perPage, string(followingJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeCheckUserFollowing executes the check_user_following tool
func (h *Handler) executeCheckUserFollowing(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the new client function
	isFollowing, err := h.githubClient.CheckUserFollowing(ctx, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error checking if following %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	status := "not following"
	if isFollowing {
		status = "following"
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Following status for %s: %s", username, status),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeFollowUser executes the follow_user tool
func (h *Handler) executeFollowUser(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the new client function
	err := h.githubClient.FollowUser(ctx, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error following %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully followed %s", username),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeUnfollowUser executes the unfollow_user tool
func (h *Handler) executeUnfollowUser(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the new client function
	err := h.githubClient.UnfollowUser(ctx, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error unfollowing %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully unfollowed %s", username),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// Organization tool execution functions

// executeGetOrganization executes the get_organization tool
func (h *Handler) executeGetOrganization(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	organization, err := h.githubClient.GetOrganization(ctx, org)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting organization %s: %v", org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	orgJSON, err := json.Marshal(organization)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting organization data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Organization information for %s:\n%s", org, string(orgJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeUpdateOrganization executes the update_organization tool
func (h *Handler) executeUpdateOrganization(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Build updates map from args
	updates := make(map[string]interface{})

	// Copy valid fields from args to updates
	validFields := []string{
		"name", "description", "company", "blog", "location", "email", "twitter_username",
		"billing_email", "has_organization_projects", "has_repository_projects",
		"default_repository_permission", "members_can_create_repositories",
	}
	for _, field := range validFields {
		if value, exists := args[field]; exists {
			updates[field] = value
		}
	}

	if len(updates) == 0 {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "No valid fields provided for update",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	organization, err := h.githubClient.UpdateOrganization(ctx, org, updates)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error updating organization %s: %v", org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	orgJSON, err := json.Marshal(organization)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting organization data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Updated organization information for %s:\n%s", org, string(orgJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListOrganizations executes the list_organizations tool
func (h *Handler) executeListOrganizations(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	var since int64
	var perPage int

	if s, ok := args["since"].(float64); ok {
		since = int64(s)
	}
	if p, ok := args["per_page"].(float64); ok {
		perPage = int(p)
	}

	// Make GitHub API request using the client function
	organizations, err := h.githubClient.ListOrganizations(ctx, since, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing organizations: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	orgsJSON, err := json.Marshal(organizations)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting organizations data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Organizations list (since: %d, per_page: %d):\n%s", since, perPage, string(orgsJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListUserOrganizations executes the list_user_organizations tool
func (h *Handler) executeListUserOrganizations(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	organizations, err := h.githubClient.ListUserOrganizations(ctx, username, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing organizations for %s: %v", username, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	orgsJSON, err := json.Marshal(organizations)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting organizations data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Organizations for %s (page: %d, per_page: %d):\n%s", username, page, perPage, string(orgsJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListAuthenticatedUserOrganizations executes the list_authenticated_user_organizations tool
func (h *Handler) executeListAuthenticatedUserOrganizations(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	organizations, err := h.githubClient.ListAuthenticatedUserOrganizations(ctx, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing authenticated user organizations: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	orgsJSON, err := json.Marshal(organizations)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting organizations data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Authenticated user organizations (page: %d, per_page: %d):\n%s", page, perPage, string(orgsJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListOrganizationMembers executes the list_organization_members tool
func (h *Handler) executeListOrganizationMembers(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var filter, role string
	var page, perPage int

	if f, ok := args["filter"].(string); ok {
		filter = f
	}
	if r, ok := args["role"].(string); ok {
		role = r
	}
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	members, err := h.githubClient.ListOrganizationMembers(ctx, org, filter, role, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing members for organization %s: %v", org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	membersJSON, err := json.Marshal(members)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting members data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Members for organization %s (filter: %s, role: %s, page: %d, per_page: %d):\n%s", org, filter, role, page, perPage, string(membersJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeCheckOrganizationMembership executes the check_organization_membership tool
func (h *Handler) executeCheckOrganizationMembership(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	isMember, err := h.githubClient.CheckOrganizationMembership(ctx, org, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error checking membership for %s in organization %s: %v", username, org, err),
			}},
			IsError: true,
		}, nil
	}

	status := "not a member"
	if isMember {
		status = "is a member"
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Membership status for %s in organization %s: %s", username, org, status),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeCheckPublicOrganizationMembership executes the check_public_organization_membership tool
func (h *Handler) executeCheckPublicOrganizationMembership(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	isPublicMember, err := h.githubClient.CheckPublicOrganizationMembership(ctx, org, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error checking public membership for %s in organization %s: %v", username, org, err),
			}},
			IsError: true,
		}, nil
	}

	status := "not a public member"
	if isPublicMember {
		status = "is a public member"
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Public membership status for %s in organization %s: %s", username, org, status),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// GitHub Teams API execution functions

// executeListTeams executes the list_teams tool
func (h *Handler) executeListTeams(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	teams, err := h.githubClient.ListTeams(ctx, org, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing teams for organization %s: %v", org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	teamsJSON, err := json.Marshal(teams)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting teams data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Teams for organization %s (page: %d, per_page: %d):\n%s", org, page, perPage, string(teamsJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeGetTeam executes the get_team tool
func (h *Handler) executeGetTeam(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	team, err := h.githubClient.GetTeam(ctx, org, teamSlug)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting team %s in organization %s: %v", teamSlug, org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	teamJSON, err := json.Marshal(team)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting team data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Team information for %s/%s:\n%s", org, teamSlug, string(teamJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeCreateTeam executes the create_team tool
func (h *Handler) executeCreateTeam(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	name, ok := args["name"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "name is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Build team data from args
	teamData := map[string]interface{}{
		"name": name,
	}

	// Add optional fields
	if description, ok := args["description"].(string); ok {
		teamData["description"] = description
	}
	if privacy, ok := args["privacy"].(string); ok {
		teamData["privacy"] = privacy
	}
	if permission, ok := args["permission"].(string); ok {
		teamData["permission"] = permission
	}
	if parentTeamID, ok := args["parent_team_id"].(float64); ok {
		teamData["parent_team_id"] = int(parentTeamID)
	}

	// Make GitHub API request using the client function
	team, err := h.githubClient.CreateTeam(ctx, org, teamData)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating team %s in organization %s: %v", name, org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	teamJSON, err := json.Marshal(team)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting team data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully created team %s in organization %s:\n%s", name, org, string(teamJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeUpdateTeam executes the update_team tool
func (h *Handler) executeUpdateTeam(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Build updates map from args
	updates := make(map[string]interface{})

	// Copy valid fields from args to updates
	validFields := []string{"name", "description", "privacy", "permission", "parent_team_id"}
	for _, field := range validFields {
		if value, exists := args[field]; exists {
			if field == "parent_team_id" {
				if parentTeamID, ok := value.(float64); ok {
					updates[field] = int(parentTeamID)
				}
			} else {
				updates[field] = value
			}
		}
	}

	if len(updates) == 0 {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "No valid fields provided for update",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	team, err := h.githubClient.UpdateTeam(ctx, org, teamSlug, updates)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error updating team %s in organization %s: %v", teamSlug, org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	teamJSON, err := json.Marshal(team)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting team data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully updated team %s in organization %s:\n%s", teamSlug, org, string(teamJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeDeleteTeam executes the delete_team tool
func (h *Handler) executeDeleteTeam(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	err := h.githubClient.DeleteTeam(ctx, org, teamSlug)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error deleting team %s in organization %s: %v", teamSlug, org, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted team %s in organization %s", teamSlug, org),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListTeamMembers executes the list_team_members tool
func (h *Handler) executeListTeamMembers(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var role string
	var page, perPage int
	if r, ok := args["role"].(string); ok {
		role = r
	}
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	members, err := h.githubClient.ListTeamMembers(ctx, org, teamSlug, role, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing members for team %s in organization %s: %v", teamSlug, org, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	membersJSON, err := json.Marshal(members)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting members data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Members for team %s/%s (role: %s, page: %d, per_page: %d):\n%s", org, teamSlug, role, page, perPage, string(membersJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeGetTeamMembership executes the get_team_membership tool
func (h *Handler) executeGetTeamMembership(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	membership, err := h.githubClient.GetTeamMembership(ctx, org, teamSlug, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting team membership for %s in team %s/%s: %v", username, org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	membershipJSON, err := json.Marshal(membership)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting membership data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Team membership for %s in team %s/%s:\n%s", username, org, teamSlug, string(membershipJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeAddTeamMembership executes the add_team_membership tool
func (h *Handler) executeAddTeamMembership(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var role string
	if r, ok := args["role"].(string); ok {
		role = r
	}

	// Make GitHub API request using the client function
	membership, err := h.githubClient.AddTeamMembership(ctx, org, teamSlug, username, role)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error adding %s to team %s/%s: %v", username, org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	membershipJSON, err := json.Marshal(membership)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting membership data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully added %s to team %s/%s:\n%s", username, org, teamSlug, string(membershipJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeRemoveTeamMembership executes the remove_team_membership tool
func (h *Handler) executeRemoveTeamMembership(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	username, ok := args["username"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "username is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	err := h.githubClient.RemoveTeamMembership(ctx, org, teamSlug, username)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error removing %s from team %s/%s: %v", username, org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully removed %s from team %s/%s", username, org, teamSlug),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeListTeamRepositories executes the list_team_repositories tool
func (h *Handler) executeListTeamRepositories(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var page, perPage int
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}
	if pp, ok := args["per_page"].(float64); ok {
		perPage = int(pp)
	}

	// Make GitHub API request using the client function
	repositories, err := h.githubClient.ListTeamRepositories(ctx, org, teamSlug, page, perPage)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing repositories for team %s/%s: %v", org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	// Format response as JSON
	repositoriesJSON, err := json.Marshal(repositories)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting repositories data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Repositories for team %s/%s (page: %d, per_page: %d):\n%s", org, teamSlug, page, perPage, string(repositoriesJSON)),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeCheckTeamRepository executes the check_team_repository tool
func (h *Handler) executeCheckTeamRepository(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	owner, ok := args["owner"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "owner is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	repo, ok := args["repo"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "repo is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	hasAccess, err := h.githubClient.CheckTeamRepository(ctx, org, teamSlug, owner, repo)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error checking team repository access for %s/%s to %s/%s: %v", org, teamSlug, owner, repo, err),
			}},
			IsError: true,
		}, nil
	}

	status := "no access"
	if hasAccess {
		status = "has access"
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Team %s/%s repository access to %s/%s: %s", org, teamSlug, owner, repo, status),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeAddTeamRepository executes the add_team_repository tool
func (h *Handler) executeAddTeamRepository(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	owner, ok := args["owner"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "owner is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	repo, ok := args["repo"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "repo is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	var permission string
	if p, ok := args["permission"].(string); ok {
		permission = p
	}

	// Make GitHub API request using the client function
	err := h.githubClient.AddTeamRepository(ctx, org, teamSlug, owner, repo, permission)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error adding repository %s/%s to team %s/%s: %v", owner, repo, org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully added repository %s/%s to team %s/%s with permission: %s", owner, repo, org, teamSlug, permission),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// executeRemoveTeamRepository executes the remove_team_repository tool
func (h *Handler) executeRemoveTeamRepository(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	org, ok := args["org"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "org is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	teamSlug, ok := args["team_slug"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "team_slug is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	owner, ok := args["owner"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "owner is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	repo, ok := args["repo"].(string)
	if !ok {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: "repo is required and must be a string",
			}},
			IsError: true,
		}, nil
	}

	// Make GitHub API request using the client function
	err := h.githubClient.RemoveTeamRepository(ctx, org, teamSlug, owner, repo)
	if err != nil {
		return &CallToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error removing repository %s/%s from team %s/%s: %v", owner, repo, org, teamSlug, err),
			}},
			IsError: true,
		}, nil
	}

	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Successfully removed repository %s/%s from team %s/%s", owner, repo, org, teamSlug),
		},
	}

	return &CallToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// readResource reads a resource by URI
func (h *Handler) readResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	// Basic resource reading - will be expanded in later tasks
	// For now, just return a placeholder
	content := []ResourceContent{
		{
			URI:      uri,
			MimeType: "application/json",
			Text:     fmt.Sprintf("Resource content for %s - implementation coming soon", uri),
		},
	}

	return &ReadResourceResult{
		Contents: content,
	}, nil
}
