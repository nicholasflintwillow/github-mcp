package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nicholasflintwillow/github-mcp/internal/client"
	"github.com/nicholasflintwillow/github-mcp/internal/config"
	"github.com/nicholasflintwillow/github-mcp/internal/errors"
	"github.com/nicholasflintwillow/github-mcp/internal/logger"
	"github.com/nicholasflintwillow/github-mcp/internal/mcp"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	logger        *logger.Logger
	httpServer    *http.Server
	mux           *http.ServeMux
	githubClient  *client.GitHubClient
	mcpHandler    *mcp.Handler
	streamHandler *mcp.StreamHandler
}

// New creates a new server instance
func New(cfg *config.Config, log *logger.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeValidation, "invalid configuration")
	}

	// Create GitHub client
	githubClient := client.NewGitHubClient(cfg.GitHubToken, log)

	// Validate GitHub token
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Validating GitHub Personal Access Token...")
	if err := githubClient.ValidateToken(ctx); err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeAuthentication, "GitHub token validation failed")
	}
	log.Info("GitHub Personal Access Token validated successfully")

	// Create MCP handler
	mcpHandler := mcp.NewHandler(githubClient, log)

	// Create stream handler
	streamHandler := mcp.NewStreamHandler(log)

	// Connect MCP handler with the streamer
	mcpHandler.SetStreamer(streamHandler.GetStreamer())

	s := &Server{
		config:        cfg,
		logger:        log,
		mux:           http.NewServeMux(),
		githubClient:  githubClient,
		mcpHandler:    mcpHandler,
		streamHandler: streamHandler,
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      s.middlewareChain(s.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start the stream handler
	s.streamHandler.Start()

	s.logger.Info("Starting HTTP server", "address", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, errors.ErrorTypeInternal, "failed to start HTTP server")
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")

	// Stop the stream handler
	s.streamHandler.Stop()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return errors.Wrap(err, errors.ErrorTypeInternal, "failed to shutdown HTTP server")
	}

	return nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.mux.HandleFunc("/health", s.handleHealth)

	// Ready check endpoint
	s.mux.HandleFunc("/ready", s.handleReady)

	// MCP endpoints
	s.mux.HandleFunc("/mcp/request", s.handleMCPRequest)
	s.mux.HandleFunc("/mcp/stream", s.handleMCPStream)

	// Legacy MCP endpoint (for backward compatibility)
	s.mux.HandleFunc("/mcp/", s.handleMCP)

	// Catch-all for undefined routes
	s.mux.HandleFunc("/", s.handleNotFound)
}

// middlewareChain applies middleware to the handler
func (s *Server) middlewareChain(next http.Handler) http.Handler {
	return s.loggingMiddleware(
		s.recoveryMiddleware(
			s.corsMiddleware(next),
		),
	)
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(rw, r)

		// Log request
		duration := time.Since(start)
		s.logger.LogRequest(
			r.Method,
			r.URL.Path,
			r.UserAgent(),
			r.RemoteAddr,
			rw.statusCode,
			duration.String(),
		)
	})
}

// recoveryMiddleware recovers from panics
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("Panic recovered", "error", err, "path", r.URL.Path)
				s.writeErrorResponse(w, errors.Internal("internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
