package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/cloud/aws"
	"github.com/catherinevee/driftmgr/internal/cloud/azure"
	"github.com/catherinevee/driftmgr/internal/cloud/gcp"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// Additional global variables for perspective handling
var (
	perspectiveAnalyzer *state.StateAnalyzer
	awsDiscoveryService *aws.Discovery
	azureDiscoveryService *azure.Discovery
	gcpDiscoveryService *gcp.Discovery
)

// InitializePerspectiveServices initializes perspective-related services
func InitializePerspectiveServices() {
	perspectiveAnalyzer = state.NewStateAnalyzer()
	
	ctx := context.Background()
	awsDiscoveryService, _ = aws.NewDiscovery(ctx, aws.Config{})
	azureDiscoveryService, _ = azure.NewDiscovery(ctx, azure.Config{})
	gcpDiscoveryService, _ = gcp.NewDiscovery(ctx, gcp.Config{})
}

// DriftAnalysis represents the results of drift analysis
type DriftAnalysis struct {
	TotalDrifts        int
	BySeverity         map[string]int
	ByProvider         map[string]int
	ByResourceType     map[string]int
	MissingCount       int
	ExtraCount         int
	ModifiedCount      int
	UnmanagedResources []map[string]interface{}
	DriftedResources   []models.DriftResult
}

