package notifications

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/catherinevee/driftmgr/internal/notifications"
)

// Server represents the notification handler server
type Server struct {
	notifier *notifications.Notifier
	mu       sync.RWMutex
}

// NewServer creates a new notification handler server
func NewServer(notifier *notifications.Notifier) *Server {
	return &Server{
		notifier: notifier,
	}
}

// respondJSON sends a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}