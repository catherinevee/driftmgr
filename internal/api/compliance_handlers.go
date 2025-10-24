package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/go-playground/validator/v10"
)

// ComplianceHandlers handles compliance API endpoints
type ComplianceHandlers struct {
	complianceService *compliance.ComplianceService
	validator         *validator.Validate
}

// NewComplianceHandlers creates a new set of compliance handlers
func NewComplianceHandlers(complianceService *compliance.ComplianceService) *ComplianceHandlers {
	return &ComplianceHandlers{
		complianceService: complianceService,
		validator:         validator.New(),
	}
}

// ListPolicies handles GET /api/v1/compliance/policies
func (h *ComplianceHandlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies := h.complianceService.ListPolicies(r.Context())

	// Convert to response format
	policyResponses := make([]PolicyResponse, len(policies))
	for i, policy := range policies {
		policyResponses[i] = PolicyResponse{
			ID:          policy.ID,
			Name:        policy.Name,
			Description: policy.Description,
			Package:     policy.Package,
			CreatedAt:   policy.CreatedAt,
			UpdatedAt:   policy.UpdatedAt,
		}
	}

	writeJSONResponse(w, http.StatusOK, policyResponses, nil)
}

// GetPolicy handles GET /api/v1/compliance/policies/{id}
func (h *ComplianceHandlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from URL
	policyID := r.URL.Query().Get("id")
	if policyID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_POLICY_ID", "Policy ID is required", "")
		return
	}

	policy, exists := h.complianceService.GetPolicy(r.Context(), policyID)
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "POLICY_NOT_FOUND", "Policy not found", "")
		return
	}

	response := PolicyResponse{
		ID:          policy.ID,
		Name:        policy.Name,
		Description: policy.Description,
		Package:     policy.Package,
		Rules:       policy.Rules,
		Metadata:    policy.Metadata,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}

	writeJSONResponse(w, http.StatusOK, response, nil)
}

// CreatePolicy handles POST /api/v1/compliance/policies
func (h *ComplianceHandlers) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Create policy
	policy := &compliance.Policy{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Package:     req.Package,
		Rules:       req.Rules,
		Metadata:    req.Metadata,
	}

	if err := h.complianceService.CreatePolicy(r.Context(), policy); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "CREATE_POLICY_FAILED", "Failed to create policy", err.Error())
		return
	}

	response := PolicyResponse{
		ID:          policy.ID,
		Name:        policy.Name,
		Description: policy.Description,
		Package:     policy.Package,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}

	writeJSONResponse(w, http.StatusCreated, response, nil)
}

// UpdatePolicy handles PUT /api/v1/compliance/policies/{id}
func (h *ComplianceHandlers) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from URL
	policyID := r.URL.Query().Get("id")
	if policyID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_POLICY_ID", "Policy ID is required", "")
		return
	}

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Update policy
	policy := &compliance.Policy{
		ID:          policyID,
		Name:        req.Name,
		Description: req.Description,
		Package:     req.Package,
		Rules:       req.Rules,
		Metadata:    req.Metadata,
	}

	if err := h.complianceService.UpdatePolicy(r.Context(), policy); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "UPDATE_POLICY_FAILED", "Failed to update policy", err.Error())
		return
	}

	response := PolicyResponse{
		ID:          policy.ID,
		Name:        policy.Name,
		Description: policy.Description,
		Package:     policy.Package,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}

	writeJSONResponse(w, http.StatusOK, response, nil)
}

// DeletePolicy handles DELETE /api/v1/compliance/policies/{id}
func (h *ComplianceHandlers) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from URL
	policyID := r.URL.Query().Get("id")
	if policyID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_POLICY_ID", "Policy ID is required", "")
		return
	}

	if err := h.complianceService.DeletePolicy(r.Context(), policyID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "DELETE_POLICY_FAILED", "Failed to delete policy", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Policy deleted successfully"}, nil)
}

