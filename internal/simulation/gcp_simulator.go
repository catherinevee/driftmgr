package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/state/parser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GCPSimulator simulates drift in GCP resources
type GCPSimulator struct {
	projectID   string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

// NewGCPSimulator creates a new GCP drift simulator
func NewGCPSimulator() *GCPSimulator {
	return &GCPSimulator{}
}

// Initialize sets up GCP authentication
func (s *GCPSimulator) Initialize(ctx context.Context) error {
	// Get project ID
	s.projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	if s.projectID == "" {
		s.projectID = os.Getenv("GCP_PROJECT")
	}
	if s.projectID == "" {
		return fmt.Errorf("no GCP project ID found")
	}

	// Get credentials
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/compute",
	)
	if err != nil {
		// Try service account key file
		if keyFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); keyFile != "" {
			data, err := ioutil.ReadFile(keyFile)
			if err != nil {
				return fmt.Errorf("failed to read service account key: %w", err)
			}

			creds, err = google.CredentialsFromJSON(ctx, data,
				"https://www.googleapis.com/auth/cloud-platform",
				"https://www.googleapis.com/auth/compute",
			)
			if err != nil {
				return fmt.Errorf("failed to create credentials: %w", err)
			}
		} else {
			return fmt.Errorf("failed to find GCP credentials: %w", err)
		}
	}

	s.tokenSource = creds.TokenSource
	s.httpClient = oauth2.NewClient(ctx, s.tokenSource)

	return nil
}

