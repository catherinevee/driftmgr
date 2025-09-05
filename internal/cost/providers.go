package cost

import (
	"context"
	"fmt"
	"strings"
)

// AWSCostProvider implements cost calculations for AWS resources
type AWSCostProvider struct {
	pricing map[string]map[string]float64 // resourceType -> size -> hourly cost
	region  string
}

// NewAWSCostProvider creates a new AWS cost provider
func NewAWSCostProvider() *AWSCostProvider {
	provider := &AWSCostProvider{
		pricing: make(map[string]map[string]float64),
		region:  "us-east-1",
	}

	// Initialize with sample pricing data
	// In production, this would fetch from AWS Pricing API
	provider.initializePricing()

	return provider
}

func (p *AWSCostProvider) initializePricing() {
	// EC2 instance pricing (hourly rates in USD)
	p.pricing["aws_instance"] = map[string]float64{
		"t2.micro":   0.0116,
		"t2.small":   0.023,
		"t2.medium":  0.0464,
		"t2.large":   0.0928,
		"t3.micro":   0.0104,
		"t3.small":   0.0208,
		"t3.medium":  0.0416,
		"t3.large":   0.0832,
		"m5.large":   0.096,
		"m5.xlarge":  0.192,
		"m5.2xlarge": 0.384,
		"c5.large":   0.085,
		"c5.xlarge":  0.17,
		"r5.large":   0.126,
		"r5.xlarge":  0.252,
	}

	// RDS instance pricing
	p.pricing["aws_db_instance"] = map[string]float64{
		"db.t2.micro":  0.017,
		"db.t2.small":  0.034,
		"db.t2.medium": 0.068,
		"db.t3.micro":  0.017,
		"db.t3.small":  0.034,
		"db.m5.large":  0.171,
		"db.m5.xlarge": 0.342,
	}

	// EBS volume pricing (per GB-month, converted to hourly)
	p.pricing["aws_ebs_volume"] = map[string]float64{
		"gp2": 0.10 / 730,  // $0.10 per GB-month
		"gp3": 0.08 / 730,  // $0.08 per GB-month
		"io1": 0.125 / 730, // $0.125 per GB-month
		"io2": 0.125 / 730,
		"st1": 0.045 / 730,
		"sc1": 0.025 / 730,
	}

	// S3 storage pricing (per GB-month, converted to hourly)
	p.pricing["aws_s3_bucket"] = map[string]float64{
		"STANDARD":            0.023 / 730,
		"INTELLIGENT_TIERING": 0.023 / 730,
		"STANDARD_IA":         0.0125 / 730,
		"ONEZONE_IA":          0.01 / 730,
		"GLACIER":             0.004 / 730,
		"DEEP_ARCHIVE":        0.001 / 730,
	}

	// Load balancer pricing
	p.pricing["aws_lb"] = map[string]float64{
		"application": 0.0225,
		"network":     0.0225,
		"gateway":     0.0125,
	}
}

func (p *AWSCostProvider) GetResourceCost(ctx context.Context, resourceType string,
	attributes map[string]interface{}) (*ResourceCost, error) {

	cost := &ResourceCost{
		ResourceType:   resourceType,
		Provider:       "aws",
		Currency:       "USD",
		PriceBreakdown: make(map[string]float64),
		Confidence:     1.0,
	}

	// Extract region
	if region, ok := attributes["region"].(string); ok {
		cost.Region = region
	} else if availabilityZone, ok := attributes["availability_zone"].(string); ok {
		// Extract region from AZ
		if len(availabilityZone) > 1 {
			cost.Region = availabilityZone[:len(availabilityZone)-1]
		}
	} else {
		cost.Region = p.region
	}

	switch resourceType {
	case "aws_instance":
		return p.calculateEC2Cost(cost, attributes)
	case "aws_db_instance", "aws_rds_cluster_instance":
		return p.calculateRDSCost(cost, attributes)
	case "aws_ebs_volume":
		return p.calculateEBSCost(cost, attributes)
	case "aws_s3_bucket":
		return p.calculateS3Cost(cost, attributes)
	case "aws_lb", "aws_alb", "aws_elb":
		return p.calculateLoadBalancerCost(cost, attributes)
	default:
		// Return zero cost for unsupported resources
		cost.Confidence = 0
		return cost, nil
	}
}

