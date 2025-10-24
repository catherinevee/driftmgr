package compliance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplianceAPI_ListPolicies(t *testing.T) {
	// Setup
	router, complianceService := setupTestRouter(t)

	// Create test policies
	createTestPolicies(t, complianceService)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/compliance/policies", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response []api.PolicyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Greater(t, len(response), 0)
}

func TestComplianceAPI_CreatePolicy(t *testing.T) {
	// Setup
	router, _ := setupTestRouter(t)

	tests := []struct {
		name           string
		request        api.CreatePolicyRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid policy creation",
			request: api.CreatePolicyRequest{
				ID:          "test-policy-1",
				Name:        "Test Policy",
				Description: "A test policy",
				Package:     "test.policy",
				Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}`,
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "missing required fields",
			request: api.CreatePolicyRequest{
				Name: "Test Policy",
				// Missing ID, Package, and Rules
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid policy syntax",
			request: api.CreatePolicyRequest{
				ID:      "test-policy-2",
				Name:    "Test Policy",
				Package: "test.policy",
				Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
    // Missing closing brace
}`,
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Test request
			req := httptest.NewRequest("POST", "/api/v1/compliance/policies", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response api.PolicyResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.request.ID, response.ID)
				assert.Equal(t, tt.request.Name, response.Name)
				assert.Equal(t, tt.request.Package, response.Package)
				assert.NotZero(t, response.CreatedAt)
			}
		})
	}
}