// handlePerspectiveActual performs actual perspective analysis without mock data
func handlePerspectiveActual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PerspectiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform actual perspective analysis
	startTime := time.Now()
	
	// Initialize services if needed
	if perspectiveAnalyzer == nil {
		InitializePerspectiveServices()
	}
	
	// Find and parse state files
	stateFiles := findTerraformStateFiles(".")
	if len(stateFiles) == 0 {
		// Return empty perspective if no state files
		response := models.PerspectiveResponse{
			Summary: models.AnalysisSummary{
				TotalDrifts:           0,
				BySeverity:            map[string]int{},
				ByProvider:            map[string]int{},
				ByResourceType:        map[string]int{},
				TotalStateResources:   0,
				TotalLiveResources:    0,
			},
			Duration: time.Since(startTime),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Parse the first state file or the one specified
	stateFilePath := stateFiles[0]
	if req.StateFileID != "" {
		for _, path := range stateFiles {
			if strings.Contains(path, req.StateFileID) {
				stateFilePath = path
				break
			}
		}
	}
	
	// Parse state file to get resources
	stateData, err := os.ReadFile(stateFilePath)
	if err != nil {
		logger.Error("Failed to read state file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read state file: %v", err), http.StatusInternalServerError)
		return
	}
	
	var stateContent map[string]interface{}
	if err := json.Unmarshal(stateData, &stateContent); err != nil {
		logger.Error("Failed to parse state file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse state file: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Extract resources and analyze
	stateResources := extractStateResources(stateContent)
	cloudResources := performCloudDiscovery(req)
	
	// Perform drift analysis
	driftAnalysis := analyzeDrift(stateResources, cloudResources)
	
	// Generate import commands for unmanaged resources
	importCommands := generateImportCommands(driftAnalysis.UnmanagedResources)
	
	// Build response with actual data
	response := models.PerspectiveResponse{
		Summary: models.AnalysisSummary{
			TotalDrifts:           driftAnalysis.TotalDrifts,
			BySeverity:            driftAnalysis.BySeverity,
			ByProvider:            driftAnalysis.ByProvider,
			ByResourceType:        driftAnalysis.ByResourceType,
			CriticalDrifts:        driftAnalysis.BySeverity["critical"],
			HighDrifts:            driftAnalysis.BySeverity["high"],
			MediumDrifts:          driftAnalysis.BySeverity["medium"],
			LowDrifts:             driftAnalysis.BySeverity["low"],
			TotalStateResources:   len(stateResources),
			TotalLiveResources:    len(cloudResources),
			Missing:               driftAnalysis.MissingCount,
			Extra:                 driftAnalysis.ExtraCount,
			Modified:              driftAnalysis.ModifiedCount,
			PerspectivePercentage: calculatePerspectiveScore(driftAnalysis),
			CoveragePercentage:    calculateCoveragePercentage(stateResources, cloudResources),
			DriftPercentage:       calculateDriftPercentage(driftAnalysis),
			DriftsFound:           driftAnalysis.TotalDrifts,
		},
		ImportCommands: importCommands,
		Duration: time.Since(startTime),
		DriftResults:   driftAnalysis.DriftedResources,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// extractStateResources extracts resources from Terraform state
func extractStateResources(stateContent map[string]interface{}) []map[string]interface{} {
	var resources []map[string]interface{}
	
	// Check for resources in modern state format (Terraform 0.12+)
	if resourcesData, ok := stateContent["resources"].([]interface{}); ok {
		for _, resData := range resourcesData {
			if resMap, ok := resData.(map[string]interface{}); ok {
				// Extract instances from each resource
				if instances, ok := resMap["instances"].([]interface{}); ok {
					for _, instData := range instances {
						if inst, ok := instData.(map[string]interface{}); ok {
							resource := map[string]interface{}{
								"id":       inst["attributes"].(map[string]interface{})["id"],
								"type":     resMap["type"],
								"name":     resMap["name"],
								"provider": resMap["provider"],
								"mode":     resMap["mode"],
								"attributes": inst["attributes"],
							}
							resources = append(resources, resource)
						}
					}
				}
			}
		}
	}
	
	// Also check modules for older state format
	if modules, ok := stateContent["modules"].([]interface{}); ok {
		for _, modData := range modules {
			if modMap, ok := modData.(map[string]interface{}); ok {
				if modResources, ok := modMap["resources"].(map[string]interface{}); ok {
					for resKey, resData := range modResources {
						if resMap, ok := resData.(map[string]interface{}); ok {
							resource := map[string]interface{}{
								"id":       resKey,
								"type":     resMap["type"],
								"provider": resMap["provider"],
								"primary":  resMap["primary"],
							}
							resources = append(resources, resource)
						}
					}
				}
			}
		}
	}
	
	return resources
}

// performCloudDiscovery performs actual cloud resource discovery
func performCloudDiscovery(req models.PerspectiveRequest) []map[string]interface{} {
	var cloudResources []map[string]interface{}
	ctx := context.Background()
	
	// Determine provider from request or detect from environment
	provider := req.Provider
	if provider == "" {
		// Auto-detect provider based on environment
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
			provider = "aws"
		} else if os.Getenv("AZURE_CLIENT_ID") != "" || os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
			provider = "azure"
		} else if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			provider = "gcp"
		}
	}
	
	// Perform discovery based on provider
	switch provider {
	case "aws":
		if awsDiscoveryService != nil {
			resources, err := awsDiscoveryService.DiscoverResources(ctx)
			if err == nil {
				for _, r := range resources {
					cloudResources = append(cloudResources, map[string]interface{}{
						"id":         r.ID,
						"type":       r.Type,
						"name":       r.Name,
						"provider":   r.Provider,
						"region":     r.Region,
						"tags":       r.Tags,
						"properties": r.Properties,
						"metadata":   r.Metadata,
					})
				}
			}
		}
		
	case "azure":
		if azureDiscoveryService != nil {
			resources, err := azureDiscoveryService.DiscoverResources(ctx)
			if err == nil {
				for _, r := range resources {
					cloudResources = append(cloudResources, map[string]interface{}{
						"id":         r.ID,
						"type":       r.Type,
						"name":       r.Name,
						"provider":   r.Provider,
						"region":     r.Region,
						"tags":       r.Tags,
						"properties": r.Properties,
						"metadata":   r.Metadata,
					})
				}
			}
		}
		
	case "gcp":
		if gcpDiscoveryService != nil {
			resources, err := gcpDiscoveryService.DiscoverResources(ctx)
			if err == nil {
				for _, r := range resources {
					cloudResources = append(cloudResources, map[string]interface{}{
						"id":         r.ID,
						"type":       r.Type,
						"name":       r.Name,
						"provider":   r.Provider,
						"region":     r.Region,
						"labels":     r.Tags,
						"properties": r.Properties,
						"metadata":   r.Metadata,
					})
				}
			}
		}
		
	default:
		// Try all providers if none specified
		if awsDiscoveryService != nil {
			if resources, err := awsDiscoveryService.DiscoverResources(ctx); err == nil {
				for _, r := range resources {
					cloudResources = append(cloudResources, map[string]interface{}{
						"id":       r.ID,
						"type":     r.Type,
						"name":     r.Name,
						"provider": "aws",
						"region":   r.Region,
						"metadata": r.Metadata,
					})
				}
			}
		}
	}
	
	return cloudResources
}

// analyzeDrift performs comprehensive drift analysis
func analyzeDrift(stateResources, cloudResources []map[string]interface{}) *DriftAnalysis {
	analysis := &DriftAnalysis{
		BySeverity:     make(map[string]int),
		ByProvider:     make(map[string]int),
		ByResourceType: make(map[string]int),
	}
	
	// Create maps for easier comparison
	stateMap := make(map[string]map[string]interface{})
	for _, res := range stateResources {
		// Build unique key from resource attributes
		var key string
		if id, ok := res["id"].(string); ok && id != "" {
			key = id
		} else {
			// Create composite key from type and name
			resType, _ := res["type"].(string)
			resName, _ := res["name"].(string)
			key = fmt.Sprintf("%s.%s", resType, resName)
		}
		stateMap[key] = res
	}
	
	cloudMap := make(map[string]map[string]interface{})
	for _, res := range cloudResources {
		if id, ok := res["id"].(string); ok {
			cloudMap[id] = res
		}
	}
	
	// Find missing resources (in state but not in cloud)
	for key, stateRes := range stateMap {
		found := false
		for _, cloudRes := range cloudMap {
			if resourcesMatch(stateRes, cloudRes) {
				found = true
				break
			}
		}
		
		if !found {
			analysis.MissingCount++
			analysis.TotalDrifts++
			analysis.BySeverity["high"]++
			
			if resType, ok := stateRes["type"].(string); ok {
				analysis.ByResourceType[resType]++
			}
			if provider, ok := stateRes["provider"].(string); ok {
				cleanProvider := strings.Split(provider, "/")[0]
				analysis.ByProvider[cleanProvider]++
			}
			
			// Add to drift results
			analysis.DriftedResources = append(analysis.DriftedResources, models.DriftResult{
				ResourceID:   key,
				ResourceType: getStringValue(stateRes, "type"),
				DriftType:    "missing",
				Severity:     "high",
				Description:  fmt.Sprintf("Resource %s exists in state but not in cloud", key),
			})
		}
	}
	
	// Find extra resources (in cloud but not in state)
	for id, cloudRes := range cloudMap {
		found := false
		for _, stateRes := range stateMap {
			if resourcesMatch(stateRes, cloudRes) {
				found = true
				break
			}
		}
		
		if !found {
			analysis.ExtraCount++
			analysis.TotalDrifts++
			analysis.BySeverity["medium"]++
			analysis.UnmanagedResources = append(analysis.UnmanagedResources, cloudRes)
			
			if resType, ok := cloudRes["type"].(string); ok {
				analysis.ByResourceType[resType]++
			}
			if provider, ok := cloudRes["provider"].(string); ok {
				analysis.ByProvider[provider]++
			}
			
			// Add to drift results
			analysis.DriftedResources = append(analysis.DriftedResources, models.DriftResult{
				ResourceID:   id,
				ResourceType: getStringValue(cloudRes, "type"),
				DriftType:    "unmanaged",
				Severity:     "medium",
				Description:  fmt.Sprintf("Resource %s exists in cloud but not in state", id),
			})
		}
	}
	
	// Find modified resources (exist in both but different)
	for key, stateRes := range stateMap {
		for cloudID, cloudRes := range cloudMap {
			if resourcesMatch(stateRes, cloudRes) {
				if hasConfigurationDrift(stateRes, cloudRes) {
					analysis.ModifiedCount++
					analysis.TotalDrifts++
					
					// Determine severity based on the type of drift
					severity := assessDriftSeverity(stateRes, cloudRes)
					analysis.BySeverity[severity]++
					
					if resType, ok := stateRes["type"].(string); ok {
						analysis.ByResourceType[resType]++
					}
					if provider, ok := stateRes["provider"].(string); ok {
						cleanProvider := strings.Split(provider, "/")[0]
						analysis.ByProvider[cleanProvider]++
					}
					
					// Add to drift results
					analysis.DriftedResources = append(analysis.DriftedResources, models.DriftResult{
						ResourceID:   cloudID,
						ResourceType: getStringValue(stateRes, "type"),
						DriftType:    "modified",
						Severity:     severity,
						Description:  fmt.Sprintf("Resource %s has configuration drift", key),
						Details:      compareResourceDetails(stateRes, cloudRes),
					})
				}
				break
			}
		}
	}
	
	// Ensure we have values for all severity levels
	if analysis.BySeverity["critical"] == 0 {
		analysis.BySeverity["critical"] = 0
	}
	if analysis.BySeverity["high"] == 0 {
		analysis.BySeverity["high"] = 0
	}
	if analysis.BySeverity["medium"] == 0 {
		analysis.BySeverity["medium"] = 0
	}
	if analysis.BySeverity["low"] == 0 {
		analysis.BySeverity["low"] = 0
	}
	
	return analysis
}

// resourcesMatch checks if a state resource matches a cloud resource
func resourcesMatch(stateRes, cloudRes map[string]interface{}) bool {
	// Try to match by ID first
	stateID := getStringValue(stateRes, "id")
	cloudID := getStringValue(cloudRes, "id")
	
	if stateID != "" && cloudID != "" && stateID == cloudID {
		return true
	}
	
	// Try to match by type and name
	stateType := getStringValue(stateRes, "type")
	cloudType := getStringValue(cloudRes, "type")
	stateName := getStringValue(stateRes, "name")
	cloudName := getStringValue(cloudRes, "name")
	
	if stateType == cloudType && stateName == cloudName {
		return true
	}
	
	// Try to match by attributes for resources without direct IDs
	if attrs, ok := stateRes["attributes"].(map[string]interface{}); ok {
		if attrID, ok := attrs["id"].(string); ok && attrID == cloudID {
			return true
		}
	}
	
	return false
}

// hasConfigurationDrift checks if two resources have configuration differences
func hasConfigurationDrift(stateRes, cloudRes map[string]interface{}) bool {
	// Compare attributes if available
	stateAttrs, stateOk := stateRes["attributes"].(map[string]interface{})
	cloudProps, cloudOk := cloudRes["properties"].(map[string]interface{})
	
	if stateOk && cloudOk {
		// Check key configuration attributes
		importantAttrs := []string{
			"instance_type", "size", "sku", "tier",
			"storage_encrypted", "encryption", "public_access",
			"security_groups", "network_security_group",
			"subnet_id", "vpc_id", "network",
		}
		
		for _, attr := range importantAttrs {
			stateVal := stateAttrs[attr]
			cloudVal := cloudProps[attr]
			
			if stateVal != nil && cloudVal != nil {
				if fmt.Sprintf("%v", stateVal) != fmt.Sprintf("%v", cloudVal) {
					return true
				}
			}
		}
	}
	
	// Check tags for drift
	stateTags := extractTags(stateRes)
	cloudTags := extractTags(cloudRes)
	
	if len(stateTags) != len(cloudTags) {
		return true
	}
	
	for key, stateVal := range stateTags {
		if cloudVal, exists := cloudTags[key]; !exists || cloudVal != stateVal {
			return true
		}
	}
	
	return false
}

// assessDriftSeverity determines the severity of configuration drift
func assessDriftSeverity(stateRes, cloudRes map[string]interface{}) string {
	resType := getStringValue(stateRes, "type")
	
	// Security-related resources get higher severity
	if strings.Contains(resType, "security_group") || 
	   strings.Contains(resType, "firewall") ||
	   strings.Contains(resType, "iam") ||
	   strings.Contains(resType, "role") ||
	   strings.Contains(resType, "policy") {
		return "critical"
	}
	
	// Network and database resources
	if strings.Contains(resType, "network") ||
	   strings.Contains(resType, "vpc") ||
	   strings.Contains(resType, "subnet") ||
	   strings.Contains(resType, "database") ||
	   strings.Contains(resType, "rds") ||
	   strings.Contains(resType, "sql") {
		return "high"
	}
	
	// Compute resources
	if strings.Contains(resType, "instance") ||
	   strings.Contains(resType, "vm") ||
	   strings.Contains(resType, "container") {
		return "medium"
	}
	
	// Everything else
	return "low"
}

// compareResourceDetails generates detailed comparison between resources
func compareResourceDetails(stateRes, cloudRes map[string]interface{}) map[string]interface{} {
	details := make(map[string]interface{})
	
	stateAttrs, _ := stateRes["attributes"].(map[string]interface{})
	cloudProps, _ := cloudRes["properties"].(map[string]interface{})
	
	if stateAttrs != nil && cloudProps != nil {
		differences := make(map[string]interface{})
		
		for key, stateVal := range stateAttrs {
			if cloudVal, exists := cloudProps[key]; exists {
				stateStr := fmt.Sprintf("%v", stateVal)
				cloudStr := fmt.Sprintf("%v", cloudVal)
				if stateStr != cloudStr {
					differences[key] = map[string]interface{}{
						"state_value": stateStr,
						"cloud_value": cloudStr,
					}
				}
			}
		}
		
		details["differences"] = differences
	}
	
	return details
}

// extractTags extracts tags from a resource
func extractTags(resource map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	
	// Try different tag field names
	tagFields := []string{"tags", "Tags", "labels", "Labels"}
	
	for _, field := range tagFields {
		if tagData, ok := resource[field]; ok {
			switch t := tagData.(type) {
			case map[string]interface{}:
				for k, v := range t {
					tags[k] = fmt.Sprintf("%v", v)
				}
			case map[string]string:
				tags = t
			}
			break
		}
	}
	
	// Also check in attributes
	if attrs, ok := resource["attributes"].(map[string]interface{}); ok {
		for _, field := range tagFields {
			if tagData, ok := attrs[field]; ok {
				switch t := tagData.(type) {
				case map[string]interface{}:
					for k, v := range t {
						tags[k] = fmt.Sprintf("%v", v)
					}
				case map[string]string:
					tags = t
				}
				break
			}
		}
	}
	
	return tags
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// generateImportCommands generates Terraform import commands for unmanaged resources
func generateImportCommands(unmanagedResources []map[string]interface{}) []string {
	var commands []string
	
	for _, res := range unmanagedResources {
		resType := getStringValue(res, "type")
		resName := getStringValue(res, "name")
		resID := getStringValue(res, "id")
		
		if resType != "" && resName != "" && resID != "" {
			// Convert resource name to valid Terraform identifier
			tfName := strings.ReplaceAll(resName, "-", "_")
			tfName = strings.ReplaceAll(tfName, ".", "_")
			tfName = strings.ReplaceAll(tfName, "/", "_")
			
			// Generate appropriate import command based on resource type
			command := generateResourceImportCommand(resType, tfName, resID)
			if command != "" {
				commands = append(commands, command)
			}
			
			// Limit to 10 import commands for readability
			if len(commands) >= 10 {
				break
			}
		}
	}
	
	return commands
}

// generateResourceImportCommand generates the appropriate import command for a resource type
func generateResourceImportCommand(resType, tfName, resID string) string {
	// Map cloud resource types to Terraform resource types
	typeMapping := map[string]string{
		"Microsoft.Compute/virtualMachines": "azurerm_virtual_machine",
		"Microsoft.Storage/storageAccounts": "azurerm_storage_account",
		"Microsoft.Network/virtualNetworks": "azurerm_virtual_network",
		"AWS::EC2::Instance": "aws_instance",
		"AWS::S3::Bucket": "aws_s3_bucket",
		"AWS::EC2::SecurityGroup": "aws_security_group",
		"compute.v1.instance": "google_compute_instance",
		"storage.v1.bucket": "google_storage_bucket",
	}
	
	// Check if we have a mapping
	if tfType, ok := typeMapping[resType]; ok {
		return fmt.Sprintf("terraform import %s.%s %s", tfType, tfName, resID)
	}
	
	// Try to infer from resource type
	if strings.Contains(strings.ToLower(resType), "instance") {
		if strings.Contains(resType, "aws") {
			return fmt.Sprintf("terraform import aws_instance.%s %s", tfName, resID)
		} else if strings.Contains(resType, "azure") {
			return fmt.Sprintf("terraform import azurerm_virtual_machine.%s %s", tfName, resID)
		} else if strings.Contains(resType, "google") {
			return fmt.Sprintf("terraform import google_compute_instance.%s %s", tfName, resID)
		}
	}
	
	// Default format
	return fmt.Sprintf("terraform import %s.%s %s", resType, tfName, resID)
}

// calculatePerspectiveScore calculates the overall perspective score
func calculatePerspectiveScore(analysis *DriftAnalysis) float64 {
	if analysis.TotalDrifts == 0 {
		return 100.0
	}
	
	// Weight different types of drift
	criticalWeight := float64(analysis.BySeverity["critical"]) * 10.0
	highWeight := float64(analysis.BySeverity["high"]) * 5.0
	mediumWeight := float64(analysis.BySeverity["medium"]) * 2.0
	lowWeight := float64(analysis.BySeverity["low"]) * 1.0
	
	totalWeight := criticalWeight + highWeight + mediumWeight + lowWeight
	maxPossibleWeight := float64(analysis.TotalDrifts) * 10.0
	
	if maxPossibleWeight == 0 {
		return 100.0
	}
	
	// Calculate score (100 = no drift, 0 = maximum drift)
	score := 100.0 * (1.0 - (totalWeight / maxPossibleWeight))
	
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// calculateCoveragePercentage calculates what percentage of cloud resources are managed
func calculateCoveragePercentage(stateResources, cloudResources []map[string]interface{}) float64 {
	if len(cloudResources) == 0 {
		if len(stateResources) > 0 {
			return 100.0 // All state resources, no cloud resources
		}
		return 0.0
	}
	
	managedCount := 0
	for _, cloudRes := range cloudResources {
		for _, stateRes := range stateResources {
			if resourcesMatch(stateRes, cloudRes) {
				managedCount++
				break
			}
		}
	}
	
	return (float64(managedCount) / float64(len(cloudResources))) * 100.0
}

// calculateDriftPercentage calculates what percentage of resources have drift
func calculateDriftPercentage(analysis *DriftAnalysis) float64 {
	totalResources := analysis.MissingCount + analysis.ExtraCount + analysis.ModifiedCount
	
	// Avoid counting resources multiple times
	uniqueResources := make(map[string]bool)
	for _, drift := range analysis.DriftedResources {
		uniqueResources[drift.ResourceID] = true
	}
	
	if len(uniqueResources) == 0 {
		return 0.0
	}
	
	if totalResources == 0 {
		return 0.0
	}
	
	return (float64(len(uniqueResources)) / float64(totalResources)) * 100.0
}