// SimulateDrift creates drift in GCP resources
func (s *GCPSimulator) SimulateDrift(ctx context.Context, driftType DriftType, resourceID string, state *parser.TerraformState) (*SimulationResult, error) {
	// Initialize if needed
	if s.httpClient == nil {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Find the resource in state
	resource := s.findResource(resourceID, state)
	if resource == nil {
		return nil, fmt.Errorf("resource %s not found in state", resourceID)
	}

	// Execute drift based on type
	switch driftType {
	case DriftTypeTagChange:
		return s.simulateLabelDrift(ctx, resource)
	case DriftTypeRuleAddition:
		return s.simulateFirewallRuleDrift(ctx, resource)
	case DriftTypeResourceCreation:
		return s.simulateResourceCreation(ctx, resource, state)
	case DriftTypeAttributeChange:
		return s.simulateAttributeChange(ctx, resource)
	default:
		return nil, fmt.Errorf("drift type %s not implemented for GCP", driftType)
	}
}

// simulateLabelDrift adds or modifies labels on a GCP resource
func (s *GCPSimulator) simulateLabelDrift(ctx context.Context, resource *parser.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "gcp",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeTagChange,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (labels are free)",
	}

	// Generate drift label
	driftLabel := map[string]string{
		"drift-simulation": fmt.Sprintf("created-%s", time.Now().Format("2006-01-02-15-04-05")),
	}

	switch resource.Type {
	case "google_compute_instance":
		// Add label to Compute Instance
		instanceName := s.extractResourceName(resource)
		zone := s.extractZone(resource)
		if instanceName == "" || zone == "" {
			return nil, fmt.Errorf("could not extract instance details")
		}

		apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s/setLabels",
			s.projectID, zone, instanceName)

		// Get current instance to get fingerprint
		getInstance, err := s.getResource(ctx, fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
			s.projectID, zone, instanceName))
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		fingerprint := ""
		if fp, ok := getInstance["labelFingerprint"].(string); ok {
			fingerprint = fp
		}

		body := map[string]interface{}{
			"labels":           driftLabel,
			"labelFingerprint": fingerprint,
		}

		if err := s.makeAPICall(ctx, "POST", apiURL, body); err != nil {
			return nil, fmt.Errorf("failed to add label: %w", err)
		}

		result.Changes["added_label"] = driftLabel
		result.RollbackData = &RollbackData{
			Provider:     "gcp",
			ResourceType: "google_compute_instance",
			ResourceID:   instanceName,
			Action:       "remove_label",
			OriginalData: map[string]interface{}{
				"zone":      zone,
				"label_key": "drift-simulation",
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	case "google_storage_bucket":
		// Add label to Storage Bucket
		bucketName := s.extractBucketName(resource)
		if bucketName == "" {
			return nil, fmt.Errorf("could not extract bucket name")
		}

		apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", bucketName)

		// Get current bucket
		bucket, err := s.getResource(ctx, apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get bucket: %w", err)
		}

		// Add drift label
		if labels, ok := bucket["labels"].(map[string]interface{}); ok {
			labels["drift-simulation"] = fmt.Sprintf("created-%s", time.Now().Format("2006-01-02-15-04-05"))
		} else {
			bucket["labels"] = driftLabel
		}

		if err := s.makeAPICall(ctx, "PATCH", apiURL, bucket); err != nil {
			return nil, fmt.Errorf("failed to add label to bucket: %w", err)
		}

		result.Changes["added_label"] = driftLabel
		result.RollbackData = &RollbackData{
			Provider:     "gcp",
			ResourceType: "google_storage_bucket",
			ResourceID:   bucketName,
			Action:       "remove_label",
			OriginalData: map[string]interface{}{
				"label_key": "drift-simulation",
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	default:
		return nil, fmt.Errorf("label drift not implemented for resource type %s", resource.Type)
	}

	return result, nil
}

// simulateFirewallRuleDrift adds a new firewall rule
func (s *GCPSimulator) simulateFirewallRuleDrift(ctx context.Context, resource *parser.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "gcp",
		ResourceType: "google_compute_firewall",
		DriftType:    DriftTypeRuleAddition,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (firewall rules are free)",
	}

	// Create a harmless firewall rule
	ruleName := fmt.Sprintf("drift-simulation-%d", time.Now().Unix())
	
	apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls", s.projectID)

	rule := map[string]interface{}{
		"name":        ruleName,
		"description": "DriftSimulation - Test firewall rule",
		"network":     fmt.Sprintf("projects/%s/global/networks/default", s.projectID),
		"priority":    65534,
		"direction":   "INGRESS",
		"sourceRanges": []string{"192.0.2.0/32"}, // TEST-NET-1
		"denied": []map[string]interface{}{
			{
				"IPProtocol": "tcp",
				"ports":      []string{"8443"},
			},
		},
	}

	if err := s.makeAPICall(ctx, "POST", apiURL, rule); err != nil {
		return nil, fmt.Errorf("failed to create firewall rule: %w", err)
	}

	result.ResourceID = ruleName
	result.Changes["created_rule"] = map[string]interface{}{
		"name":        ruleName,
		"protocol":    "tcp",
		"port":        "8443",
		"source":      "192.0.2.0/32",
		"action":      "deny",
		"description": "DriftSimulation - Test firewall rule",
	}
	result.RollbackData = &RollbackData{
		Provider:     "gcp",
		ResourceType: "google_compute_firewall",
		ResourceID:   ruleName,
		Action:       "delete_resource",
		Timestamp:    time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateResourceCreation creates a new resource not in state
func (s *GCPSimulator) simulateResourceCreation(ctx context.Context, resource *parser.Resource, state *parser.TerraformState) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "gcp",
		DriftType:    DriftTypeResourceCreation,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (using free tier resources)",
	}

	// Create a small storage bucket (free tier)
	bucketName := fmt.Sprintf("drift-simulation-%d", time.Now().Unix())
	
	apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b?project=%s", s.projectID)

	bucket := map[string]interface{}{
		"name":         bucketName,
		"location":     "us-central1",
		"storageClass": "STANDARD",
		"labels": map[string]string{
			"drift-simulation": "true",
			"auto-delete":      time.Now().Add(24 * time.Hour).Format("2006-01-02"),
		},
		"lifecycle": map[string]interface{}{
			"rule": []map[string]interface{}{
				{
					"action": map[string]interface{}{
						"type": "Delete",
					},
					"condition": map[string]interface{}{
						"age": 1,
					},
				},
			},
		},
	}

	if err := s.makeAPICall(ctx, "POST", apiURL, bucket); err != nil {
		return nil, fmt.Errorf("failed to create storage bucket: %w", err)
	}

	result.ResourceType = "google_storage_bucket"
	result.ResourceID = bucketName
	result.Changes["created_resource"] = map[string]interface{}{
		"type":     "google_storage_bucket",
		"name":     bucketName,
		"location": "us-central1",
	}
	result.RollbackData = &RollbackData{
		Provider:     "gcp",
		ResourceType: "google_storage_bucket",
		ResourceID:   bucketName,
		Action:       "delete_resource",
		Timestamp:    time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateAttributeChange modifies a resource attribute
func (s *GCPSimulator) simulateAttributeChange(ctx context.Context, resource *parser.Resource) (*SimulationResult, error) {
	// For GCP, we'll just add a label as most attribute changes require resource recreation
	return s.simulateLabelDrift(ctx, resource)
}

// DetectDrift detects drift in GCP resources
func (s *GCPSimulator) DetectDrift(ctx context.Context, state *parser.TerraformState) ([]DriftItem, error) {
	var drifts []DriftItem

	// Initialize if needed
	if s.httpClient == nil {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Check each resource in state
	for _, resource := range state.Resources {
		if !strings.HasPrefix(resource.Type, "google_") {
			continue
		}

		drift := s.checkResourceDrift(ctx, resource)
		if drift != nil {
			drifts = append(drifts, *drift)
		}
	}

	// Check for unmanaged resources
	unmanagedDrifts := s.checkUnmanagedResources(ctx, state)
	drifts = append(drifts, unmanagedDrifts...)

	return drifts, nil
}

// checkResourceDrift checks a single GCP resource for drift
func (s *GCPSimulator) checkResourceDrift(ctx context.Context, resource *parser.Resource) *DriftItem {
	switch resource.Type {
	case "google_compute_instance":
		return s.checkInstanceDrift(ctx, resource)
	case "google_storage_bucket":
		return s.checkBucketDrift(ctx, resource)
	case "google_compute_firewall":
		return s.checkFirewallDrift(ctx, resource)
	default:
		return nil
	}
}

// checkInstanceDrift checks Compute Instance for drift
func (s *GCPSimulator) checkInstanceDrift(ctx context.Context, resource *parser.Resource) *DriftItem {
	instanceName := s.extractResourceName(resource)
	zone := s.extractZone(resource)
	if instanceName == "" || zone == "" {
		return nil
	}

	apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
		s.projectID, zone, instanceName)

	instance, err := s.getResource(ctx, apiURL)
	if err != nil {
		return nil
	}

	// Check for drift simulation labels
	if labels, ok := instance["labels"].(map[string]interface{}); ok {
		if _, exists := labels["drift-simulation"]; exists {
			return &DriftItem{
				ResourceID:   instanceName,
				ResourceType: "google_compute_instance",
				DriftType:    "label_addition",
				Before: map[string]interface{}{
					"labels": s.extractResourceLabels(resource),
				},
				After: map[string]interface{}{
					"labels": labels,
				},
				Impact: "Low - Label addition detected",
			}
		}
	}

	return nil
}

// checkBucketDrift checks Storage Bucket for drift
func (s *GCPSimulator) checkBucketDrift(ctx context.Context, resource *parser.Resource) *DriftItem {
	bucketName := s.extractBucketName(resource)
	if bucketName == "" {
		return nil
	}

	apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", bucketName)

	bucket, err := s.getResource(ctx, apiURL)
	if err != nil {
		return nil
	}

	// Check for drift simulation labels
	if labels, ok := bucket["labels"].(map[string]interface{}); ok {
		if _, exists := labels["drift-simulation"]; exists {
			return &DriftItem{
				ResourceID:   bucketName,
				ResourceType: "google_storage_bucket",
				DriftType:    "label_addition",
				Before: map[string]interface{}{
					"labels": s.extractResourceLabels(resource),
				},
				After: map[string]interface{}{
					"labels": labels,
				},
				Impact: "Low - Label addition detected",
			}
		}
	}

	return nil
}

// checkFirewallDrift checks Firewall rules for drift
func (s *GCPSimulator) checkFirewallDrift(ctx context.Context, resource *parser.Resource) *DriftItem {
	ruleName := s.extractResourceName(resource)
	if ruleName == "" {
		return nil
	}

	apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s",
		s.projectID, ruleName)

	rule, err := s.getResource(ctx, apiURL)
	if err != nil {
		return nil
	}

	// Check for drift simulation in description
	if desc, ok := rule["description"].(string); ok && strings.Contains(desc, "DriftSimulation") {
		return &DriftItem{
			ResourceID:   ruleName,
			ResourceType: "google_compute_firewall",
			DriftType:    "attribute_change",
			Before: map[string]interface{}{
				"description": s.extractResourceDescription(resource),
			},
			After: map[string]interface{}{
				"description": desc,
			},
			Impact: "Medium - Firewall description changed",
		}
	}

	return nil
}

// checkUnmanagedResources checks for resources not in state
func (s *GCPSimulator) checkUnmanagedResources(ctx context.Context, state *parser.TerraformState) []DriftItem {
	var drifts []DriftItem

	// Check for drift simulation buckets
	apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b?project=%s", s.projectID)
	
	response, err := s.makeAPICallWithResponse(ctx, "GET", apiURL, nil)
	if err != nil {
		return drifts
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return drifts
	}

	if items, ok := result["items"].([]interface{}); ok {
		for _, item := range items {
			if bucket, ok := item.(map[string]interface{}); ok {
				if name, ok := bucket["name"].(string); ok {
					if strings.HasPrefix(name, "drift-simulation-") {
						// Check if this bucket is in state
						found := false
						for _, resource := range state.Resources {
							if resource.Type == "google_storage_bucket" {
								bucketName := s.extractBucketName(resource)
								if bucketName == name {
									found = true
									break
								}
							}
						}

						if !found {
							drifts = append(drifts, DriftItem{
								ResourceID:   name,
								ResourceType: "google_storage_bucket",
								DriftType:    "unmanaged_resource",
								After: map[string]interface{}{
									"name":     name,
									"location": bucket["location"],
									"labels":   bucket["labels"],
								},
								Impact: "High - Unmanaged storage bucket detected",
							})
						}
					}
				}
			}
		}
	}

	// Check for drift simulation firewall rules
	firewallURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls", s.projectID)
	
	response, err = s.makeAPICallWithResponse(ctx, "GET", firewallURL, nil)
	if err == nil {
		var firewallResult map[string]interface{}
		if err := json.Unmarshal(response, &firewallResult); err == nil {
			if items, ok := firewallResult["items"].([]interface{}); ok {
				for _, item := range items {
					if rule, ok := item.(map[string]interface{}); ok {
						if name, ok := rule["name"].(string); ok {
							if strings.HasPrefix(name, "drift-simulation-") {
								drifts = append(drifts, DriftItem{
									ResourceID:   name,
									ResourceType: "google_compute_firewall",
									DriftType:    "unmanaged_resource",
									After:        rule,
									Impact:       "High - Unmanaged firewall rule detected",
								})
							}
						}
					}
				}
			}
		}
	}

	return drifts
}

// Rollback undoes the simulated drift
func (s *GCPSimulator) Rollback(ctx context.Context, data *RollbackData) error {
	// Initialize if needed
	if s.httpClient == nil {
		if err := s.Initialize(ctx); err != nil {
			return err
		}
	}

	switch data.Action {
	case "remove_label":
		return s.rollbackLabelRemoval(ctx, data)
	case "delete_resource":
		return s.rollbackResourceDeletion(ctx, data)
	default:
		return fmt.Errorf("unknown rollback action: %s", data.Action)
	}
}

// Helper functions

func (s *GCPSimulator) findResource(resourceID string, state *parser.TerraformState) *parser.Resource {
	for _, resource := range state.Resources {
		if resource.ID == resourceID || resource.Name == resourceID {
			return resource
		}
	}
	return nil
}

func (s *GCPSimulator) extractResourceName(resource *parser.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if name, ok := resource.Instances[0].Attributes["name"].(string); ok {
			return name
		}
	}
	return ""
}

func (s *GCPSimulator) extractBucketName(resource *parser.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if name, ok := resource.Instances[0].Attributes["name"].(string); ok {
			return name
		}
		if id, ok := resource.Instances[0].Attributes["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (s *GCPSimulator) extractZone(resource *parser.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if zone, ok := resource.Instances[0].Attributes["zone"].(string); ok {
			return zone
		}
	}
	return "us-central1-a" // Default zone
}

