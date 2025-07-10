package mocks

import (
"io"
"net/http"
"strconv"
"strings"
)

// HTTPClientInterface defines the interface for HTTP clients
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// MockHTTPClient is a mock implementation of HTTPClientInterface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do executes the mock function
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// MockResponse creates a mock HTTP response
func MockResponse(statusCode int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	
	for key, value := range headers {
		resp.Header.Set(key, value)
	}
	
	return resp
}

// MockJSONResponse creates a mock HTTP response with JSON content
func MockJSONResponse(statusCode int, jsonBody string) *http.Response {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	return MockResponse(statusCode, jsonBody, headers)
}

// MockErrorResponse creates a mock HTTP error response
func MockErrorResponse(statusCode int, message string) *http.Response {
	errorBody := `{"message":"` + message + `","documentation_url":"https://docs.github.com/rest"}`
	return MockJSONResponse(statusCode, errorBody)
}

// MockPaginatedResponse creates a mock HTTP response with pagination headers
func MockPaginatedResponse(statusCode int, jsonBody string, page, perPage, total int) *http.Response {
	resp := MockJSONResponse(statusCode, jsonBody)
	
	// Add pagination headers
	resp.Header.Set("X-RateLimit-Limit", "5000")
	resp.Header.Set("X-RateLimit-Remaining", "4999")
	resp.Header.Set("X-RateLimit-Reset", "1640995200")
	
	// Calculate pagination links
	if page > 1 {
		resp.Header.Set("Link", `<https://api.github.com/users?page=`+strconv.Itoa(page-1)+`&per_page=`+strconv.Itoa(perPage)+`>; rel="prev"`)
	}
	if page*perPage < total {
		if resp.Header.Get("Link") != "" {
			resp.Header.Set("Link", resp.Header.Get("Link")+`, <https://api.github.com/users?page=`+strconv.Itoa(page+1)+`&per_page=`+strconv.Itoa(perPage)+`>; rel="next"`)
		} else {
			resp.Header.Set("Link", `<https://api.github.com/users?page=`+strconv.Itoa(page+1)+`&per_page=`+strconv.Itoa(perPage)+`>; rel="next"`)
		}
	}
	
	return resp
}