// EvaluatePolicy handles POST /api/v1/compliance/policies/{id}/evaluate
func (h *ComplianceHandlers) EvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	// Extract policy ID from URL
	policyID := r.URL.Query().Get("id")
	if policyID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_POLICY_ID", "Policy ID is required", "")
		return
	}

	var req EvaluatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Get policy to determine package name
	policy, exists := h.complianceService.GetPolicy(r.Context(), policyID)
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "POLICY_NOT_FOUND", "Policy not found", "")
		return
	}

	// Create policy input
	input := compliance.PolicyInput{
		Resource:  req.Resource,
		Action:    req.Action,
		Principal: req.Principal,
		Context:   req.Context,
		Provider:  req.Provider,
		Region:    req.Region,
		Tags:      req.Tags,
	}

	// Evaluate policy
	decision, err := h.complianceService.EvaluatePolicy(r.Context(), policy.Package, input)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "EVALUATION_FAILED", "Failed to evaluate policy", err.Error())
		return
	}

	// Convert to response format
	response := PolicyEvaluationResponse{
		Allow:       decision.Allow,
		Reasons:     decision.Reasons,
		Violations:  convertViolations(decision.Violations),
		Suggestions: decision.Suggestions,
		EvaluatedAt: decision.EvaluatedAt,
	}

	writeJSONResponse(w, http.StatusOK, response, nil)
}

// GenerateReport handles POST /api/v1/compliance/reports
func (h *ComplianceHandlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Create report period
	period := compliance.ReportPeriod{
		Start: req.Period.Start,
		End:   req.Period.End,
	}

	// Generate report
	report, err := h.complianceService.GenerateComplianceReport(r.Context(), req.Type, period)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "REPORT_GENERATION_FAILED", "Failed to generate report", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, report, nil)
}

// ExportReport handles GET /api/v1/compliance/reports/{id}/export
func (h *ComplianceHandlers) ExportReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL
	reportID := r.URL.Query().Get("id")
	if reportID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_REPORT_ID", "Report ID is required", "")
		return
	}

	// Get format from query parameter
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json" // Default format
	}

	// For now, we'll create a mock report since we don't have report storage implemented
	// In a real implementation, this would retrieve the report from storage
	report := &compliance.ComplianceReport{
		ID:          reportID,
		Type:        compliance.ComplianceSOC2,
		Title:       "SOC 2 Compliance Report",
		GeneratedAt: time.Now(),
		Period: compliance.ReportPeriod{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Summary: compliance.ReportSummary{
			TotalControls:    10,
			PassedControls:   8,
			FailedControls:   2,
			ComplianceScore:  80.0,
			CriticalFindings: 1,
			HighFindings:     2,
			MediumFindings:   3,
			LowFindings:      1,
		},
	}

	// Export report
	data, err := h.complianceService.ExportReport(r.Context(), report, format)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "EXPORT_FAILED", "Failed to export report", err.Error())
		return
	}

	// Set appropriate content type
	contentType := "application/json"
	switch format {
	case "pdf":
		contentType = "application/pdf"
	case "html":
		contentType = "text/html"
	case "yaml":
		contentType = "application/x-yaml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report-%s.%s\"", reportID, format))
	w.Write(data)
}

// Helper functions

func convertViolations(violations []compliance.PolicyViolation) []PolicyViolationResponse {
	responses := make([]PolicyViolationResponse, len(violations))
	for i, violation := range violations {
		responses[i] = PolicyViolationResponse{
			Rule:        violation.Rule,
			Message:     violation.Message,
			Severity:    violation.Severity,
			Resource:    violation.Resource,
			Details:     violation.Details,
			Remediation: violation.Remediation,
		}
	}
	return responses
}

// Request/Response types

type CreatePolicyRequest struct {
	ID          string                 `json:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description"`
	Package     string                 `json:"package" validate:"required"`
	Rules       string                 `json:"rules" validate:"required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdatePolicyRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description"`
	Package     string                 `json:"package" validate:"required"`
	Rules       string                 `json:"rules" validate:"required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type EvaluatePolicyRequest struct {
	Resource  interface{}            `json:"resource" validate:"required"`
	Action    string                 `json:"action" validate:"required"`
	Principal string                 `json:"principal,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Region    string                 `json:"region,omitempty"`
	Tags      map[string]string      `json:"tags,omitempty"`
}

type GenerateReportRequest struct {
	Type   compliance.ComplianceType `json:"type" validate:"required"`
	Period struct {
		Start time.Time `json:"start" validate:"required"`
		End   time.Time `json:"end" validate:"required"`
	} `json:"period" validate:"required"`
}

type PolicyResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Package     string                 `json:"package"`
	Rules       string                 `json:"rules,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type PolicyEvaluationResponse struct {
	Allow       bool                      `json:"allow"`
	Reasons     []string                  `json:"reasons,omitempty"`
	Violations  []PolicyViolationResponse `json:"violations,omitempty"`
	Suggestions []string                  `json:"suggestions,omitempty"`
	EvaluatedAt time.Time                 `json:"evaluated_at"`
}

type PolicyViolationResponse struct {
	Rule        string                 `json:"rule"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Resource    string                 `json:"resource,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
}