func (s *GCPSimulator) extractResourceLabels(resource *parser.Resource) map[string]string {
	labels := make(map[string]string)
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if l, ok := resource.Instances[0].Attributes["labels"].(map[string]interface{}); ok {
			for k, v := range l {
				labels[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return labels
}

func (s *GCPSimulator) extractResourceDescription(resource *parser.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if desc, ok := resource.Instances[0].Attributes["description"].(string); ok {
			return desc
		}
	}
	return ""
}

func (s *GCPSimulator) makeAPICall(ctx context.Context, method, url string, body interface{}) error {
	_, err := s.makeAPICallWithResponse(ctx, method, url, body)
	return err
}

func (s *GCPSimulator) makeAPICallWithResponse(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyData []byte
	var err error

	if body != nil {
		bodyData, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(bodyData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// For simulation, return mock response
	mockResponse := map[string]interface{}{
		"simulated": true,
		"message":   "GCP drift simulation response",
	}
	
	return json.Marshal(mockResponse)
}

func (s *GCPSimulator) getResource(ctx context.Context, url string) (map[string]interface{}, error) {
	body, err := s.makeAPICallWithResponse(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resource map[string]interface{}
	if err := json.Unmarshal(body, &resource); err != nil {
		return nil, err
	}

	return resource, nil
}

// Rollback functions

func (s *GCPSimulator) rollbackLabelRemoval(ctx context.Context, data *RollbackData) error {
	labelKey := data.OriginalData["label_key"].(string)

	switch data.ResourceType {
	case "google_compute_instance":
		zone := data.OriginalData["zone"].(string)
		apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s/setLabels",
			s.projectID, zone, data.ResourceID)

		// Get current instance
		getInstance, err := s.getResource(ctx, fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
			s.projectID, zone, data.ResourceID))
		if err != nil {
			return err
		}

		// Remove drift label
		if labels, ok := getInstance["labels"].(map[string]interface{}); ok {
			delete(labels, labelKey)
			
			body := map[string]interface{}{
				"labels":           labels,
				"labelFingerprint": getInstance["labelFingerprint"],
			}
			
			return s.makeAPICall(ctx, "POST", apiURL, body)
		}

	case "google_storage_bucket":
		apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", data.ResourceID)

		// Get current bucket
		bucket, err := s.getResource(ctx, apiURL)
		if err != nil {
			return err
		}

		// Remove drift label
		if labels, ok := bucket["labels"].(map[string]interface{}); ok {
			delete(labels, labelKey)
			bucket["labels"] = labels
		}

		return s.makeAPICall(ctx, "PATCH", apiURL, bucket)
	}

	return nil
}

func (s *GCPSimulator) rollbackResourceDeletion(ctx context.Context, data *RollbackData) error {
	switch data.ResourceType {
	case "google_storage_bucket":
		// Delete the bucket
		apiURL := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", data.ResourceID)
		return s.makeAPICall(ctx, "DELETE", apiURL, nil)

	case "google_compute_firewall":
		// Delete the firewall rule
		apiURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s",
			s.projectID, data.ResourceID)
		return s.makeAPICall(ctx, "DELETE", apiURL, nil)
	}

	return fmt.Errorf("rollback not implemented for resource type %s", data.ResourceType)
}