package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/errors"
	"github.com/nicholasflintwillow/github-mcp/internal/logger"
)

// HTTPClientInterface defines the interface for HTTP clients
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	// GitHubAPIBaseURL is the base URL for GitHub API v4 (GraphQL)
	GitHubAPIBaseURL = "https://api.github.com"
	// GitHubAPIVersion is the API version to use
	GitHubAPIVersion = "2022-11-28"
	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = 30 * time.Second
	// DefaultUserAgent is the default user agent for requests
	DefaultUserAgent = "github-mcp-server/1.0.0"
)

// GitHubClient represents a GitHub API client
type GitHubClient struct {
	token      string
	baseURL    string
	httpClient HTTPClientInterface
	logger     *logger.Logger
	userAgent  string
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string, logger *logger.Logger) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: GitHubAPIBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		logger:    logger,
		userAgent: DefaultUserAgent,
	}
}

// SetTimeout sets the HTTP client timeout
func (c *GitHubClient) SetTimeout(timeout time.Duration) {
	if httpClient, ok := c.httpClient.(*http.Client); ok {
		httpClient.Timeout = timeout
	}
}

// SetUserAgent sets the user agent for requests

// SetHTTPClient sets the HTTP client for testing
func (c *GitHubClient) SetHTTPClient(client HTTPClientInterface) {
	c.httpClient = client
}
func (c *GitHubClient) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
}

// ValidateToken validates the GitHub Personal Access Token
func (c *GitHubClient) ValidateToken(ctx context.Context) error {
	c.logger.Info("Validating GitHub Personal Access Token")

	// Make a simple request to /user to validate the token
	req, err := c.newRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return errors.Wrap(err, errors.ErrorTypeInternal, "failed to create validation request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, errors.ErrorTypeNetwork, "failed to validate GitHub token")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.Authentication("invalid GitHub Personal Access Token")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("Token validation failed", "status", resp.StatusCode, "body", string(body))
		return errors.GitHubAPI(fmt.Sprintf("GitHub API returned status %d", resp.StatusCode))
	}

	// Parse the response to get user info
	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		c.logger.Warn("Failed to parse user response during token validation", "error", err)
		// Don't fail validation just because we can't parse the response
	} else {
		if login, ok := user["login"].(string); ok {
			c.logger.Info("GitHub token validated successfully", "user", login)
		}
	}

	return nil
}

// Get performs a GET request to the GitHub API
func (c *GitHubClient) Get(ctx context.Context, endpoint string, params map[string]string) (*APIResponse, error) {
	return c.request(ctx, "GET", endpoint, params, nil)
}

// Post performs a POST request to the GitHub API
func (c *GitHubClient) Post(ctx context.Context, endpoint string, body interface{}) (*APIResponse, error) {
	return c.request(ctx, "POST", endpoint, nil, body)
}

// Put performs a PUT request to the GitHub API
func (c *GitHubClient) Put(ctx context.Context, endpoint string, body interface{}) (*APIResponse, error) {
	return c.request(ctx, "PUT", endpoint, nil, body)
}

// Delete performs a DELETE request to the GitHub API
func (c *GitHubClient) Delete(ctx context.Context, endpoint string) (*APIResponse, error) {
	return c.request(ctx, "DELETE", endpoint, nil, nil)
}

// Patch performs a PATCH request to the GitHub API
func (c *GitHubClient) Patch(ctx context.Context, endpoint string, body interface{}) (*APIResponse, error) {
	return c.request(ctx, "PATCH", endpoint, nil, body)
}

