package test

import (
"context"
"errors"
"net/http"
"testing"

"github.com/nicholasflintwillow/github-mcp/internal/client"
"github.com/nicholasflintwillow/github-mcp/internal/logger"
"github.com/nicholasflintwillow/github-mcp/test/fixtures"
"github.com/nicholasflintwillow/github-mcp/test/mocks"
)

func TestGitHubClient_GetUser(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		mockResponse   *http.Response
		mockError      error
		expectedError  bool
		expectedLogin  string
		expectedID     int64
	}{
		{
			name:          "successful user retrieval",
			username:      "testuser",
			mockResponse:  mocks.MockJSONResponse(200, fixtures.UserResponse),
			mockError:     nil,
			expectedError: false,
			expectedLogin: "testuser",
			expectedID:    12345,
		},
		{
			name:          "user not found",
			username:      "nonexistent",
			mockResponse:  mocks.MockErrorResponse(404, "Not Found"),
			mockError:     nil,
			expectedError: true,
		},
		{
			name:          "network error",
			username:      "testuser",
			mockResponse:  nil,
			mockError:     errors.New("network error"),
			expectedError: true,
		},
		{
			name:          "empty username",
			username:      "",
			mockResponse:  nil,
			mockError:     nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
// Create a test logger
testLogger, err := logger.New("DEBUG", "text")
if err != nil {
t.Fatalf("Failed to create test logger: %v", err)
}

mockClient := &mocks.MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					
					// Validate request URL and method
					expectedPath := "/users/" + tt.username
					if req.URL.Path != expectedPath {
						t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
					}
					if req.Method != "GET" {
						t.Errorf("Expected GET method, got %s", req.Method)
					}
					
					return tt.mockResponse, nil
				},
			}

			githubClient := client.NewGitHubClient("test-token", testLogger)
			githubClient.SetHTTPClient(mockClient)

			ctx := context.Background()
			user, err := githubClient.GetUser(ctx, tt.username)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user.Login != tt.expectedLogin {
				t.Errorf("Expected login %s, got %s", tt.expectedLogin, user.Login)
			}

			if user.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, user.ID)
			}
		})
	}
}

func TestGitHubClient_CheckUserFollowing(t *testing.T) {
	tests := []struct {
		name           string
		targetUser     string
		mockResponse   *http.Response
		mockError      error
		expectedError  bool
		expectedResult bool
	}{
		{
			name:           "user is following",
			targetUser:     "targetuser",
			mockResponse:   mocks.MockResponse(204, "", nil),
			mockError:      nil,
			expectedError:  false,
			expectedResult: true,
		},
		{
			name:           "user is not following",
			targetUser:     "targetuser",
			mockResponse:   mocks.MockResponse(404, "", nil),
			mockError:      nil,
			expectedError:  false,
			expectedResult: false,
		},
		{
			name:          "network error",
			targetUser:    "targetuser",
			mockResponse:  nil,
			mockError:     errors.New("network error"),
			expectedError: true,
		},
		{
			name:          "empty target user",
			targetUser:    "",
			mockResponse:  nil,
			mockError:     nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
// Create a test logger
testLogger, err := logger.New("DEBUG", "text")
if err != nil {
t.Fatalf("Failed to create test logger: %v", err)
}

mockClient := &mocks.MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					
					// Validate request URL and method
					expectedPath := "/user/following/" + tt.targetUser
					if req.URL.Path != expectedPath {
						t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
					}
					if req.Method != "GET" {
						t.Errorf("Expected GET method, got %s", req.Method)
					}
					
					return tt.mockResponse, nil
				},
			}

			githubClient := client.NewGitHubClient("test-token", testLogger)
			githubClient.SetHTTPClient(mockClient)

			ctx := context.Background()
			isFollowing, err := githubClient.CheckUserFollowing(ctx, tt.targetUser)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if isFollowing != tt.expectedResult {
				t.Errorf("Expected following status %v, got %v", tt.expectedResult, isFollowing)
			}
		})
	}
}
