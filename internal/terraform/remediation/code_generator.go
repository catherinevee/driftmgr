package remediation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// CodeGenerator generates Terraform code for remediation
type CodeGenerator struct {
	config          *RemediationConfig
	providerConfigs map[string]*ProviderConfig
	moduleTemplates map[string]*ModuleTemplate
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Alias   string            `json:"alias,omitempty"`
	Config  map[string]string `json:"config"`
}

// ModuleTemplate represents a module template
type ModuleTemplate struct {
	Name        string                 `json:"name"`
	Source      string                 `json:"source"`
	Version     string                 `json:"version"`
	Variables   map[string]interface{} `json:"variables"`
	Description string                 `json:"description"`
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(config *RemediationConfig) *CodeGenerator {
	return &CodeGenerator{
		config:          config,
		providerConfigs: make(map[string]*ProviderConfig),
		moduleTemplates: make(map[string]*ModuleTemplate),
	}
}

// GenerateCode generates Terraform code for a remediation plan
func (g *CodeGenerator) GenerateCode(plan *RemediationPlan) (string, error) {
	switch g.config.OutputFormat {
	case "hcl":
		return g.generateHCLCode(plan)
	case "json":
		return g.generateJSONCode(plan)
	default:
		return g.generateHCLCode(plan)
	}
}

// generateHCLCode generates HCL format Terraform code
func (g *CodeGenerator) generateHCLCode(plan *RemediationPlan) (string, error) {
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	// Add header comment
	g.addHeaderComment(rootBody, plan)

	// Generate provider block if needed
	if err := g.generateProviderBlock(rootBody, plan); err != nil {
		return "", fmt.Errorf("failed to generate provider block: %w", err)
	}

	// Generate resource block based on remediation type
	switch plan.Type {
	case RemediationTypeCreate:
		if err := g.generateCreateResource(rootBody, plan); err != nil {
			return "", fmt.Errorf("failed to generate create resource: %w", err)
		}

	case RemediationTypeUpdate:
		if err := g.generateUpdateResource(rootBody, plan); err != nil {
			return "", fmt.Errorf("failed to generate update resource: %w", err)
		}

	case RemediationTypeReplace:
		if err := g.generateReplaceResource(rootBody, plan); err != nil {
			return "", fmt.Errorf("failed to generate replace resource: %w", err)
		}

	case RemediationTypeDelete:
		if err := g.generateDeleteResource(rootBody, plan); err != nil {
			return "", fmt.Errorf("failed to generate delete resource: %w", err)
		}

	case RemediationTypeImport:
		if err := g.generateImportResource(rootBody, plan); err != nil {
			return "", fmt.Errorf("failed to generate import resource: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported remediation type: %s", plan.Type)
	}

	// Add outputs if needed
	g.generateOutputs(rootBody, plan)

	// Format the HCL code
	return string(f.Bytes()), nil
}

// generateJSONCode generates JSON format Terraform code
func (g *CodeGenerator) generateJSONCode(plan *RemediationPlan) (string, error) {
	// Implementation for JSON format
	// This would generate terraform.tf.json format
	config := map[string]interface{}{
		"terraform": map[string]interface{}{
			"required_version": g.config.TerraformVersion,
		},
	}

	// Add provider configuration
	providers := make(map[string]interface{})
	providerName := g.getProviderName(plan.Provider)
	providers[providerName] = map[string]interface{}{
		"version": g.config.ProviderVersions[providerName],
	}
	config["provider"] = providers

	// Add resource configuration
	resources := make(map[string]interface{})
	resourceKey := fmt.Sprintf("%s.%s", plan.ResourceType, plan.ResourceName)
	resources[resourceKey] = plan.DesiredState
	config["resource"] = resources

	// Convert to JSON
	jsonBytes, err := jsonMarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// addHeaderComment adds a header comment to the generated code
func (g *CodeGenerator) addHeaderComment(body *hclwrite.Body, plan *RemediationPlan) {
	// HCL writer doesn't support comments directly, so we'll add metadata as locals
	locals := body.AppendNewBlock("locals", nil)
	localsBody := locals.Body()
	localsBody.SetAttributeValue("_remediation_plan_id", cty.StringVal(plan.ID))
	localsBody.SetAttributeValue("_remediation_type", cty.StringVal(string(plan.Type)))
	localsBody.SetAttributeValue("_generated_at", cty.StringVal(plan.CreatedAt.Format("2006-01-02 15:04:05")))
}

// generateProviderBlock generates the provider configuration block
func (g *CodeGenerator) generateProviderBlock(body *hclwrite.Body, plan *RemediationPlan) error {
	providerName := g.getProviderName(plan.Provider)
	
	// Check if we have a custom provider configuration
	if config, ok := g.providerConfigs[providerName]; ok {
		provider := body.AppendNewBlock("provider", []string{providerName})
		providerBody := provider.Body()
		
		if config.Version != "" {
			providerBody.SetAttributeValue("version", cty.StringVal(config.Version))
		}
		
		if config.Alias != "" {
			providerBody.SetAttributeValue("alias", cty.StringVal(config.Alias))
		}
		
		// Add provider-specific configuration
		for key, value := range config.Config {
			providerBody.SetAttributeValue(key, cty.StringVal(value))
		}
	} else {
		// Generate default provider configuration
		provider := body.AppendNewBlock("provider", []string{providerName})
		providerBody := provider.Body()
		
		// Set version if configured
		if version, ok := g.config.ProviderVersions[providerName]; ok {
			providerBody.SetAttributeValue("version", cty.StringVal(version))
		}
		
		// Add common provider settings based on provider type
		switch providerName {
		case "aws":
			// AWS provider configuration
			providerBody.SetAttributeValue("region", cty.StringVal("${var.aws_region}"))
			
		case "azurerm":
			// Azure provider configuration
			providerBody.SetAttributeValue("features", cty.ObjectVal(map[string]cty.Value{}))
			
		case "google":
			// GCP provider configuration
			providerBody.SetAttributeValue("project", cty.StringVal("${var.gcp_project}"))
			providerBody.SetAttributeValue("region", cty.StringVal("${var.gcp_region}"))
			
		case "digitalocean":
			// DigitalOcean provider configuration
			providerBody.SetAttributeValue("token", cty.StringVal("${var.do_token}"))
		}
	}
	
	return nil
}

// generateCreateResource generates code for creating a new resource
func (g *CodeGenerator) generateCreateResource(body *hclwrite.Body, plan *RemediationPlan) error {
	resource := body.AppendNewBlock("resource", []string{plan.ResourceType, plan.ResourceName})
	resourceBody := resource.Body()
	
	// Add resource attributes from desired state
	for key, value := range plan.DesiredState {
		if err := g.setAttribute(resourceBody, key, value); err != nil {
			return fmt.Errorf("failed to set attribute %s: %w", key, err)
		}
	}
	
	// Add lifecycle block if needed
	if g.shouldAddLifecycle(plan) {
		lifecycle := resourceBody.AppendNewBlock("lifecycle", nil)
		lifecycleBody := lifecycle.Body()
		lifecycleBody.SetAttributeValue("create_before_destroy", cty.BoolVal(true))
		lifecycleBody.SetAttributeValue("prevent_destroy", cty.BoolVal(false))
	}
	
	// Add tags
	if tags := g.generateTags(plan); len(tags) > 0 {
		tagsVal := make(map[string]cty.Value)
		for k, v := range tags {
			tagsVal[k] = cty.StringVal(v)
		}
		resourceBody.SetAttributeValue("tags", cty.ObjectVal(tagsVal))
	}
	
	return nil
}

// generateUpdateResource generates code for updating an existing resource
func (g *CodeGenerator) generateUpdateResource(body *hclwrite.Body, plan *RemediationPlan) error {
	resource := body.AppendNewBlock("resource", []string{plan.ResourceType, plan.ResourceName})
	resourceBody := resource.Body()
	
	// Only include changed attributes
	for _, change := range plan.Changes {
		if change.Action == "update" || change.Action == "add" {
			if err := g.setAttribute(resourceBody, change.Path, change.NewValue); err != nil {
				return fmt.Errorf("failed to set attribute %s: %w", change.Path, err)
			}
		}
	}
	
	// Add lifecycle block to ignore certain changes if configured
	if len(g.getIgnoreChanges(plan)) > 0 {
		lifecycle := resourceBody.AppendNewBlock("lifecycle", nil)
		lifecycleBody := lifecycle.Body()
		
		ignoreList := make([]cty.Value, 0)
		for _, field := range g.getIgnoreChanges(plan) {
			ignoreList = append(ignoreList, cty.StringVal(field))
		}
		lifecycleBody.SetAttributeValue("ignore_changes", cty.ListVal(ignoreList))
	}
	
	return nil
}

// generateReplaceResource generates code for replacing a resource
func (g *CodeGenerator) generateReplaceResource(body *hclwrite.Body, plan *RemediationPlan) error {
	// Add comment about replacement
	comment := body.AppendNewBlock("locals", nil)
	commentBody := comment.Body()
	commentBody.SetAttributeValue("_replacement_note", cty.StringVal(
		fmt.Sprintf("Resource %s.%s will be replaced", plan.ResourceType, plan.ResourceName),
	))
	
	// Generate the new resource configuration
	resource := body.AppendNewBlock("resource", []string{plan.ResourceType, plan.ResourceName})
	resourceBody := resource.Body()
	
	// Add all attributes from desired state
	for key, value := range plan.DesiredState {
		if err := g.setAttribute(resourceBody, key, value); err != nil {
			return fmt.Errorf("failed to set attribute %s: %w", key, err)
		}
	}
	
	// Add lifecycle block for replacement
	lifecycle := resourceBody.AppendNewBlock("lifecycle", nil)
	lifecycleBody := lifecycle.Body()
	lifecycleBody.SetAttributeValue("create_before_destroy", cty.BoolVal(true))
	
	return nil
}

// generateDeleteResource generates code for deleting a resource
func (g *CodeGenerator) generateDeleteResource(body *hclwrite.Body, plan *RemediationPlan) error {
	// For deletion, we generate a moved block or removed block (Terraform 1.1+)
	removed := body.AppendNewBlock("removed", nil)
	removedBody := removed.Body()
	removedBody.SetAttributeValue("from", cty.StringVal(
		fmt.Sprintf("%s.%s", plan.ResourceType, plan.ResourceName),
	))
	
	// Add lifecycle block to prevent accidental recreation
	lifecycle := removedBody.AppendNewBlock("lifecycle", nil)
	lifecycleBody := lifecycle.Body()
	lifecycleBody.SetAttributeValue("destroy", cty.BoolVal(true))
	
	return nil
}

// generateImportResource generates code for importing a resource
func (g *CodeGenerator) generateImportResource(body *hclwrite.Body, plan *RemediationPlan) error {
	// Generate resource block with minimal required configuration
	resource := body.AppendNewBlock("resource", []string{plan.ResourceType, plan.ResourceName})
	resourceBody := resource.Body()
	
	// Add required attributes only
	requiredAttrs := g.getRequiredAttributes(plan.ResourceType)
	for _, attr := range requiredAttrs {
		if value, ok := plan.CurrentState[attr]; ok {
			if err := g.setAttribute(resourceBody, attr, value); err != nil {
				return fmt.Errorf("failed to set required attribute %s: %w", attr, err)
			}
		}
	}
	
	// Add import block (Terraform 1.5+)
	importBlock := body.AppendNewBlock("import", nil)
	importBody := importBlock.Body()
	importBody.SetAttributeValue("to", cty.StringVal(
		fmt.Sprintf("%s.%s", plan.ResourceType, plan.ResourceName),
	))
	importBody.SetAttributeValue("id", cty.StringVal(plan.ResourceID))
	
	return nil
}

// GenerateImportCommands generates terraform import commands
func (g *CodeGenerator) GenerateImportCommands(plan *RemediationPlan) []string {
	commands := []string{}
	
	// Generate main import command
	mainCommand := fmt.Sprintf("terraform import %s.%s %s",
		plan.ResourceType,
		plan.ResourceName,
		plan.ResourceID,
	)
	commands = append(commands, mainCommand)
	
	// Add any additional import commands for related resources
	relatedResources := g.getRelatedResources(plan)
	for _, related := range relatedResources {
		cmd := fmt.Sprintf("terraform import %s %s",
			related.ResourceAddress,
			related.ImportID,
		)
		commands = append(commands, cmd)
	}
	
	return commands
}

// setAttribute sets an attribute value in the HCL body
func (g *CodeGenerator) setAttribute(body *hclwrite.Body, key string, value interface{}) error {
	// Handle nested attributes
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		blockName := parts[0]
		block := body.AppendNewBlock(blockName, nil)
		blockBody := block.Body()
		return g.setAttribute(blockBody, strings.Join(parts[1:], "."), value)
	}
	
	// Convert value to cty.Value
	ctyValue, err := g.convertToCtyValue(value)
	if err != nil {
		return err
	}
	
	body.SetAttributeValue(key, ctyValue)
	return nil
}

// convertToCtyValue converts a Go value to cty.Value
func (g *CodeGenerator) convertToCtyValue(value interface{}) (cty.Value, error) {
	switch v := value.(type) {
	case string:
		return cty.StringVal(v), nil
	case bool:
		return cty.BoolVal(v), nil
	case int:
		return cty.NumberIntVal(int64(v)), nil
	case int64:
		return cty.NumberIntVal(v), nil
	case float64:
		return cty.NumberFloatVal(v), nil
	case []interface{}:
		vals := make([]cty.Value, len(v))
		for i, item := range v {
			val, err := g.convertToCtyValue(item)
			if err != nil {
				return cty.NilVal, err
			}
			vals[i] = val
		}
		return cty.ListVal(vals), nil
	case map[string]interface{}:
		vals := make(map[string]cty.Value)
		for k, item := range v {
			val, err := g.convertToCtyValue(item)
			if err != nil {
				return cty.NilVal, err
			}
			vals[k] = val
		}
		return cty.ObjectVal(vals), nil
	case nil:
		return cty.NullVal(cty.String), nil
	default:
		// Try to convert to string as fallback
		return cty.StringVal(fmt.Sprintf("%v", value)), nil
	}
}

// generateOutputs generates output blocks
func (g *CodeGenerator) generateOutputs(body *hclwrite.Body, plan *RemediationPlan) {
	// Generate output for resource ID
	output := body.AppendNewBlock("output", []string{fmt.Sprintf("%s_id", plan.ResourceName)})
	outputBody := output.Body()
	outputBody.SetAttributeValue("value", cty.StringVal(
		fmt.Sprintf("${%s.%s.id}", plan.ResourceType, plan.ResourceName),
	))
	outputBody.SetAttributeValue("description", cty.StringVal(
		fmt.Sprintf("ID of %s resource", plan.ResourceName),
	))
}

// Helper methods

// getProviderName maps provider names to Terraform provider names
func (g *CodeGenerator) getProviderName(provider string) string {
	providerMap := map[string]string{
		"aws":          "aws",
		"azure":        "azurerm",
		"gcp":          "google",
		"digitalocean": "digitalocean",
	}
	
	if mapped, ok := providerMap[strings.ToLower(provider)]; ok {
		return mapped
	}
	return strings.ToLower(provider)
}

// shouldAddLifecycle determines if lifecycle block should be added
func (g *CodeGenerator) shouldAddLifecycle(plan *RemediationPlan) bool {
	// Add lifecycle for critical resources or based on configuration
	criticalTypes := []string{
		"aws_db_instance",
		"aws_rds_cluster",
		"azurerm_sql_database",
		"google_sql_database_instance",
	}
	
	for _, critical := range criticalTypes {
		if plan.ResourceType == critical {
			return true
		}
	}
	
	return false
}

// generateTags generates standard tags for resources
func (g *CodeGenerator) generateTags(plan *RemediationPlan) map[string]string {
	tags := make(map[string]string)
	
	tags["ManagedBy"] = "DriftMgr"
	tags["RemediationPlanID"] = plan.ID
	tags["RemediationType"] = string(plan.Type)
	tags["Timestamp"] = plan.CreatedAt.Format("2006-01-02T15:04:05Z")
	
	// Add any existing tags from desired state
	if existingTags, ok := plan.DesiredState["tags"].(map[string]interface{}); ok {
		for k, v := range existingTags {
			tags[k] = fmt.Sprintf("%v", v)
		}
	}
	
	return tags
}

// getIgnoreChanges returns fields to ignore in lifecycle block
func (g *CodeGenerator) getIgnoreChanges(plan *RemediationPlan) []string {
	ignoreFields := []string{}
	
	// Add fields marked as low sensitivity
	for _, change := range plan.Changes {
		if change.Sensitivity == "low" {
			ignoreFields = append(ignoreFields, change.Path)
		}
	}
	
	// Add commonly ignored fields
	commonIgnored := []string{"tags", "tags_all"}
	ignoreFields = append(ignoreFields, commonIgnored...)
	
	return ignoreFields
}

// getRequiredAttributes returns required attributes for a resource type
func (g *CodeGenerator) getRequiredAttributes(resourceType string) []string {
	requiredMap := map[string][]string{
		"aws_instance":          {"ami", "instance_type"},
		"aws_s3_bucket":         {"bucket"},
		"aws_security_group":    {"name"},
		"azurerm_virtual_machine": {"name", "location", "resource_group_name"},
		"google_compute_instance": {"name", "machine_type", "zone"},
		"digitalocean_droplet":   {"name", "size", "image", "region"},
	}
	
	if required, ok := requiredMap[resourceType]; ok {
		return required
	}
	
	return []string{"name"}
}

// RelatedResource represents a related resource for import
type RelatedResource struct {
	ResourceAddress string
	ImportID        string
}

// getRelatedResources identifies related resources that need importing
func (g *CodeGenerator) getRelatedResources(plan *RemediationPlan) []RelatedResource {
	related := []RelatedResource{}
	
	// Add logic to identify related resources based on resource type
	// For example, security group rules for a security group
	
	return related
}

// SetProviderConfig sets a custom provider configuration
func (g *CodeGenerator) SetProviderConfig(provider string, config *ProviderConfig) {
	g.providerConfigs[provider] = config
}

// SetModuleTemplate sets a module template
func (g *CodeGenerator) SetModuleTemplate(name string, template *ModuleTemplate) {
	g.moduleTemplates[name] = template
}

// jsonMarshalIndent is a helper function for JSON marshaling with indentation
func jsonMarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent(prefix, indent)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}