// request performs an HTTP request to the GitHub API
func (c *GitHubClient) request(ctx context.Context, method, endpoint string, params map[string]string, body interface{}) (*APIResponse, error) {
	req, err := c.newRequest(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	if params != nil && len(params) > 0 {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	c.logger.Debug("Making GitHub API request",
		"method", method,
		"url", req.URL.String(),
		"endpoint", endpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeNetwork, "GitHub API request failed")
	}
	defer resp.Body.Close()

	return c.parseResponse(resp)
}

// newRequest creates a new HTTP request with proper headers
func (c *GitHubClient) newRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	// Build full URL
	fullURL := c.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrorTypeValidation, "failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "failed to create HTTP request")
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", GitHubAPIVersion)
	req.Header.Set("User-Agent", c.userAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// parseResponse parses the HTTP response from GitHub API
func (c *GitHubClient) parseResponse(resp *http.Response) (*APIResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeNetwork, "failed to read response body")
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}

	// Parse rate limit headers
	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		apiResp.RateLimit.Limit = limit
	}
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		apiResp.RateLimit.Remaining = remaining
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		apiResp.RateLimit.Reset = reset
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return apiResp, c.handleAPIError(resp.StatusCode, body)
	}

	// Try to parse JSON response
	if len(body) > 0 && resp.Header.Get("Content-Type") == "application/json" {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			c.logger.Warn("Failed to parse JSON response", "error", err)
			// Don't return error, just log warning
		} else {
			apiResp.Data = jsonData
		}
	}

	return apiResp, nil
}

// handleAPIError handles GitHub API errors
func (c *GitHubClient) handleAPIError(statusCode int, body []byte) error {
	var errorResp struct {
		Message          string `json:"message"`
		DocumentationURL string `json:"documentation_url"`
		Errors           []struct {
			Resource string `json:"resource"`
			Field    string `json:"field"`
			Code     string `json:"code"`
		} `json:"errors"`
	}

	// Try to parse error response
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error, return a generic error
		return errors.GitHubAPI(fmt.Sprintf("GitHub API error (status %d): %s", statusCode, string(body)))
	}

	message := errorResp.Message
	if message == "" {
		message = fmt.Sprintf("GitHub API error (status %d)", statusCode)
	}

	// Map status codes to error types
	switch statusCode {
	case http.StatusUnauthorized:
		return errors.Authentication(message)
	case http.StatusForbidden:
		return errors.Authorization(message)
	case http.StatusNotFound:
		return errors.NotFound(message)
	case http.StatusUnprocessableEntity:
		return errors.Validation(message)
	case http.StatusTooManyRequests:
		return errors.RateLimit(message)
	default:
		return errors.GitHubAPI(message)
	}
}

// APIResponse represents a response from the GitHub API
type APIResponse struct {
	StatusCode int           `json:"status_code"`
	Headers    http.Header   `json:"headers"`
	Body       []byte        `json:"body"`
	Data       interface{}   `json:"data,omitempty"`
	RateLimit  RateLimitInfo `json:"rate_limit"`
}

// RateLimitInfo contains rate limit information from GitHub API
type RateLimitInfo struct {
	Limit     string `json:"limit"`
	Remaining string `json:"remaining"`
	Reset     string `json:"reset"`
}

// GetJSON unmarshals the response body into the provided interface
func (r *APIResponse) GetJSON(v interface{}) error {
	if len(r.Body) == 0 {
		return errors.Validation("empty response body")
	}

	if err := json.Unmarshal(r.Body, v); err != nil {
		return errors.Wrap(err, errors.ErrorTypeValidation, "failed to unmarshal response")
	}

	return nil
}

// IsSuccess returns true if the response indicates success
func (r *APIResponse) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// GitHub User data structures

