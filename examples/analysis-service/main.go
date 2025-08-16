package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/analysis"
	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/monitoring"
	"github.com/gorilla/mux"
)

const (
	serviceName = "analysis-service"
	servicePort = "8082"
)

var (
	analyzer       *analysis.EnhancedAnalyzer
	driftPredictor *drift.DriftPredictor
	cacheManager   = cache.GetGlobalManager()
	logger         = monitoring.GetGlobalLogger()
)

func main() {
	// Initialize the analyzer and drift predictor
	analyzer = analysis.NewEnhancedAnalyzer()
	driftPredictor = drift.NewDriftPredictor()

	// Set up router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Analysis endpoints
	api.HandleFunc("/analyze", handleAnalyze).Methods("POST")
	api.HandleFunc("/analyze/enhanced", handleEnhancedAnalyze).Methods("POST")
	api.HandleFunc("/analysis/{id}", handleGetAnalysis).Methods("GET")
	api.HandleFunc("/analysis/{id}/details", handleGetAnalysisDetails).Methods("GET")

	// Drift prediction endpoints
	api.HandleFunc("/predict", handlePredictDrifts).Methods("POST")
	api.HandleFunc("/patterns", handleGetDriftPatterns).Methods("GET")
	api.HandleFunc("/patterns/{id}", handleGetDriftPattern).Methods("GET")
	api.HandleFunc("/prediction/stats", handleGetPredictionStats).Methods("GET")

	// Risk assessment endpoints
	api.HandleFunc("/risks", handleGetRisks).Methods("GET")
	api.HandleFunc("/risks/assess", handleAssessRisks).Methods("POST")
	api.HandleFunc("/risks/{id}", handleGetRisk).Methods("GET")

	// Historical analysis endpoints
	api.HandleFunc("/history", handleGetAnalysisHistory).Methods("GET")
	api.HandleFunc("/history/{id}", handleGetHistoricalAnalysis).Methods("GET")
	api.HandleFunc("/trends", handleGetTrends).Methods("GET")

	// Start server
	logger.Info("Starting analysis service on port " + servicePort)
	log.Fatal(http.ListenAndServe(":"+servicePort, router))
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":   serviceName,
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAnalyze handles basic drift analysis
func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider  string                 `json:"provider"`
		Region    string                 `json:"region"`
		StateFile string                 `json:"state_file,omitempty"`
		Resources []models.Resource      `json:"resources,omitempty"`
		Options   map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("analysis:%s:%s:%s", request.Provider, request.Region, request.StateFile)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Perform analysis
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	result, err := analyzer.AnalyzeDrift(ctx, request.Provider, request.Region, request.StateFile, request.Resources, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the results
	cacheManager.Set(cacheKey, result, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleEnhancedAnalyze handles enhanced drift analysis with ML
func handleEnhancedAnalyze(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider  string                 `json:"provider"`
		Region    string                 `json:"region"`
		StateFile string                 `json:"state_file,omitempty"`
		Resources []models.Resource      `json:"resources,omitempty"`
		Options   map[string]interface{} `json:"options,omitempty"`
		MLEnabled bool                   `json:"ml_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("enhanced_analysis:%s:%s:%s:%t", request.Provider, request.Region, request.StateFile, request.MLEnabled)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Perform enhanced analysis
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()

	result, err := analyzer.EnhancedAnalyzeDrift(ctx, request.Provider, request.Region, request.StateFile, request.Resources, request.Options, request.MLEnabled)
	if err != nil {
		http.Error(w, fmt.Sprintf("Enhanced analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the results
	cacheManager.Set(cacheKey, result, 30*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetAnalysis retrieves a specific analysis by ID
func handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	analysisID := vars["id"]

	// Get analysis from storage
	analysis, err := analyzer.GetAnalysis(analysisID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// handleGetAnalysisDetails retrieves detailed analysis information
func handleGetAnalysisDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	analysisID := vars["id"]

	// Get detailed analysis
	details, err := analyzer.GetAnalysisDetails(analysisID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis details not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

// handlePredictDrifts handles drift prediction using ML
func handlePredictDrifts(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider  string            `json:"provider"`
		Region    string            `json:"region"`
		Resources []models.Resource `json:"resources"`
		Timeframe string            `json:"timeframe"` // e.g., "24h", "7d", "30d"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform prediction
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	predictions, err := driftPredictor.PredictDrifts(ctx, request.Provider, request.Region, request.Resources, request.Timeframe)
	if err != nil {
		http.Error(w, fmt.Sprintf("Prediction failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(predictions)
}

// handleGetDriftPatterns retrieves known drift patterns
func handleGetDriftPatterns(w http.ResponseWriter, r *http.Request) {
	patterns := driftPredictor.GetDriftPatterns()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// handleGetDriftPattern retrieves a specific drift pattern
func handleGetDriftPattern(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patternID := vars["id"]

	pattern, err := driftPredictor.GetDriftPattern(patternID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Pattern not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pattern)
}

// handleGetPredictionStats retrieves prediction statistics
func handleGetPredictionStats(w http.ResponseWriter, r *http.Request) {
	stats := driftPredictor.GetPredictionStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleGetRisks retrieves risk assessments
func handleGetRisks(w http.ResponseWriter, r *http.Request) {
	risks := analyzer.GetRisks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risks)
}

// handleAssessRisks performs risk assessment
func handleAssessRisks(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider    string            `json:"provider"`
		Region      string            `json:"region"`
		Resources   []models.Resource `json:"resources"`
		RiskFactors []string          `json:"risk_factors"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform risk assessment
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	assessment, err := analyzer.AssessRisks(ctx, request.Provider, request.Region, request.Resources, request.RiskFactors)
	if err != nil {
		http.Error(w, fmt.Sprintf("Risk assessment failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assessment)
}

// handleGetRisk retrieves a specific risk assessment
func handleGetRisk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	riskID := vars["id"]

	risk, err := analyzer.GetRisk(riskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Risk not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risk)
}

// handleGetAnalysisHistory retrieves analysis history
func handleGetAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	provider := query.Get("provider")
	region := query.Get("region")
	limit := query.Get("limit")

	history, err := analyzer.GetAnalysisHistory(provider, region, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleGetHistoricalAnalysis retrieves a specific historical analysis
func handleGetHistoricalAnalysis(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	historyID := vars["id"]

	analysis, err := analyzer.GetHistoricalAnalysis(historyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Historical analysis not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// handleGetTrends retrieves trend analysis
func handleGetTrends(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	provider := query.Get("provider")
	region := query.Get("region")
	timeframe := query.Get("timeframe")

	trends, err := analyzer.GetTrends(provider, region, timeframe)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get trends: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}