func (p *AWSCostProvider) calculateEC2Cost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	instanceType, ok := attrs["instance_type"].(string)
	if !ok {
		return nil, fmt.Errorf("instance_type not found")
	}

	pricing, exists := p.pricing["aws_instance"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for EC2 instances")
	}

	hourlyRate, exists := pricing[instanceType]
	if !exists {
		// Try to estimate based on similar instance types
		cost.Confidence = 0.7
		hourlyRate = 0.1 // Default estimate
	}

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["compute"] = hourlyRate

	// Add EBS costs if root block device is specified
	if rootDevice, ok := attrs["root_block_device"].([]interface{}); ok && len(rootDevice) > 0 {
		if device, ok := rootDevice[0].(map[string]interface{}); ok {
			if size, ok := device["volume_size"].(float64); ok {
				ebsCost := (0.10 / 730) * size // GP2 pricing
				cost.HourlyCost += ebsCost
				cost.PriceBreakdown["storage"] = ebsCost
			}
		}
	}

	// Check for data transfer costs (simplified)
	if publicIP, ok := attrs["associate_public_ip_address"].(bool); ok && publicIP {
		dataTransferCost := 0.01 // Simplified data transfer estimate
		cost.HourlyCost += dataTransferCost
		cost.PriceBreakdown["data_transfer"] = dataTransferCost
	}

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AWSCostProvider) calculateRDSCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	instanceClass, ok := attrs["instance_class"].(string)
	if !ok {
		return nil, fmt.Errorf("instance_class not found")
	}

	pricing, exists := p.pricing["aws_db_instance"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for RDS instances")
	}

	hourlyRate, exists := pricing[instanceClass]
	if !exists {
		cost.Confidence = 0.7
		hourlyRate = 0.15 // Default estimate
	}

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["compute"] = hourlyRate

	// Add storage costs
	if storage, ok := attrs["allocated_storage"].(float64); ok {
		storageCost := (0.115 / 730) * storage // GP2 pricing for RDS
		cost.HourlyCost += storageCost
		cost.PriceBreakdown["storage"] = storageCost
	}

	// Add backup storage costs
	if retention, ok := attrs["backup_retention_period"].(float64); ok && retention > 0 {
		if storage, ok := attrs["allocated_storage"].(float64); ok {
			backupCost := (0.095 / 730) * storage * (retention / 30) // Simplified backup cost
			cost.HourlyCost += backupCost
			cost.PriceBreakdown["backup"] = backupCost
		}
	}

	// Multi-AZ deployment doubles the cost
	if multiAZ, ok := attrs["multi_az"].(bool); ok && multiAZ {
		cost.HourlyCost *= 2
		cost.PriceBreakdown["multi_az"] = cost.PriceBreakdown["compute"]
	}

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AWSCostProvider) calculateEBSCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	volumeType := "gp2" // Default
	if vType, ok := attrs["type"].(string); ok {
		volumeType = vType
	}

	size := 8.0 // Default size
	if vSize, ok := attrs["size"].(float64); ok {
		size = vSize
	}

	pricing, exists := p.pricing["aws_ebs_volume"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for EBS volumes")
	}

	hourlyRate, exists := pricing[volumeType]
	if !exists {
		hourlyRate = pricing["gp2"] // Default to GP2
	}

	cost.HourlyCost = hourlyRate * size
	cost.PriceBreakdown["storage"] = cost.HourlyCost

	// Add IOPS costs for io1/io2
	if volumeType == "io1" || volumeType == "io2" {
		if iops, ok := attrs["iops"].(float64); ok {
			iopsCost := (0.065 / 730) * iops // $0.065 per IOPS-month
			cost.HourlyCost += iopsCost
			cost.PriceBreakdown["iops"] = iopsCost
		}
	}

	// Add snapshot costs if applicable
	if _, ok := attrs["snapshot_id"]; ok {
		snapshotCost := (0.05 / 730) * size // Simplified snapshot cost
		cost.HourlyCost += snapshotCost
		cost.PriceBreakdown["snapshot"] = snapshotCost
	}

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AWSCostProvider) calculateS3Cost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	// S3 costs are usage-based, so we'll provide estimates
	cost.Confidence = 0.5 // Lower confidence for S3 estimates

	storageClass := "STANDARD"
	if lifecycle, ok := attrs["lifecycle_rule"].([]interface{}); ok && len(lifecycle) > 0 {
		// Simplified: check for storage class transitions
		storageClass = "INTELLIGENT_TIERING"
	}

	pricing, exists := p.pricing["aws_s3_bucket"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for S3")
	}

	// Estimate 100GB of storage as baseline
	estimatedStorage := 100.0
	hourlyRate := pricing[storageClass] * estimatedStorage

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["storage"] = hourlyRate

	// Add request costs (simplified estimate)
	requestCost := 0.01 // Simplified request cost estimate
	cost.HourlyCost += requestCost
	cost.PriceBreakdown["requests"] = requestCost

	// Add data transfer costs if applicable
	if acceleration, ok := attrs["acceleration_status"].(string); ok && acceleration == "Enabled" {
		transferCost := 0.04 // Transfer acceleration cost estimate
		cost.HourlyCost += transferCost
		cost.PriceBreakdown["transfer_acceleration"] = transferCost
	}

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AWSCostProvider) calculateLoadBalancerCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	lbType := "application"
	if loadBalancerType, ok := attrs["load_balancer_type"].(string); ok {
		lbType = loadBalancerType
	}

	pricing, exists := p.pricing["aws_lb"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for load balancers")
	}

	hourlyRate, exists := pricing[lbType]
	if !exists {
		hourlyRate = pricing["application"]
	}

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["load_balancer"] = hourlyRate

	// Add LCU costs (simplified)
	lcuCost := 0.008 * 10 // Estimate 10 LCUs
	cost.HourlyCost += lcuCost
	cost.PriceBreakdown["lcu"] = lcuCost

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AWSCostProvider) GetPricingData(ctx context.Context, region string) error {
	// In production, this would fetch current pricing from AWS Pricing API
	p.region = region
	return nil
}