// User represents a GitHub user
type User struct {
	Login             string  `json:"login"`
	ID                int64   `json:"id"`
	NodeID            string  `json:"node_id"`
	AvatarURL         string  `json:"avatar_url"`
	GravatarID        string  `json:"gravatar_id"`
	URL               string  `json:"url"`
	HTMLURL           string  `json:"html_url"`
	FollowersURL      string  `json:"followers_url"`
	FollowingURL      string  `json:"following_url"`
	GistsURL          string  `json:"gists_url"`
	StarredURL        string  `json:"starred_url"`
	SubscriptionsURL  string  `json:"subscriptions_url"`
	OrganizationsURL  string  `json:"organizations_url"`
	ReposURL          string  `json:"repos_url"`
	EventsURL         string  `json:"events_url"`
	ReceivedEventsURL string  `json:"received_events_url"`
	Type              string  `json:"type"`
	SiteAdmin         bool    `json:"site_admin"`
	Name              *string `json:"name"`
	Company           *string `json:"company"`
	Blog              *string `json:"blog"`
	Location          *string `json:"location"`
	Email             *string `json:"email"`
	Hireable          *bool   `json:"hireable"`
	Bio               *string `json:"bio"`
	TwitterUsername   *string `json:"twitter_username"`
	PublicRepos       int     `json:"public_repos"`
	PublicGists       int     `json:"public_gists"`
	Followers         int     `json:"followers"`
	Following         int     `json:"following"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// UserSearchResult represents the result of a user search
type UserSearchResult struct {
	TotalCount        int    `json:"total_count"`
	IncompleteResults bool   `json:"incomplete_results"`
	Items             []User `json:"items"`
}

// GitHub Users API client functions

// GetUser gets a user by username
func (c *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
	c.logger.Debug("Getting user", "username", username)

	resp, err := c.Get(ctx, fmt.Sprintf("/users/%s", username), nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := resp.GetJSON(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAuthenticatedUser gets the authenticated user
func (c *GitHubClient) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	c.logger.Debug("Getting authenticated user")

	resp, err := c.Get(ctx, "/user", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := resp.GetJSON(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateAuthenticatedUser updates the authenticated user
func (c *GitHubClient) UpdateAuthenticatedUser(ctx context.Context, updates map[string]interface{}) (*User, error) {
	c.logger.Debug("Updating authenticated user")

	resp, err := c.Patch(ctx, "/user", updates)
	if err != nil {
		return nil, err
	}

	var user User
	if err := resp.GetJSON(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ListUsers lists all users
func (c *GitHubClient) ListUsers(ctx context.Context, since int64, perPage int) ([]User, error) {
	c.logger.Debug("Listing users", "since", since, "per_page", perPage)

	params := make(map[string]string)
	if since > 0 {
		params["since"] = fmt.Sprintf("%d", since)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, "/users", params)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := resp.GetJSON(&users); err != nil {
		return nil, err
	}

	return users, nil
}

// ListUserFollowers lists followers of a user
func (c *GitHubClient) ListUserFollowers(ctx context.Context, username string, page, perPage int) ([]User, error) {
	c.logger.Debug("Listing user followers", "username", username, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/users/%s/followers", username), params)
	if err != nil {
		return nil, err
	}

	var followers []User
	if err := resp.GetJSON(&followers); err != nil {
		return nil, err
	}

	return followers, nil
}

// ListUserFollowing lists users followed by a user
func (c *GitHubClient) ListUserFollowing(ctx context.Context, username string, page, perPage int) ([]User, error) {
	c.logger.Debug("Listing user following", "username", username, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/users/%s/following", username), params)
	if err != nil {
		return nil, err
	}

	var following []User
	if err := resp.GetJSON(&following); err != nil {
		return nil, err
	}

	return following, nil
}

// CheckUserFollowing checks if the authenticated user follows another user
func (c *GitHubClient) CheckUserFollowing(ctx context.Context, username string) (bool, error) {
	c.logger.Debug("Checking if user is followed", "username", username)

	resp, err := c.Get(ctx, fmt.Sprintf("/user/following/%s", username), nil)
	if err != nil {
		// If it's a 404, the user is not followed
		if appErr, ok := err.(*errors.AppError); ok && appErr.Type == errors.ErrorTypeNotFound {
			return false, nil
		}
		return false, err
	}

	return resp.StatusCode == 204, nil
}

// FollowUser follows a user
func (c *GitHubClient) FollowUser(ctx context.Context, username string) error {
	c.logger.Debug("Following user", "username", username)

	_, err := c.Put(ctx, fmt.Sprintf("/user/following/%s", username), nil)
	return err
}

// UnfollowUser unfollows a user
func (c *GitHubClient) UnfollowUser(ctx context.Context, username string) error {
	c.logger.Debug("Unfollowing user", "username", username)

	_, err := c.Delete(ctx, fmt.Sprintf("/user/following/%s", username))
	return err
}

// GitHub Organization data structures

// Organization represents a GitHub organization
type Organization struct {
	Login                                          string  `json:"login"`
	ID                                             int64   `json:"id"`
	NodeID                                         string  `json:"node_id"`
	URL                                            string  `json:"url"`
	ReposURL                                       string  `json:"repos_url"`
	EventsURL                                      string  `json:"events_url"`
	HooksURL                                       string  `json:"hooks_url"`
	IssuesURL                                      string  `json:"issues_url"`
	MembersURL                                     string  `json:"members_url"`
	PublicMembersURL                               string  `json:"public_members_url"`
	AvatarURL                                      string  `json:"avatar_url"`
	Description                                    *string `json:"description"`
	GravatarID                                     *string `json:"gravatar_id"`
	Name                                           *string `json:"name"`
	Company                                        *string `json:"company"`
	Blog                                           *string `json:"blog"`
	Location                                       *string `json:"location"`
	Email                                          *string `json:"email"`
	TwitterUsername                                *string `json:"twitter_username"`
	IsVerified                                     *bool   `json:"is_verified"`
	HasOrganizationProjects                        bool    `json:"has_organization_projects"`
	HasRepositoryProjects                          bool    `json:"has_repository_projects"`
	PublicRepos                                    int     `json:"public_repos"`
	PublicGists                                    int     `json:"public_gists"`
	Followers                                      int     `json:"followers"`
	Following                                      int     `json:"following"`
	HTMLURL                                        string  `json:"html_url"`
	CreatedAt                                      string  `json:"created_at"`
	UpdatedAt                                      string  `json:"updated_at"`
	Type                                           string  `json:"type"`
	TotalPrivateRepos                              *int    `json:"total_private_repos"`
	OwnedPrivateRepos                              *int    `json:"owned_private_repos"`
	PrivateGists                                   *int    `json:"private_gists"`
	DiskUsage                                      *int    `json:"disk_usage"`
	Collaborators                                  *int    `json:"collaborators"`
	BillingEmail                                   *string `json:"billing_email"`
	Plan                                           *Plan   `json:"plan"`
	DefaultRepositoryPermission                    *string `json:"default_repository_permission"`
	MembersCanCreateRepos                          *bool   `json:"members_can_create_repositories"`
	TwoFactorRequirementEnabled                    *bool   `json:"two_factor_requirement_enabled"`
	MembersAllowedRepositoryCreationType           *string `json:"members_allowed_repository_creation_type"`
	MembersCanCreatePublicRepos                    *bool   `json:"members_can_create_public_repositories"`
	MembersCanCreatePrivateRepos                   *bool   `json:"members_can_create_private_repositories"`
	MembersCanCreateInternalRepos                  *bool   `json:"members_can_create_internal_repositories"`
	MembersCanCreatePages                          *bool   `json:"members_can_create_pages"`
	MembersCanCreatePublicPages                    *bool   `json:"members_can_create_public_pages"`
	MembersCanCreatePrivatePages                   *bool   `json:"members_can_create_private_pages"`
	MembersCanForkPrivateRepos                     *bool   `json:"members_can_fork_private_repositories"`
	WebCommitSignoffRequired                       *bool   `json:"web_commit_signoff_required"`
	MembersCanCreateRepoInDefaultOrg               *bool   `json:"members_can_create_repositories_in_default_org"`
	DependencyGraphEnabledForNewRepos              *bool   `json:"dependency_graph_enabled_for_new_repositories"`
	DependabotAlertsEnabledForNewRepos             *bool   `json:"dependabot_alerts_enabled_for_new_repositories"`
	DependabotSecurityUpdatesEnabledForNewRepos    *bool   `json:"dependabot_security_updates_enabled_for_new_repositories"`
	AdvancedSecurityEnabledForNewRepos             *bool   `json:"advanced_security_enabled_for_new_repositories"`
	SecretScanningEnabledForNewRepos               *bool   `json:"secret_scanning_enabled_for_new_repositories"`
	SecretScanningPushProtectionEnabledForNewRepos *bool   `json:"secret_scanning_push_protection_enabled_for_new_repositories"`
	SecretScanningValidityChecksEnabled            *bool   `json:"secret_scanning_validity_checks_enabled"`
}

// Plan represents a GitHub plan
type Plan struct {
	Name         string `json:"name"`
	Space        int    `json:"space"`
	PrivateRepos int    `json:"private_repos"`
	FilledSeats  *int   `json:"filled_seats"`
	Seats        *int   `json:"seats"`
}

// OrganizationMember represents a member of an organization
type OrganizationMember struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

// GitHub Organizations API client functions

// GetOrganization gets an organization by name
func (c *GitHubClient) GetOrganization(ctx context.Context, org string) (*Organization, error) {
	c.logger.Debug("Getting organization", "org", org)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s", org), nil)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := resp.GetJSON(&organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

// UpdateOrganization updates an organization
func (c *GitHubClient) UpdateOrganization(ctx context.Context, org string, updates map[string]interface{}) (*Organization, error) {
	c.logger.Debug("Updating organization", "org", org)

	resp, err := c.Patch(ctx, fmt.Sprintf("/orgs/%s", org), updates)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := resp.GetJSON(&organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

// ListOrganizations lists all organizations
func (c *GitHubClient) ListOrganizations(ctx context.Context, since int64, perPage int) ([]Organization, error) {
	c.logger.Debug("Listing organizations", "since", since, "per_page", perPage)

	params := make(map[string]string)
	if since > 0 {
		params["since"] = fmt.Sprintf("%d", since)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, "/organizations", params)
	if err != nil {
		return nil, err
	}

	var organizations []Organization
	if err := resp.GetJSON(&organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

// ListUserOrganizations lists organizations for a user
func (c *GitHubClient) ListUserOrganizations(ctx context.Context, username string, page, perPage int) ([]Organization, error) {
	c.logger.Debug("Listing user organizations", "username", username, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/users/%s/orgs", username), params)
	if err != nil {
		return nil, err
	}

	var organizations []Organization
	if err := resp.GetJSON(&organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

// ListAuthenticatedUserOrganizations lists organizations for the authenticated user
func (c *GitHubClient) ListAuthenticatedUserOrganizations(ctx context.Context, page, perPage int) ([]Organization, error) {
	c.logger.Debug("Listing authenticated user organizations", "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, "/user/orgs", params)
	if err != nil {
		return nil, err
	}

	var organizations []Organization
	if err := resp.GetJSON(&organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

// ListOrganizationMembers lists members of an organization
func (c *GitHubClient) ListOrganizationMembers(ctx context.Context, org string, filter string, role string, page, perPage int) ([]OrganizationMember, error) {
	c.logger.Debug("Listing organization members", "org", org, "filter", filter, "role", role, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if filter != "" {
		params["filter"] = filter
	}
	if role != "" {
		params["role"] = role
	}
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/members", org), params)
	if err != nil {
		return nil, err
	}

	var members []OrganizationMember
	if err := resp.GetJSON(&members); err != nil {
		return nil, err
	}

	return members, nil
}

// CheckOrganizationMembership checks if a user is a member of an organization
func (c *GitHubClient) CheckOrganizationMembership(ctx context.Context, org, username string) (bool, error) {
	c.logger.Debug("Checking organization membership", "org", org, "username", username)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/members/%s", org, username), nil)
	if err != nil {
		// If it's a 404, the user is not a member
		if appErr, ok := err.(*errors.AppError); ok && appErr.Type == errors.ErrorTypeNotFound {
			return false, nil
		}
		return false, err
	}

	return resp.StatusCode == 204, nil
}

// CheckPublicOrganizationMembership checks if a user is a public member of an organization
func (c *GitHubClient) CheckPublicOrganizationMembership(ctx context.Context, org, username string) (bool, error) {
	c.logger.Debug("Checking public organization membership", "org", org, "username", username)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/public_members/%s", org, username), nil)
	if err != nil {
		// If it's a 404, the user is not a public member
		if appErr, ok := err.(*errors.AppError); ok && appErr.Type == errors.ErrorTypeNotFound {
			return false, nil
		}
		return false, err
	}

	return resp.StatusCode == 204, nil
}

// GitHub Teams data structures

// Team represents a GitHub team
type Team struct {
	ID              int64   `json:"id"`
	NodeID          string  `json:"node_id"`
	URL             string  `json:"url"`
	HTMLURL         string  `json:"html_url"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Description     *string `json:"description"`
	Privacy         string  `json:"privacy"`
	Permission      string  `json:"permission"`
	MembersURL      string  `json:"members_url"`
	RepositoriesURL string  `json:"repositories_url"`
	Parent          *Team   `json:"parent"`
	MembersCount    int     `json:"members_count"`
	ReposCount      int     `json:"repos_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Organization    struct {
		Login                   string `json:"login"`
		ID                      int64  `json:"id"`
		NodeID                  string `json:"node_id"`
		URL                     string `json:"url"`
		ReposURL                string `json:"repos_url"`
		EventsURL               string `json:"events_url"`
		HooksURL                string `json:"hooks_url"`
		IssuesURL               string `json:"issues_url"`
		MembersURL              string `json:"members_url"`
		PublicMembersURL        string `json:"public_members_url"`
		AvatarURL               string `json:"avatar_url"`
		Description             string `json:"description"`
		GravatarID              string `json:"gravatar_id"`
		Name                    string `json:"name"`
		Company                 string `json:"company"`
		Blog                    string `json:"blog"`
		Location                string `json:"location"`
		Email                   string `json:"email"`
		TwitterUsername         string `json:"twitter_username"`
		IsVerified              bool   `json:"is_verified"`
		HasOrganizationProjects bool   `json:"has_organization_projects"`
		HasRepositoryProjects   bool   `json:"has_repository_projects"`
		PublicRepos             int    `json:"public_repos"`
		PublicGists             int    `json:"public_gists"`
		Followers               int    `json:"followers"`
		Following               int    `json:"following"`
		HTMLURL                 string `json:"html_url"`
		CreatedAt               string `json:"created_at"`
		UpdatedAt               string `json:"updated_at"`
		Type                    string `json:"type"`
	} `json:"organization"`
}

// TeamMember represents a member of a team
type TeamMember struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

// TeamMembership represents a team membership
type TeamMembership struct {
	URL   string `json:"url"`
	Role  string `json:"role"`
	State string `json:"state"`
}

// TeamRepository represents a repository associated with a team
type TeamRepository struct {
	ID               int64    `json:"id"`
	NodeID           string   `json:"node_id"`
	Name             string   `json:"name"`
	FullName         string   `json:"full_name"`
	Private          bool     `json:"private"`
	Owner            User     `json:"owner"`
	HTMLURL          string   `json:"html_url"`
	Description      *string  `json:"description"`
	Fork             bool     `json:"fork"`
	URL              string   `json:"url"`
	ArchiveURL       string   `json:"archive_url"`
	AssigneesURL     string   `json:"assignees_url"`
	BlobsURL         string   `json:"blobs_url"`
	BranchesURL      string   `json:"branches_url"`
	CollaboratorsURL string   `json:"collaborators_url"`
	CommentsURL      string   `json:"comments_url"`
	CommitsURL       string   `json:"commits_url"`
	CompareURL       string   `json:"compare_url"`
	ContentsURL      string   `json:"contents_url"`
	ContributorsURL  string   `json:"contributors_url"`
	DeploymentsURL   string   `json:"deployments_url"`
	DownloadsURL     string   `json:"downloads_url"`
	EventsURL        string   `json:"events_url"`
	ForksURL         string   `json:"forks_url"`
	GitCommitsURL    string   `json:"git_commits_url"`
	GitRefsURL       string   `json:"git_refs_url"`
	GitTagsURL       string   `json:"git_tags_url"`
	GitURL           string   `json:"git_url"`
	IssueCommentURL  string   `json:"issue_comment_url"`
	IssueEventsURL   string   `json:"issue_events_url"`
	IssuesURL        string   `json:"issues_url"`
	KeysURL          string   `json:"keys_url"`
	LabelsURL        string   `json:"labels_url"`
	LanguagesURL     string   `json:"languages_url"`
	MergesURL        string   `json:"merges_url"`
	MilestonesURL    string   `json:"milestones_url"`
	NotificationsURL string   `json:"notifications_url"`
	PullsURL         string   `json:"pulls_url"`
	ReleasesURL      string   `json:"releases_url"`
	SSHURL           string   `json:"ssh_url"`
	StargazersURL    string   `json:"stargazers_url"`
	StatusesURL      string   `json:"statuses_url"`
	SubscribersURL   string   `json:"subscribers_url"`
	SubscriptionURL  string   `json:"subscription_url"`
	TagsURL          string   `json:"tags_url"`
	TeamsURL         string   `json:"teams_url"`
	TreesURL         string   `json:"trees_url"`
	CloneURL         string   `json:"clone_url"`
	MirrorURL        *string  `json:"mirror_url"`
	HooksURL         string   `json:"hooks_url"`
	SvnURL           string   `json:"svn_url"`
	Homepage         *string  `json:"homepage"`
	Language         *string  `json:"language"`
	ForksCount       int      `json:"forks_count"`
	StargazersCount  int      `json:"stargazers_count"`
	WatchersCount    int      `json:"watchers_count"`
	Size             int      `json:"size"`
	DefaultBranch    string   `json:"default_branch"`
	OpenIssuesCount  int      `json:"open_issues_count"`
	IsTemplate       bool     `json:"is_template"`
	Topics           []string `json:"topics"`
	HasIssues        bool     `json:"has_issues"`
	HasProjects      bool     `json:"has_projects"`
	HasWiki          bool     `json:"has_wiki"`
	HasPages         bool     `json:"has_pages"`
	HasDownloads     bool     `json:"has_downloads"`
	Archived         bool     `json:"archived"`
	Disabled         bool     `json:"disabled"`
	Visibility       string   `json:"visibility"`
	PushedAt         *string  `json:"pushed_at"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	Permissions      struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	RoleName *string `json:"role_name"`
}

