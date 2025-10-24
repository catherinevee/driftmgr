package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/catherinevee/driftmgr/internal/models"
)

// RDSService handles RDS-related operations
type RDSService struct {
	client *rds.Client
	region string
}

// NewRDSService creates a new RDS service
func NewRDSService(client *rds.Client, region string) *RDSService {
	return &RDSService{
		client: client,
		region: region,
	}
}

// DiscoverInstances discovers RDS instances
func (s *RDSService) DiscoverInstances(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe DB instances
	result, err := s.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB instances: %w", err)
	}

	for _, instance := range result.DBInstances {
		resource := s.convertDBInstanceToResource(instance)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertDBInstanceToResource converts an RDS DB instance to a CloudResource
func (s *RDSService) convertDBInstanceToResource(instance rds.DBInstance) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("rds", *instance.DBInstanceIdentifier),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSRDSInstance),
		Name:      *instance.DBInstanceIdentifier,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the instance
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"db_instance_identifier":                     instance.DBInstanceIdentifier,
			"db_instance_class":                          instance.DBInstanceClass,
			"engine":                                     instance.Engine,
			"engine_version":                             instance.EngineVersion,
			"allocated_storage":                          instance.AllocatedStorage,
			"storage_type":                               instance.StorageType,
			"db_instance_status":                         instance.DBInstanceStatus,
			"master_username":                            instance.MasterUsername,
			"db_name":                                    instance.DBName,
			"endpoint":                                   instance.Endpoint,
			"availability_zone":                          instance.AvailabilityZone,
			"multi_az":                                   instance.MultiAZ,
			"publicly_accessible":                        instance.PubliclyAccessible,
			"storage_encrypted":                          instance.StorageEncrypted,
			"kms_key_id":                                 instance.KmsKeyId,
			"backup_retention_period":                    instance.BackupRetentionPeriod,
			"preferred_backup_window":                    instance.PreferredBackupWindow,
			"preferred_maintenance_window":               instance.PreferredMaintenanceWindow,
			"vpc_security_groups":                        instance.VpcSecurityGroups,
			"db_subnet_group":                            instance.DBSubnetGroup,
			"parameter_groups":                           instance.DBParameterGroups,
			"option_groups":                              instance.OptionGroupMemberships,
			"monitoring_interval":                        instance.MonitoringInterval,
			"monitoring_role_arn":                        instance.MonitoringRoleArn,
			"performance_insights_enabled":               instance.PerformanceInsightsEnabled,
			"performance_insights_kms_key_id":            instance.PerformanceInsightsKMSKeyId,
			"performance_insights_retention_period":      instance.PerformanceInsightsRetentionPeriod,
			"enabled_cloudwatch_logs_exports":            instance.EnabledCloudwatchLogsExports,
			"deletion_protection":                        instance.DeletionProtection,
			"max_allocated_storage":                      instance.MaxAllocatedStorage,
			"customer_owned_ip_enabled":                  instance.CustomerOwnedIpEnabled,
			"auto_minor_version_upgrade":                 instance.AutoMinorVersionUpgrade,
			"license_model":                              instance.LicenseModel,
			"iops":                                       instance.Iops,
			"storage_throughput":                         instance.StorageThroughput,
			"db_instance_automated_backups_replications": instance.DBInstanceAutomatedBackupsReplications,
			"listener_endpoint":                          instance.ListenerEndpoint,
			"nchar_character_set_name":                   instance.NcharCharacterSetName,
			"character_set_name":                         instance.CharacterSetName,
			"secondary_availability_zone":                instance.SecondaryAvailabilityZone,
			"read_replica_source_db_instance_identifier": instance.ReadReplicaSourceDBInstanceIdentifier,
			"read_replica_db_instance_identifiers":       instance.ReadReplicaDBInstanceIdentifiers,
			"read_replica_db_cluster_identifiers":        instance.ReadReplicaDBClusterIdentifiers,
			"replica_mode":                               instance.ReplicaMode,
			"timezone":                                   instance.Timezone,
			"iam_database_authentication_enabled":        instance.IAMDatabaseAuthenticationEnabled,
			"pending_modified_values":                    instance.PendingModifiedValues,
			"latest_restorable_time":                     instance.LatestRestorableTime,
			"multi_tenant":                               instance.MultiTenant,
			"dedicated_log_volume":                       instance.DedicatedLogVolume,
			"certificate_details":                        instance.CertificateDetails,
		},
		Configuration: map[string]interface{}{
			"db_instance_class":            instance.DBInstanceClass,
			"engine":                       instance.Engine,
			"engine_version":               instance.EngineVersion,
			"allocated_storage":            instance.AllocatedStorage,
			"storage_type":                 instance.StorageType,
			"multi_az":                     instance.MultiAZ,
			"publicly_accessible":          instance.PubliclyAccessible,
			"storage_encrypted":            instance.StorageEncrypted,
			"backup_retention_period":      instance.BackupRetentionPeriod,
			"monitoring_interval":          instance.MonitoringInterval,
			"performance_insights_enabled": instance.PerformanceInsightsEnabled,
			"deletion_protection":          instance.DeletionProtection,
			"auto_minor_version_upgrade":   instance.AutoMinorVersionUpgrade,
			"license_model":                instance.LicenseModel,
			"iops":                         instance.Iops,
			"storage_throughput":           instance.StorageThroughput,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}