func (p *AWSCostProvider) SupportsResource(resourceType string) bool {
	supportedTypes := []string{
		"aws_instance",
		"aws_db_instance",
		"aws_rds_cluster_instance",
		"aws_ebs_volume",
		"aws_s3_bucket",
		"aws_lb",
		"aws_alb",
		"aws_elb",
	}

	for _, supported := range supportedTypes {
		if resourceType == supported {
			return true
		}
	}
	return false
}

// AzureCostProvider implements cost calculations for Azure resources
type AzureCostProvider struct {
	pricing map[string]map[string]float64
	region  string
}

// NewAzureCostProvider creates a new Azure cost provider
func NewAzureCostProvider() *AzureCostProvider {
	provider := &AzureCostProvider{
		pricing: make(map[string]map[string]float64),
		region:  "eastus",
	}

	provider.initializePricing()
	return provider
}

func (p *AzureCostProvider) initializePricing() {
	// Azure VM pricing
	p.pricing["azurerm_virtual_machine"] = map[string]float64{
		"Standard_B1s":    0.0104,
		"Standard_B1ms":   0.0207,
		"Standard_B2s":    0.0416,
		"Standard_B2ms":   0.0832,
		"Standard_D2s_v3": 0.096,
		"Standard_D4s_v3": 0.192,
		"Standard_E2s_v3": 0.126,
		"Standard_E4s_v3": 0.252,
	}

	// Azure Storage pricing (per GB-month)
	p.pricing["azurerm_storage_account"] = map[string]float64{
		"Standard_LRS": 0.0184 / 730,
		"Standard_GRS": 0.0368 / 730,
		"Standard_ZRS": 0.023 / 730,
		"Premium_LRS":  0.15 / 730,
	}
}

