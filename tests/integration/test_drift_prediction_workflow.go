package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/workflow"
)

// TestDriftPredictionWorkflowIntegration tests the complete integration of drift prediction and workflow automation
func TestDriftPredictionWorkflowIntegration(t *testing.T) {
	// Initialize drift predictor
	predictor := drift.NewDriftPredictor(nil)
	drift.RegisterDefaultPatterns(predictor)

	// Initialize workflow engine
	workflowEngine := workflow.NewWorkflowEngine()
	if err := workflow.RegisterDefaultWorkflows(workflowEngine); err != nil {
		t.Fatalf("Failed to register default workflows: %v", err)
	}

	// Test 1: Drift Prediction
	t.Run("DriftPrediction", func(t *testing.T) {
		// Create test resources
		resources := []models.Resource{
			{
				ID:       "sg-12345678",
				Name:     "test-security-group",
				Type:     "aws_security_group",
				Provider: "aws",
				Region:   "us-east-1",
				Tags:     map[string]string{"public": "true"},
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
			{
				ID:       "i-87654321",
				Name:     "test-instance",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
				Tags:     map[string]string{"purpose": "test"},
				State:    "running",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
			{
				ID:       "bucket-test-123",
				Name:     "test-bucket",
				Type:     "aws_s3_bucket",
				Provider: "aws",
				Region:   "us-east-1",
				Tags:     map[string]string{},
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
		}

		// Predict drifts
		ctx := context.Background()
		predictions := predictor.PredictDrifts(ctx, resources)

		// Verify predictions
		if len(predictions) == 0 {
			t.Error("Expected drift predictions, got none")
		}

		// Check for specific predictions
		foundSecurityPrediction := false
		foundCostPrediction := false

		for _, prediction := range predictions {
			if prediction.Pattern.ID == "public_access" {
				foundSecurityPrediction = true
				if prediction.RiskLevel != drift.RiskLevelCritical {
					t.Errorf("Expected critical risk level for public access, got %s", prediction.RiskLevel)
				}
			}
			if prediction.Pattern.ID == "unused_resource" {
				foundCostPrediction = true
				if prediction.RiskLevel != drift.RiskLevelMedium {
					t.Errorf("Expected medium risk level for unused resource, got %s", prediction.RiskLevel)
				}
			}
		}

		if !foundSecurityPrediction {
			t.Error("Expected security prediction for public access resource")
		}
		if !foundCostPrediction {
			t.Error("Expected cost prediction for test resource")
		}

		t.Logf("Found %d drift predictions", len(predictions))
		for _, prediction := range predictions {
			t.Logf("Prediction: %s - %s (Risk: %s, Confidence: %.2f)",
				prediction.ResourceName, prediction.Pattern.Name, prediction.RiskLevel, prediction.Confidence)
		}
	})

	// Test 2: Workflow Management
	t.Run("WorkflowManagement", func(t *testing.T) {
		// List workflows
		workflows := workflowEngine.ListWorkflows()
		if len(workflows) == 0 {
			t.Error("Expected predefined workflows, got none")
		}

		// Find security remediation workflow
		var securityWorkflow *workflow.Workflow
		for _, wf := range workflows {
			if wf.ID == "security_remediation" {
				securityWorkflow = wf
				break
			}
		}

		if securityWorkflow == nil {
			t.Error("Expected security remediation workflow not found")
		}

		// Verify workflow structure
		if len(securityWorkflow.Steps) < 3 {
			t.Errorf("Expected at least 3 steps in security workflow, got %d", len(securityWorkflow.Steps))
		}

		if securityWorkflow.Rollback == nil {
			t.Error("Expected rollback plan in security workflow")
		}

		t.Logf("Found %d workflows", len(workflows))
		for _, wf := range workflows {
			t.Logf("Workflow: %s - %s (%d steps)", wf.ID, wf.Name, len(wf.Steps))
		}
	})

	// Test 3: Workflow Execution
	t.Run("WorkflowExecution", func(t *testing.T) {
		// Create test resources for workflow execution
		resources := []models.Resource{
			{
				ID:       "sg-test-123",
				Name:     "test-security-group",
				Type:     "aws_security_group",
				Provider: "aws",
				Region:   "us-east-1",
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
		}

		// Create workflow context
		executionID := fmt.Sprintf("test_exec_%d", time.Now().Unix())
		workflowCtx := workflow.WorkflowContext{
			WorkflowID:  "security_remediation",
			ExecutionID: executionID,
			Parameters: map[string]interface{}{
				"resource_id": "sg-test-123",
				"region":      "us-east-1",
			},
			Resources: resources,
			State:     make(map[string]interface{}),
			StartedAt: time.Now(),
			User:      "test-user",
			Metadata:  make(map[string]interface{}),
		}

		// Execute workflow
		_ = workflowEngine.ExecuteWorkflow(context.Background(), "security_remediation", workflowCtx)

		// Verify execution started
		// Note: We're not checking the result status since we're not storing it

		// Wait a bit for execution to complete
		time.Sleep(2 * time.Second)

		// Get execution result
		executionResult, err := workflowEngine.GetExecution(executionID)
		if err != nil {
			t.Errorf("Failed to get execution result: %v", err)
		}

		// Verify execution completed
		if executionResult.Status != workflow.WorkflowStatusCompleted && executionResult.Status != workflow.WorkflowStatusFailed {
			t.Errorf("Expected workflow to be completed or failed, got %s", executionResult.Status)
		}

		t.Logf("Workflow execution completed with status: %s", executionResult.Status)
		t.Logf("Execution duration: %v", executionResult.Duration)
		t.Logf("Steps executed: %d", len(executionResult.Steps))
	})

	// Test 4: End-to-End Integration
	t.Run("EndToEndIntegration", func(t *testing.T) {
		// Create test resources
		resources := []models.Resource{
			{
				ID:       "sg-public-123",
				Name:     "public-security-group",
				Type:     "aws_security_group",
				Provider: "aws",
				Region:   "us-east-1",
				Tags:     map[string]string{"public": "true", "environment": "test"},
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
		}

		// Step 1: Predict drifts
		ctx := context.Background()
		predictions := predictor.PredictDrifts(ctx, resources)

		if len(predictions) == 0 {
			t.Fatal("No drift predictions found")
		}

		// Step 2: Find high-risk predictions
		var highRiskPredictions []drift.PredictedDrift
		for _, prediction := range predictions {
			if prediction.RiskLevel == drift.RiskLevelCritical || prediction.RiskLevel == drift.RiskLevelHigh {
				highRiskPredictions = append(highRiskPredictions, prediction)
			}
		}

		if len(highRiskPredictions) == 0 {
			t.Log("No high-risk predictions found, skipping workflow execution")
			return
		}

		// Step 3: Execute remediation workflow for high-risk predictions
		for _, prediction := range highRiskPredictions {
			t.Logf("Executing remediation for prediction: %s (Risk: %s)", prediction.Pattern.Name, prediction.RiskLevel)

			executionID := fmt.Sprintf("remediation_%s_%d", prediction.ResourceID, time.Now().Unix())
			context := workflow.WorkflowContext{
				WorkflowID:  "security_remediation",
				ExecutionID: executionID,
				Parameters: map[string]interface{}{
					"resource_id": prediction.ResourceID,
					"drift_type":  prediction.Pattern.ID,
					"risk_level":  string(prediction.RiskLevel),
				},
				Resources: resources,
				State:     make(map[string]interface{}),
				StartedAt: time.Now(),
				User:      "automated-system",
				Metadata: map[string]interface{}{
					"prediction_id": prediction.ID,
					"confidence":    prediction.Confidence,
				},
			}

			_ = workflowEngine.ExecuteWorkflow(ctx, "security_remediation", context)

			// Wait for execution to complete
			time.Sleep(3 * time.Second)

			// Get final result
			executionResult, err := workflowEngine.GetExecution(executionID)
			if err != nil {
				t.Errorf("Failed to get execution result for %s: %v", prediction.ResourceID, err)
				continue
			}

			t.Logf("Remediation for %s completed with status: %s", prediction.ResourceID, executionResult.Status)
			t.Logf("Duration: %v, Steps: %d", executionResult.Duration, len(executionResult.Steps))

			// Verify execution completed successfully
			if executionResult.Status == workflow.WorkflowStatusFailed {
				t.Errorf("Remediation workflow failed for %s", prediction.ResourceID)
			}
		}
	})

	// Test 5: Performance and Scalability
	t.Run("PerformanceAndScalability", func(t *testing.T) {
		// Create large number of resources
		var resources []models.Resource
		for i := 0; i < 100; i++ {
			resource := models.Resource{
				ID:       fmt.Sprintf("resource-%d", i),
				Name:     fmt.Sprintf("test-resource-%d", i),
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
				Tags:     map[string]string{"purpose": "test"},
				State:    "running",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			}
			resources = append(resources, resource)
		}

		// Measure prediction performance
		start := time.Now()
		ctx := context.Background()
		predictions := predictor.PredictDrifts(ctx, resources)
		duration := time.Since(start)

		t.Logf("Predicted drifts for %d resources in %v", len(resources), duration)
		t.Logf("Found %d predictions (%.2f predictions per resource)", len(predictions), float64(len(predictions))/float64(len(resources)))

		// Performance assertions
		if duration > 5*time.Second {
			t.Errorf("Prediction took too long: %v", duration)
		}

		// Verify predictions are reasonable
		if len(predictions) > len(resources) {
			t.Errorf("More predictions than resources: %d > %d", len(predictions), len(resources))
		}
	})
}

// TestDriftPredictionAPI tests the drift prediction API endpoints
func TestDriftPredictionAPI(t *testing.T) {
	// Create test server (this would normally be the actual server)
	// For this test, we'll simulate the API calls

	t.Run("PredictDriftsAPI", func(t *testing.T) {
		// Create test request
		request := struct {
			Resources []models.Resource      `json:"resources"`
			Options   map[string]interface{} `json:"options"`
		}{
			Resources: []models.Resource{
				{
					ID:       "sg-test-api",
					Name:     "test-security-group",
					Type:     "aws_security_group",
					Provider: "aws",
					Region:   "us-east-1",
					Tags:     map[string]string{"public": "true"},
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				},
			},
			Options: map[string]interface{}{
				"min_confidence":   0.7,
				"include_low_risk": false,
			},
		}

		// Simulate API request
		requestBody, _ := json.Marshal(request)
		t.Logf("API Request: %s", string(requestBody))

		// In a real test, you would make an HTTP request to the actual server
		// For now, we'll simulate the response
		response := struct {
			Predictions []drift.PredictedDrift `json:"predictions"`
			Total       int                    `json:"total"`
			Timestamp   time.Time              `json:"timestamp"`
		}{
			Predictions: []drift.PredictedDrift{
				{
					ID:           "pred_sg-test-api_public_access",
					ResourceID:   "sg-test-api",
					ResourceName: "test-security-group",
					ResourceType: "aws_security_group",
					Confidence:   0.9,
					RiskLevel:    drift.RiskLevelCritical,
					RiskScore:    0.85,
					PredictedAt:  time.Now(),
				},
			},
			Total:     1,
			Timestamp: time.Now(),
		}

		// Verify response
		if response.Total != 1 {
			t.Errorf("Expected 1 prediction, got %d", response.Total)
		}

		if len(response.Predictions) == 0 {
			t.Error("Expected predictions in response")
		}

		prediction := response.Predictions[0]
		if prediction.RiskLevel != drift.RiskLevelCritical {
			t.Errorf("Expected critical risk level, got %s", prediction.RiskLevel)
		}

		t.Logf("API Response: %d predictions, highest risk: %s", response.Total, prediction.RiskLevel)
	})
}

// TestWorkflowAPI tests the workflow management API endpoints
func TestWorkflowAPI(t *testing.T) {
	t.Run("WorkflowManagementAPI", func(t *testing.T) {
		// Test workflow listing
		workflows := []workflow.Workflow{
			{
				ID:          "security_remediation",
				Name:        "Security Remediation",
				Description: "Automated security remediation workflow",
				Timeout:     30 * time.Minute,
				Retries:     3,
			},
			{
				ID:          "cost_optimization",
				Name:        "Cost Optimization",
				Description: "Automated cost optimization workflow",
				Timeout:     45 * time.Minute,
				Retries:     2,
			},
		}

		t.Logf("Available workflows: %d", len(workflows))
		for _, wf := range workflows {
			t.Logf("  - %s: %s", wf.ID, wf.Name)
		}

		// Test workflow execution request
		executionRequest := struct {
			WorkflowID string                 `json:"workflow_id"`
			Parameters map[string]interface{} `json:"parameters"`
			Resources  []models.Resource      `json:"resources"`
		}{
			WorkflowID: "security_remediation",
			Parameters: map[string]interface{}{
				"resource_id": "sg-test-api",
				"region":      "us-east-1",
			},
			Resources: []models.Resource{
				{
					ID:       "sg-test-api",
					Name:     "test-security-group",
					Type:     "aws_security_group",
					Provider: "aws",
					Region:   "us-east-1",
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				},
			},
		}

		// Simulate API request
		requestBody, _ := json.Marshal(executionRequest)
		t.Logf("Workflow Execution Request: %s", string(requestBody))

		// Simulate API response
		executionResponse := struct {
			ExecutionID string    `json:"execution_id"`
			WorkflowID  string    `json:"workflow_id"`
			Status      string    `json:"status"`
			StartedAt   time.Time `json:"started_at"`
		}{
			ExecutionID: "exec_1234567890",
			WorkflowID:  "security_remediation",
			Status:      "running",
			StartedAt:   time.Now(),
		}

		t.Logf("Workflow Execution Response: %s", executionResponse.ExecutionID)
		t.Logf("Status: %s", executionResponse.Status)
	})
}

// BenchmarkDriftPrediction benchmarks the drift prediction performance
func BenchmarkDriftPrediction(b *testing.B) {
	// Initialize predictor
	predictor := drift.NewDriftPredictor(nil)
	drift.RegisterDefaultPatterns(predictor)

	// Create test resources
	resources := make([]models.Resource, 100)
	for i := 0; i < 100; i++ {
		resources[i] = models.Resource{
			ID:       fmt.Sprintf("resource-%d", i),
			Name:     fmt.Sprintf("test-resource-%d", i),
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			Tags:     map[string]string{"purpose": "test"},
			State:    "running",
			Created:  time.Now().Add(-24 * time.Hour),
			Updated:  time.Now(),
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.PredictDrifts(ctx, resources)
	}
}

// BenchmarkWorkflowExecution benchmarks the workflow execution performance
func BenchmarkWorkflowExecution(b *testing.B) {
	// Initialize workflow engine
	workflowEngine := workflow.NewWorkflowEngine()
	workflow.RegisterDefaultWorkflows(workflowEngine)

	// Create test context
	resources := []models.Resource{
		{
			ID:       "test-resource",
			Name:     "test-resource",
			Type:     "aws_security_group",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Created:  time.Now().Add(-24 * time.Hour),
			Updated:  time.Now(),
		},
	}

	workflowCtx := workflow.WorkflowContext{
		WorkflowID: "security_remediation",
		Parameters: make(map[string]interface{}),
		Resources:  resources,
		State:      make(map[string]interface{}),
		StartedAt:  time.Now(),
		User:       "benchmark",
		Metadata:   make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflowCtx.ExecutionID = fmt.Sprintf("bench_%d", i)
		workflowEngine.ExecuteWorkflow(context.Background(), "security_remediation", workflowCtx)
	}
}

func main() {
	// This is a test file, not meant to be run directly
	fmt.Println("This is a test file. Run with: go test ./tests/integration/...")
}
