package api

import (
	"net/http"
	"strings"
)

// Router methods

// GET registers a GET route
func (r *Router) GET(path string, handler http.HandlerFunc) {
	r.registerRoute("GET", path, handler)
}

// POST registers a POST route
func (r *Router) POST(path string, handler http.HandlerFunc) {
	r.registerRoute("POST", path, handler)
}

// PUT registers a PUT route
func (r *Router) PUT(path string, handler http.HandlerFunc) {
	r.registerRoute("PUT", path, handler)
}

// DELETE registers a DELETE route
func (r *Router) DELETE(path string, handler http.HandlerFunc) {
	r.registerRoute("DELETE", path, handler)
}

// registerRoute registers a route
func (r *Router) registerRoute(method, path string, handler http.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.routes[method] == nil {
		r.routes[method] = make(map[string]http.HandlerFunc)
	}
	r.routes[method][path] = handler
}

// ServeHTTP implements http.Handler for Router
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	methodRoutes, exists := r.routes[req.Method]
	r.mu.RUnlock()

	if !exists {
		http.NotFound(w, req)
		return
	}

	// Find matching route
	for pattern, handler := range methodRoutes {
		if r.matchRoute(pattern, req.URL.Path) {
			handler(w, req)
			return
		}
	}

	http.NotFound(w, req)
}

// matchRoute matches a route pattern with a path
func (r *Router) matchRoute(pattern, path string) bool {
	// Simple pattern matching - in a real system, you'd use a proper router
	if pattern == path {
		return true
	}

	// Handle wildcard patterns (e.g., /js/*)
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/")
	}

	// Handle path parameters (e.g., /api/v1/resources/{id})
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			// This is a path parameter, continue
			continue
		}
		if patternPart != pathParts[i] {
			return false
		}
	}

	return true
}