func (p *AzureCostProvider) GetResourceCost(ctx context.Context, resourceType string,
	attributes map[string]interface{}) (*ResourceCost, error) {

	cost := &ResourceCost{
		ResourceType:   resourceType,
		Provider:       "azure",
		Currency:       "USD",
		PriceBreakdown: make(map[string]float64),
		Confidence:     1.0,
	}

	if location, ok := attributes["location"].(string); ok {
		cost.Region = location
	} else {
		cost.Region = p.region
	}

	switch resourceType {
	case "azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine":
		return p.calculateVMCost(cost, attributes)
	case "azurerm_storage_account":
		return p.calculateStorageCost(cost, attributes)
	default:
		cost.Confidence = 0
		return cost, nil
	}
}

func (p *AzureCostProvider) calculateVMCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	vmSize, ok := attrs["vm_size"].(string)
	if !ok {
		if size, ok := attrs["size"].(string); ok {
			vmSize = size
		} else {
			return nil, fmt.Errorf("vm_size not found")
		}
	}

	pricing, exists := p.pricing["azurerm_virtual_machine"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for Azure VMs")
	}

	hourlyRate, exists := pricing[vmSize]
	if !exists {
		cost.Confidence = 0.7
		hourlyRate = 0.1
	}

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["compute"] = hourlyRate

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *AzureCostProvider) calculateStorageCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	accountTier := "Standard"
	if tier, ok := attrs["account_tier"].(string); ok {
		accountTier = tier
	}

	replicationType := "LRS"
	if replication, ok := attrs["account_replication_type"].(string); ok {
		replicationType = replication
	}

	storageType := fmt.Sprintf("%s_%s", accountTier, replicationType)

	pricing, exists := p.pricing["azurerm_storage_account"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for Azure Storage")
	}

	hourlyRate, exists := pricing[storageType]
	if !exists {
		cost.Confidence = 0.7
		hourlyRate = pricing["Standard_LRS"]
	}

	// Estimate 100GB storage
	estimatedStorage := 100.0
	cost.HourlyCost = hourlyRate * estimatedStorage
	cost.PriceBreakdown["storage"] = cost.HourlyCost

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12
	cost.Confidence = 0.5 // Lower confidence for storage estimates

	return cost, nil
}

func (p *AzureCostProvider) GetPricingData(ctx context.Context, region string) error {
	p.region = region
	return nil
}

func (p *AzureCostProvider) SupportsResource(resourceType string) bool {
	supportedTypes := []string{
		"azurerm_virtual_machine",
		"azurerm_linux_virtual_machine",
		"azurerm_windows_virtual_machine",
		"azurerm_storage_account",
		"azurerm_managed_disk",
	}

	for _, supported := range supportedTypes {
		if resourceType == supported {
			return true
		}
	}
	return false
}

// GCPCostProvider implements cost calculations for GCP resources
type GCPCostProvider struct {
	pricing map[string]map[string]float64
	region  string
}

// NewGCPCostProvider creates a new GCP cost provider
func NewGCPCostProvider() *GCPCostProvider {
	provider := &GCPCostProvider{
		pricing: make(map[string]map[string]float64),
		region:  "us-central1",
	}

	provider.initializePricing()
	return provider
}

func (p *GCPCostProvider) initializePricing() {
	// GCP VM pricing
	p.pricing["google_compute_instance"] = map[string]float64{
		"f1-micro":      0.0076,
		"g1-small":      0.0257,
		"n1-standard-1": 0.0475,
		"n1-standard-2": 0.0950,
		"n1-standard-4": 0.1900,
		"n2-standard-2": 0.0971,
		"n2-standard-4": 0.1942,
		"e2-micro":      0.0084,
		"e2-small":      0.0168,
		"e2-medium":     0.0335,
	}

	// GCP Storage pricing (per GB-month)
	p.pricing["google_storage_bucket"] = map[string]float64{
		"STANDARD": 0.020 / 730,
		"NEARLINE": 0.010 / 730,
		"COLDLINE": 0.004 / 730,
		"ARCHIVE":  0.0012 / 730,
	}
}

