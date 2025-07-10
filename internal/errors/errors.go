package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeAuthentication represents authentication errors
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeAuthorization represents authorization errors
	ErrorTypeAuthorization ErrorType = "authorization"
	// ErrorTypeNotFound represents not found errors
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeRateLimit represents rate limit errors
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeInternal represents internal server errors
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeGitHubAPI represents GitHub API errors
	ErrorTypeGitHubAPI ErrorType = "github_api"
	// ErrorTypeNetwork represents network errors
	ErrorTypeNetwork ErrorType = "network"
)

// AppError represents an application error with context
type AppError struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new AppError
func New(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: getDefaultStatusCode(errorType),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: getDefaultStatusCode(errorType),
		Cause:      err,
	}
}

// getDefaultStatusCode returns the default HTTP status code for an error type
func getDefaultStatusCode(errorType ErrorType) int {
	switch errorType {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypeAuthorization:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeGitHubAPI:
		return http.StatusBadGateway
	case ErrorTypeNetwork:
		return http.StatusServiceUnavailable
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Validation creates a validation error
func Validation(message string) *AppError {
	return New(ErrorTypeValidation, message)
}

// Authentication creates an authentication error
func Authentication(message string) *AppError {
	return New(ErrorTypeAuthentication, message)
}

// Authorization creates an authorization error
func Authorization(message string) *AppError {
	return New(ErrorTypeAuthorization, message)
}

// NotFound creates a not found error
func NotFound(message string) *AppError {
	return New(ErrorTypeNotFound, message)
}

// RateLimit creates a rate limit error
func RateLimit(message string) *AppError {
	return New(ErrorTypeRateLimit, message)
}

// Internal creates an internal server error
func Internal(message string) *AppError {
	return New(ErrorTypeInternal, message)
}

// GitHubAPI creates a GitHub API error
func GitHubAPI(message string) *AppError {
	return New(ErrorTypeGitHubAPI, message)
}

// Network creates a network error
func Network(message string) *AppError {
	return New(ErrorTypeNetwork, message)
}

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// GetStatusCode extracts the HTTP status code from an error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetType extracts the error type from an error
func GetType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ErrorTypeInternal
}