// GitHub Teams API client functions

// ListTeams lists teams in an organization
func (c *GitHubClient) ListTeams(ctx context.Context, org string, page, perPage int) ([]Team, error) {
	c.logger.Debug("Listing teams", "org", org, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams", org), params)
	if err != nil {
		return nil, err
	}

	var teams []Team
	if err := resp.GetJSON(&teams); err != nil {
		return nil, err
	}

	return teams, nil
}

// GetTeam gets a team by organization and team slug
func (c *GitHubClient) GetTeam(ctx context.Context, org, teamSlug string) (*Team, error) {
	c.logger.Debug("Getting team", "org", org, "team_slug", teamSlug)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams/%s", org, teamSlug), nil)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := resp.GetJSON(&team); err != nil {
		return nil, err
	}

	return &team, nil
}

// CreateTeam creates a new team in an organization
func (c *GitHubClient) CreateTeam(ctx context.Context, org string, teamData map[string]interface{}) (*Team, error) {
	c.logger.Debug("Creating team", "org", org)

	resp, err := c.Post(ctx, fmt.Sprintf("/orgs/%s/teams", org), teamData)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := resp.GetJSON(&team); err != nil {
		return nil, err
	}

	return &team, nil
}

// UpdateTeam updates a team
func (c *GitHubClient) UpdateTeam(ctx context.Context, org, teamSlug string, updates map[string]interface{}) (*Team, error) {
	c.logger.Debug("Updating team", "org", org, "team_slug", teamSlug)

	resp, err := c.Patch(ctx, fmt.Sprintf("/orgs/%s/teams/%s", org, teamSlug), updates)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := resp.GetJSON(&team); err != nil {
		return nil, err
	}

	return &team, nil
}