func TestComplianceAPI_GetPolicy(t *testing.T) {
	// Setup
	router, complianceService := setupTestRouter(t)

	// Create a test policy
	policy := &compliance.Policy{
		ID:          "test-policy",
		Name:        "Test Policy",
		Description: "A test policy",
		Package:     "test.policy",
		Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}`,
	}

	ctx := context.Background()
	err := complianceService.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	tests := []struct {
		name           string
		policyID       string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "existing policy",
			policyID:       "test-policy",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "non-existing policy",
			policyID:       "non-existing",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "empty policy ID",
			policyID:       "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test request
			url := fmt.Sprintf("/api/v1/compliance/policies?id=%s", tt.policyID)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response api.PolicyResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.policyID, response.ID)
				assert.Equal(t, policy.Name, response.Name)
				assert.Equal(t, policy.Package, response.Package)
			}
		})
	}
}

func TestComplianceAPI_EvaluatePolicy(t *testing.T) {
	// Setup
	router, complianceService := setupTestRouter(t)

	// Create a test policy
	policy := &compliance.Policy{
		ID:          "test-policy",
		Name:        "Test Policy",
		Description: "A test policy",
		Package:     "test.policy",
		Rules: `package test.policy

default allow = false

allow {
    input.action == "read"
}

violations["missing_owner"] {
    not input.tags.Owner
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Owner",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add Owner tag to the resource"
    }
}`,
	}

	ctx := context.Background()
	err := complianceService.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	tests := []struct {
		name           string
		request        api.EvaluatePolicyRequest
		expectedStatus int
		expectedAllow  bool
		expectError    bool
	}{
		{
			name: "allowed action with required tags",
			request: api.EvaluatePolicyRequest{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "read",
				Tags: map[string]string{
					"Owner": "test-user",
				},
			},
			expectedStatus: http.StatusOK,
			expectedAllow:  true,
			expectError:    false,
		},
		{
			name: "denied action with violation",
			request: api.EvaluatePolicyRequest{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "write",
				Tags:   map[string]string{},
			},
			expectedStatus: http.StatusOK,
			expectedAllow:  false,
			expectError:    false,
		},
		{
			name:    "missing required fields",
			request: api.EvaluatePolicyRequest{
				// Missing Resource and Action
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Test request
			url := fmt.Sprintf("/api/v1/compliance/policies/evaluate?id=%s", "test-policy")
			req := httptest.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response api.PolicyEvaluationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedAllow, response.Allow)
				assert.NotZero(t, response.EvaluatedAt)
			}
		})
	}
}

func TestComplianceAPI_GenerateReport(t *testing.T) {
	// Setup
	router, _ := setupTestRouter(t)

	tests := []struct {
		name           string
		request        api.GenerateReportRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid SOC2 report generation",
			request: api.GenerateReportRequest{
				Type: compliance.ComplianceSOC2,
				Period: struct {
					Start time.Time `json:"start" validate:"required"`
					End   time.Time `json:"end" validate:"required"`
				}{
					Start: time.Now().AddDate(0, -1, 0),
					End:   time.Now(),
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "valid HIPAA report generation",
			request: api.GenerateReportRequest{
				Type: compliance.ComplianceHIPAA,
				Period: struct {
					Start time.Time `json:"start" validate:"required"`
					End   time.Time `json:"end" validate:"required"`
				}{
					Start: time.Now().AddDate(0, -1, 0),
					End:   time.Now(),
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:    "missing required fields",
			request: api.GenerateReportRequest{
				// Missing Type and Period
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Test request
			req := httptest.NewRequest("POST", "/api/v1/compliance/reports", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var response compliance.ComplianceReport
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.request.Type, response.Type)
				assert.NotEmpty(t, response.ID)
				assert.NotEmpty(t, response.Title)
				assert.NotZero(t, response.GeneratedAt)
			}
		})
	}
}

func TestComplianceAPI_ExportReport(t *testing.T) {
	// Setup
	router, _ := setupTestRouter(t)

	tests := []struct {
		name           string
		reportID       string
		format         string
		expectedStatus int
		expectedType   string
		expectError    bool
	}{
		{
			name:           "export JSON report",
			reportID:       "test-report",
			format:         "json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			expectError:    false,
		},
		{
			name:           "export PDF report",
			reportID:       "test-report",
			format:         "pdf",
			expectedStatus: http.StatusOK,
			expectedType:   "application/pdf",
			expectError:    false,
		},
		{
			name:           "export HTML report",
			reportID:       "test-report",
			format:         "html",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			expectError:    false,
		},
		{
			name:           "export YAML report",
			reportID:       "test-report",
			format:         "yaml",
			expectedStatus: http.StatusOK,
			expectedType:   "application/x-yaml",
			expectError:    false,
		},
		{
			name:           "missing report ID",
			reportID:       "",
			format:         "json",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test request
			url := fmt.Sprintf("/api/v1/compliance/reports/export?id=%s&format=%s", tt.reportID, tt.format)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				assert.Equal(t, tt.expectedType, w.Header().Get("Content-Type"))
				assert.NotEmpty(t, w.Body.Bytes())
			}
		})
	}
}

// Helper functions

func setupTestRouter(t *testing.T) (*gin.Engine, *compliance.ComplianceService) {
	// Create mock OPA engine
	config := compliance.OPAConfig{
		LocalPolicies: "../../../policies",
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}
	policyEngine := compliance.NewOPAEngine(config)

	// Create mock compliance reporter
	dataSource := &mockDataSource{}
	reporter := compliance.NewComplianceReporter(dataSource, policyEngine)

	// Create compliance service
	complianceService := compliance.NewComplianceService(policyEngine, reporter)

	// Create compliance handlers
	complianceHandlers := api.NewComplianceHandlers(complianceService)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup compliance routes
	api.SetupComplianceRoutes(router, complianceHandlers)

	return router, complianceService
}

func createTestPolicies(t *testing.T, service *compliance.ComplianceService) {
	ctx := context.Background()

	policies := []*compliance.Policy{
		{
			ID:          "policy-1",
			Name:        "AWS Security Policy",
			Description: "AWS security compliance policy",
			Package:     "aws.security",
			Rules: `package aws.security

default allow = false

allow {
    input.action == "read"
}`,
		},
		{
			ID:          "policy-2",
			Name:        "HIPAA Compliance Policy",
			Description: "HIPAA compliance policy",
			Package:     "hipaa.compliance",
			Rules: `package hipaa.compliance

default allow = false

allow {
    input.action == "read"
}`,
		},
	}

	for _, policy := range policies {
		err := service.CreatePolicy(ctx, policy)
		require.NoError(t, err)
	}
}

// Mock data source for testing
type mockDataSource struct{}

func (m *mockDataSource) GetDriftResults(ctx context.Context) ([]*detector.DriftResult, error) {
	return []*detector.DriftResult{}, nil
}

func (m *mockDataSource) GetPolicyViolations(ctx context.Context) ([]compliance.PolicyViolation, error) {
	return []compliance.PolicyViolation{}, nil
}

func (m *mockDataSource) GetResourceInventory(ctx context.Context) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *mockDataSource) GetAuditLogs(ctx context.Context, since time.Time) ([]interface{}, error) {
	return []interface{}{}, nil
}
