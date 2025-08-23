package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/utils/graceful"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	router *mux.Router
	port   string
}

// HandleServer handles the server command
func HandleServer(args []string) {
	var port string = "8080"

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr server [flags]")
			fmt.Println()
			fmt.Println("Start DriftMgr in server mode with REST API")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --port, -p string    Port to run server on (default: 8080)")
			fmt.Println()
			fmt.Println("API Endpoints:")
			fmt.Println("  GET  /health         Health check")
			fmt.Println("  POST /discover       Discover resources")
			fmt.Println("  POST /drift/detect   Detect drift")
			fmt.Println("  GET  /state          List state files")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr server")
			fmt.Println("  driftmgr server --port 9000")
			return
		}
	}

	server := NewServer(port)
	server.setupRoutes()

	fmt.Printf("Starting DriftMgr Server on port %s\n", port)
	fmt.Printf("API available at http://localhost:%s\n", port)
	fmt.Println("\nPress Ctrl+C to stop the server")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: server.router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			graceful.HandleError(err, "Failed to start server")
		}
	}()

	<-sigChan
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		graceful.HandleError(err, "Server shutdown failed")
	}

	fmt.Println("Server stopped")
}

func NewServer(port string) *Server {
	if port == "" {
		port = "8080"
	}

	return &Server{
		router: mux.NewRouter(),
		port:   port,
	}
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Discovery endpoints
	s.router.HandleFunc("/discover", s.discoverHandler).Methods("POST")

	// Drift detection endpoints
	s.router.HandleFunc("/drift/detect", s.driftDetectHandler).Methods("POST")

	// State management endpoints
	s.router.HandleFunc("/state", s.stateListHandler).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	s.router.Use(c.Handler)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy","service":"driftmgr-server"}`)
}

func (s *Server) discoverHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"Discovery endpoint - implementation pending"}`)
}

func (s *Server) driftDetectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"Drift detection endpoint - implementation pending"}`)
}

func (s *Server) stateListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"State list endpoint - implementation pending"}`)
}