// DeleteTeam deletes a team
func (c *GitHubClient) DeleteTeam(ctx context.Context, org, teamSlug string) error {
	c.logger.Debug("Deleting team", "org", org, "team_slug", teamSlug)

	_, err := c.Delete(ctx, fmt.Sprintf("/orgs/%s/teams/%s", org, teamSlug))
	return err
}

// ListTeamMembers lists members of a team
func (c *GitHubClient) ListTeamMembers(ctx context.Context, org, teamSlug string, role string, page, perPage int) ([]TeamMember, error) {
	c.logger.Debug("Listing team members", "org", org, "team_slug", teamSlug, "role", role, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if role != "" {
		params["role"] = role
	}
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams/%s/members", org, teamSlug), params)
	if err != nil {
		return nil, err
	}

	var members []TeamMember
	if err := resp.GetJSON(&members); err != nil {
		return nil, err
	}

	return members, nil
}

// GetTeamMembership gets team membership for a user
func (c *GitHubClient) GetTeamMembership(ctx context.Context, org, teamSlug, username string) (*TeamMembership, error) {
	c.logger.Debug("Getting team membership", "org", org, "team_slug", teamSlug, "username", username)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", org, teamSlug, username), nil)
	if err != nil {
		return nil, err
	}

	var membership TeamMembership
	if err := resp.GetJSON(&membership); err != nil {
		return nil, err
	}

	return &membership, nil
}

// AddTeamMembership adds or updates team membership for a user
func (c *GitHubClient) AddTeamMembership(ctx context.Context, org, teamSlug, username string, role string) (*TeamMembership, error) {
	c.logger.Debug("Adding team membership", "org", org, "team_slug", teamSlug, "username", username, "role", role)

	body := map[string]interface{}{}
	if role != "" {
		body["role"] = role
	}

	resp, err := c.Put(ctx, fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", org, teamSlug, username), body)
	if err != nil {
		return nil, err
	}

	var membership TeamMembership
	if err := resp.GetJSON(&membership); err != nil {
		return nil, err
	}

	return &membership, nil
}

// RemoveTeamMembership removes a user from a team
func (c *GitHubClient) RemoveTeamMembership(ctx context.Context, org, teamSlug, username string) error {
	c.logger.Debug("Removing team membership", "org", org, "team_slug", teamSlug, "username", username)

	_, err := c.Delete(ctx, fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", org, teamSlug, username))
	return err
}

// ListTeamRepositories lists repositories for a team
func (c *GitHubClient) ListTeamRepositories(ctx context.Context, org, teamSlug string, page, perPage int) ([]TeamRepository, error) {
	c.logger.Debug("Listing team repositories", "org", org, "team_slug", teamSlug, "page", page, "per_page", perPage)

	params := make(map[string]string)
	if page > 0 {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", perPage)
	}

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams/%s/repos", org, teamSlug), params)
	if err != nil {
		return nil, err
	}

	var repositories []TeamRepository
	if err := resp.GetJSON(&repositories); err != nil {
		return nil, err
	}

	return repositories, nil
}

// CheckTeamRepository checks if a team has access to a repository
func (c *GitHubClient) CheckTeamRepository(ctx context.Context, org, teamSlug, owner, repo string) (bool, error) {
	c.logger.Debug("Checking team repository access", "org", org, "team_slug", teamSlug, "owner", owner, "repo", repo)

	resp, err := c.Get(ctx, fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", org, teamSlug, owner, repo), nil)
	if err != nil {
		// If it's a 404, the team doesn't have access to the repository
		if appErr, ok := err.(*errors.AppError); ok && appErr.Type == errors.ErrorTypeNotFound {
			return false, nil
		}
		return false, err
	}

	return resp.StatusCode == 200, nil
}

// AddTeamRepository adds a repository to a team
func (c *GitHubClient) AddTeamRepository(ctx context.Context, org, teamSlug, owner, repo string, permission string) error {
	c.logger.Debug("Adding team repository", "org", org, "team_slug", teamSlug, "owner", owner, "repo", repo, "permission", permission)

	body := map[string]interface{}{}
	if permission != "" {
		body["permission"] = permission
	}

	_, err := c.Put(ctx, fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", org, teamSlug, owner, repo), body)
	return err
}

// RemoveTeamRepository removes a repository from a team
func (c *GitHubClient) RemoveTeamRepository(ctx context.Context, org, teamSlug, owner, repo string) error {
	c.logger.Debug("Removing team repository", "org", org, "team_slug", teamSlug, "owner", owner, "repo", repo)

	_, err := c.Delete(ctx, fmt.Sprintf("/orgs/%s/teams/%s/repos/%s/%s", org, teamSlug, owner, repo))
	return err
}
