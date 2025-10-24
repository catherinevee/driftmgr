package discovery

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all discovery-related routes
func RegisterRoutes(router *mux.Router, handler *Handler) {
	// Discovery job routes
	discoveryRouter := router.PathPrefix("/api/v1/discovery").Subrouter()

	// Discovery jobs
	discoveryRouter.HandleFunc("/jobs", handler.CreateDiscoveryJob).Methods("POST")
	discoveryRouter.HandleFunc("/jobs", handler.ListDiscoveryJobs).Methods("GET")
	discoveryRouter.HandleFunc("/jobs/{id}", handler.GetDiscoveryJob).Methods("GET")
	discoveryRouter.HandleFunc("/jobs/{id}", handler.UpdateDiscoveryJob).Methods("PUT")
	discoveryRouter.HandleFunc("/jobs/{id}", handler.DeleteDiscoveryJob).Methods("DELETE")
	discoveryRouter.HandleFunc("/jobs/{id}/start", handler.StartDiscoveryJob).Methods("POST")
	discoveryRouter.HandleFunc("/jobs/{id}/stop", handler.StopDiscoveryJob).Methods("POST")
	discoveryRouter.HandleFunc("/jobs/{id}/results", handler.GetDiscoveryResults).Methods("GET")

	// Resource routes
	resourceRouter := router.PathPrefix("/api/v1/resources").Subrouter()

	resourceRouter.HandleFunc("", handler.ListResources).Methods("GET")
	resourceRouter.HandleFunc("/{id}", handler.GetResource).Methods("GET")
	resourceRouter.HandleFunc("/search", handler.SearchResources).Methods("GET")
	resourceRouter.HandleFunc("/{id}/relationships", handler.GetResourceRelationships).Methods("GET")
	resourceRouter.HandleFunc("/{id}/tags", handler.UpdateResourceTags).Methods("PUT")
	resourceRouter.HandleFunc("/{id}/compliance", handler.GetResourceCompliance).Methods("GET")
	resourceRouter.HandleFunc("/{id}/cost", handler.GetResourceCost).Methods("GET")

	// Statistics routes
	statsRouter := router.PathPrefix("/api/v1/statistics").Subrouter()

	statsRouter.HandleFunc("/discovery", handler.GetDiscoveryStatistics).Methods("GET")
	statsRouter.HandleFunc("/resources", handler.GetResourceStatistics).Methods("GET")

	// Health check route
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
}
