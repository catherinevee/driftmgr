package api

import (
	"context"
	"net/http"
	"time"
)

// NewAPIServer creates a new API server for testing
func NewAPIServer(address string) *TestServer {
	return &TestServer{
		address: address,
		router:  http.NewServeMux(),
	}
}

// TestServer is a simplified server for testing
type TestServer struct {
	address string
	router  *http.ServeMux
	server  *http.Server
}

// Start starts the test server
func (s *TestServer) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:    s.address,
		Handler: s.router,
	}
	return s.server.ListenAndServe()
}

// SetupTestServer creates a test server with default configuration
func SetupTestServer() *Server {
	config := &Config{
		Host:             "localhost",
		Port:             8080,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      120 * time.Second,
		MaxHeaderBytes:   1 << 20,
		CORSEnabled:      true,
		AuthEnabled:      false,
		RateLimitEnabled: false,
		LoggingEnabled:   false,
	}

	// Create minimal services for testing
	services := &Services{}

	return NewServer(config, services)
}