func (p *GCPCostProvider) GetResourceCost(ctx context.Context, resourceType string,
	attributes map[string]interface{}) (*ResourceCost, error) {

	cost := &ResourceCost{
		ResourceType:   resourceType,
		Provider:       "gcp",
		Currency:       "USD",
		PriceBreakdown: make(map[string]float64),
		Confidence:     1.0,
	}

	if zone, ok := attributes["zone"].(string); ok {
		// Extract region from zone
		parts := strings.Split(zone, "-")
		if len(parts) >= 2 {
			cost.Region = strings.Join(parts[:2], "-")
		}
	} else if region, ok := attributes["region"].(string); ok {
		cost.Region = region
	} else {
		cost.Region = p.region
	}

	switch resourceType {
	case "google_compute_instance":
		return p.calculateComputeCost(cost, attributes)
	case "google_storage_bucket":
		return p.calculateStorageCost(cost, attributes)
	default:
		cost.Confidence = 0
		return cost, nil
	}
}

func (p *GCPCostProvider) calculateComputeCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	machineType, ok := attrs["machine_type"].(string)
	if !ok {
		return nil, fmt.Errorf("machine_type not found")
	}

	// Extract machine type (could be full URL or just type)
	parts := strings.Split(machineType, "/")
	if len(parts) > 0 {
		machineType = parts[len(parts)-1]
	}

	pricing, exists := p.pricing["google_compute_instance"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for GCP instances")
	}

	hourlyRate, exists := pricing[machineType]
	if !exists {
		cost.Confidence = 0.7
		hourlyRate = 0.05
	}

	cost.HourlyCost = hourlyRate
	cost.PriceBreakdown["compute"] = hourlyRate

	// Add disk costs
	if disks, ok := attrs["boot_disk"].([]interface{}); ok && len(disks) > 0 {
		if disk, ok := disks[0].(map[string]interface{}); ok {
			diskSize := 10.0 // Default
			if size, ok := disk["initialize_params"].([]interface{}); ok && len(size) > 0 {
				if params, ok := size[0].(map[string]interface{}); ok {
					if s, ok := params["size"].(float64); ok {
						diskSize = s
					}
				}
			}
			diskCost := (0.040 / 730) * diskSize // Standard persistent disk
			cost.HourlyCost += diskCost
			cost.PriceBreakdown["disk"] = diskCost
		}
	}

	// Check for preemptible instances
	if scheduling, ok := attrs["scheduling"].([]interface{}); ok && len(scheduling) > 0 {
		if sched, ok := scheduling[0].(map[string]interface{}); ok {
			if preemptible, ok := sched["preemptible"].(bool); ok && preemptible {
				// Preemptible instances are ~60-80% cheaper
				cost.HourlyCost *= 0.3
				cost.PriceBreakdown["preemptible_discount"] = -cost.PriceBreakdown["compute"] * 0.7
			}
		}
	}

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12

	return cost, nil
}

func (p *GCPCostProvider) calculateStorageCost(cost *ResourceCost, attrs map[string]interface{}) (*ResourceCost, error) {
	storageClass := "STANDARD"
	if class, ok := attrs["storage_class"].(string); ok {
		storageClass = class
	}

	pricing, exists := p.pricing["google_storage_bucket"]
	if !exists {
		return nil, fmt.Errorf("no pricing data for GCS")
	}

	hourlyRate, exists := pricing[storageClass]
	if !exists {
		hourlyRate = pricing["STANDARD"]
	}

	// Estimate 100GB storage
	estimatedStorage := 100.0
	cost.HourlyCost = hourlyRate * estimatedStorage
	cost.PriceBreakdown["storage"] = cost.HourlyCost

	cost.MonthlyCost = cost.HourlyCost * 730
	cost.AnnualCost = cost.MonthlyCost * 12
	cost.Confidence = 0.5 // Lower confidence for storage estimates

	return cost, nil
}

func (p *GCPCostProvider) GetPricingData(ctx context.Context, region string) error {
	p.region = region
	return nil
}

func (p *GCPCostProvider) SupportsResource(resourceType string) bool {
	supportedTypes := []string{
		"google_compute_instance",
		"google_storage_bucket",
		"google_compute_disk",
		"google_sql_database_instance",
	}

	for _, supported := range supportedTypes {
		if resourceType == supported {
			return true
		}
	}
	return false
}
