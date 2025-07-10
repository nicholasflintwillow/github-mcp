package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the GitHub MCP server
type Config struct {
	// Server configuration
	Port int    `json:"port"`
	Host string `json:"host"`

	// GitHub API configuration
	GitHubToken string `json:"-"` // Don't serialize the token

	// Logging configuration
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// Cache configuration
	CacheTTL int `json:"cache_ttl"`

	// Performance configuration
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Port:                  8080,
		Host:                  "0.0.0.0",
		LogLevel:              "INFO",
		LogFormat:             "json",
		CacheTTL:              60,
		MaxConcurrentRequests: 100,
	}

	// Load GitHub token (required)
	cfg.GitHubToken = os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_PERSONAL_ACCESS_TOKEN environment variable is required")
	}

	// Load optional configuration
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 && p <= 65535 {
			cfg.Port = p
		} else {
			return nil, fmt.Errorf("invalid PORT value: %s", port)
		}
	}

	if host := os.Getenv("HOST"); host != "" {
		cfg.Host = host
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		logLevel = strings.ToUpper(logLevel)
		if isValidLogLevel(logLevel) {
			cfg.LogLevel = logLevel
		} else {
			return nil, fmt.Errorf("invalid LOG_LEVEL value: %s (must be DEBUG, INFO, WARN, or ERROR)", logLevel)
		}
	}

	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		logFormat = strings.ToLower(logFormat)
		if logFormat == "json" || logFormat == "text" {
			cfg.LogFormat = logFormat
		} else {
			return nil, fmt.Errorf("invalid LOG_FORMAT value: %s (must be 'json' or 'text')", logFormat)
		}
	}

	if cacheTTL := os.Getenv("CACHE_TTL"); cacheTTL != "" {
		if ttl, err := strconv.Atoi(cacheTTL); err == nil && ttl >= 0 {
			cfg.CacheTTL = ttl
		} else {
			return nil, fmt.Errorf("invalid CACHE_TTL value: %s", cacheTTL)
		}
	}

	if maxReq := os.Getenv("MAX_CONCURRENT_REQUESTS"); maxReq != "" {
		if max, err := strconv.Atoi(maxReq); err == nil && max > 0 {
			cfg.MaxConcurrentRequests = max
		} else {
			return nil, fmt.Errorf("invalid MAX_CONCURRENT_REQUESTS value: %s", maxReq)
		}
	}

	return cfg, nil
}

// isValidLogLevel checks if the provided log level is valid
func isValidLogLevel(level string) bool {
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.GitHubToken == "" {
		return fmt.Errorf("GitHub token is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if !isValidLogLevel(c.LogLevel) {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	if c.LogFormat != "json" && c.LogFormat != "text" {
		return fmt.Errorf("log format must be 'json' or 'text'")
	}

	if c.CacheTTL < 0 {
		return fmt.Errorf("cache TTL must be non-negative")
	}

	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max concurrent requests must be positive")
	}

	return nil
}
