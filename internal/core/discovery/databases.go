package discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"google.golang.org/api/sqladmin/v1"
	"google.golang.org/api/spanner/v1"
)

// DatabaseDiscovery handles database resource discovery across cloud providers
type DatabaseDiscovery struct {
	awsClient   *AWSDatabaseClient
	azureClient *AzureDatabaseClient
	gcpClient   *GCPDatabaseClient
}

// AWSDatabaseClient handles AWS database services
type AWSDatabaseClient struct {
	rdsClient         *rds.Client
	dynamoClient      *dynamodb.Client
	redshiftClient    *redshift.Client
	elasticacheClient *elasticache.Client
	docdbClient       *docdb.Client
	neptuneClient     *neptune.Client
}

// AzureDatabaseClient handles Azure database services
type AzureDatabaseClient struct {
	credential     *azidentity.DefaultAzureCredential
	subscriptionID string
	sqlClient      *armsql.ServersClient
	cosmosClient   *armcosmos.DatabaseAccountsClient
	mysqlClient    *armmysql.ServersClient
	postgresClient *armpostgresql.ServersClient
	redisClient    *armredis.Client
}

// GCPDatabaseClient handles GCP database services
type GCPDatabaseClient struct {
	projectID      string
	sqlAdminClient *sqladmin.Service
	spannerClient  *spanner.Service
}

// NewDatabaseDiscovery creates a new database discovery instance
func NewDatabaseDiscovery() (*DatabaseDiscovery, error) {
	awsClient, err := newAWSDatabaseClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS database client: %w", err)
	}

	azureClient, err := newAzureDatabaseClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure database client: %w", err)
	}

	gcpClient, err := newGCPDatabaseClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP database client: %w", err)
	}

	return &DatabaseDiscovery{
		awsClient:   awsClient,
		azureClient: azureClient,
		gcpClient:   gcpClient,
	}, nil
}

// newAWSDatabaseClient creates AWS database clients
func newAWSDatabaseClient() (*AWSDatabaseClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &AWSDatabaseClient{
		rdsClient:         rds.NewFromConfig(cfg),
		dynamoClient:      dynamodb.NewFromConfig(cfg),
		redshiftClient:    redshift.NewFromConfig(cfg),
		elasticacheClient: elasticache.NewFromConfig(cfg),
		docdbClient:       docdb.NewFromConfig(cfg),
		neptuneClient:     neptune.NewFromConfig(cfg),
	}, nil
}

// newAzureDatabaseClient creates Azure database clients
func newAzureDatabaseClient() (*AzureDatabaseClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	subscriptionID := getAzureSubscriptionID()
	if subscriptionID == "" {
		return nil, fmt.Errorf("Azure subscription ID not configured")
	}

	sqlClient, err := armsql.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	cosmosClient, err := armcosmos.NewDatabaseAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	mysqlClient, err := armmysql.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	postgresClient, err := armpostgresql.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	redisClient, err := armredis.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return &AzureDatabaseClient{
		credential:     cred,
		subscriptionID: subscriptionID,
		sqlClient:      sqlClient,
		cosmosClient:   cosmosClient,
		mysqlClient:    mysqlClient,
		postgresClient: postgresClient,
		redisClient:    redisClient,
	}, nil
}

// newGCPDatabaseClient creates GCP database clients
func newGCPDatabaseClient() (*GCPDatabaseClient, error) {
	ctx := context.Background()
	
	sqlAdminClient, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	spannerClient, err := spanner.NewService(ctx)
	if err != nil {
		return nil, err
	}

	projectID := getGCPProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID not configured")
	}

	return &GCPDatabaseClient{
		projectID:      projectID,
		sqlAdminClient: sqlAdminClient,
		spannerClient:  spannerClient,
	}, nil
}

// DiscoverAllDatabases discovers databases across all cloud providers
func (d *DatabaseDiscovery) DiscoverAllDatabases(ctx context.Context) ([]models.Resource, error) {
	var allResources []models.Resource

	// Discover AWS databases
	awsResources, err := d.discoverAWSDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("AWS database discovery failed: %w", err)
	}
	allResources = append(allResources, awsResources...)

	// Discover Azure databases
	azureResources, err := d.discoverAzureDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("Azure database discovery failed: %w", err)
	}
	allResources = append(allResources, azureResources...)

	// Discover GCP databases
	gcpResources, err := d.discoverGCPDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("GCP database discovery failed: %w", err)
	}
	allResources = append(allResources, gcpResources...)

	return allResources, nil
}

// discoverAWSDatabases discovers all AWS database resources
func (d *DatabaseDiscovery) discoverAWSDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover RDS instances
	rdsInstances, err := d.discoverRDSInstances(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, rdsInstances...)

	// Discover RDS clusters
	rdsClusters, err := d.discoverRDSClusters(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, rdsClusters...)

	// Discover DynamoDB tables
	dynamoTables, err := d.discoverDynamoDBTables(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, dynamoTables...)

	// Discover Redshift clusters
	redshiftClusters, err := d.discoverRedshiftClusters(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, redshiftClusters...)

	// Discover ElastiCache clusters
	elasticacheClusters, err := d.discoverElastiCacheClusters(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, elasticacheClusters...)

	// Discover DocumentDB clusters
	docdbClusters, err := d.discoverDocumentDBClusters(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, docdbClusters...)

	// Discover Neptune clusters
	neptuneClusters, err := d.discoverNeptuneClusters(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, neptuneClusters...)

	return resources, nil
}

// discoverRDSInstances discovers RDS database instances
func (d *DatabaseDiscovery) discoverRDSInstances(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}

	for _, db := range result.DBInstances {
		tags := make(map[string]interface{})
		
		// Get tags for the instance
		tagResult, err := d.awsClient.rdsClient.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: db.DBInstanceArn,
		})
		if err == nil {
			for _, tag := range tagResult.TagList {
				tags[*tag.Key] = *tag.Value
			}
		}

		state := "available"
		if db.DBInstanceStatus != nil {
			state = *db.DBInstanceStatus
		}

		var cost float64
		if db.DBInstanceClass != nil {
			// Estimate cost based on instance class
			cost = estimateRDSCost(*db.DBInstanceClass)
		}

		resources = append(resources, models.Resource{
			ID:       *db.DBInstanceIdentifier,
			Name:     *db.DBInstanceIdentifier,
			Type:     "aws_rds_instance",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*db.DBInstanceArn),
			State:    state,
			Tags:     tags,
			CreatedAt: *db.InstanceCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"cost": cost,
				"engine":             *db.Engine,
				"engine_version":     *db.EngineVersion,
				"instance_class":     *db.DBInstanceClass,
				"allocated_storage":  *db.AllocatedStorage,
				"multi_az":           *db.MultiAZ,
				"publicly_accessible": *db.PubliclyAccessible,
				"backup_retention":   *db.BackupRetentionPeriod,
				"encrypted":          *db.StorageEncrypted,
				"endpoint":           db.Endpoint.Address,
				"port":               *db.Endpoint.Port,
			},
		})
	}

	return resources, nil
}

// discoverRDSClusters discovers RDS clusters (Aurora)
func (d *DatabaseDiscovery) discoverRDSClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, err
	}

	for _, cluster := range result.DBClusters {
		tags := make(map[string]interface{})
		
		// Get tags for the cluster
		tagResult, err := d.awsClient.rdsClient.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: cluster.DBClusterArn,
		})
		if err == nil {
			for _, tag := range tagResult.TagList {
				tags[*tag.Key] = *tag.Value
			}
		}

		resources = append(resources, models.Resource{
			ID:       *cluster.DBClusterIdentifier,
			Name:     *cluster.DBClusterIdentifier,
			Type:     "aws_rds_cluster",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*cluster.DBClusterArn),
			State:    *cluster.Status,
			Tags:     tags,
			CreatedAt: *cluster.ClusterCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"engine":            *cluster.Engine,
				"engine_version":    *cluster.EngineVersion,
				"multi_az":          *cluster.MultiAZ,
				"backup_retention":  *cluster.BackupRetentionPeriod,
				"encrypted":         *cluster.StorageEncrypted,
				"endpoint":          *cluster.Endpoint,
				"reader_endpoint":   *cluster.ReaderEndpoint,
				"members":           len(cluster.DBClusterMembers),
			},
		})
	}

	return resources, nil
}

// discoverDynamoDBTables discovers DynamoDB tables
func (d *DatabaseDiscovery) discoverDynamoDBTables(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.dynamoClient.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, err
	}

	for _, tableName := range result.TableNames {
		// Get table details
		tableDesc, err := d.awsClient.dynamoClient.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: &tableName,
		})
		if err != nil {
			continue
		}

		table := tableDesc.Table
		tags := make(map[string]interface{})

		// Get tags
		tagResult, err := d.awsClient.dynamoClient.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{
			ResourceArn: table.TableArn,
		})
		if err == nil {
			for _, tag := range tagResult.Tags {
				tags[*tag.Key] = *tag.Value
			}
		}

		resources = append(resources, models.Resource{
			ID:       *table.TableName,
			Name:     *table.TableName,
			Type:     "aws_dynamodb_table",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*table.TableArn),
			State:    string(table.TableStatus),
			Tags:     tags,
			CreatedAt: *table.CreationDateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"billing_mode":     table.BillingModeSummary,
				"item_count":       *table.ItemCount,
				"table_size_bytes": *table.TableSizeBytes,
				"read_capacity":    table.ProvisionedThroughput.ReadCapacityUnits,
				"write_capacity":   table.ProvisionedThroughput.WriteCapacityUnits,
				"encryption":       table.SSEDescription != nil,
				"stream_enabled":   table.StreamSpecification != nil,
				"global_indexes":   len(table.GlobalSecondaryIndexes),
				"local_indexes":    len(table.LocalSecondaryIndexes),
			},
		})
	}

	return resources, nil
}

// discoverRedshiftClusters discovers Redshift clusters
func (d *DatabaseDiscovery) discoverRedshiftClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.redshiftClient.DescribeClusters(ctx, &redshift.DescribeClustersInput{})
	if err != nil {
		return nil, err
	}

	for _, cluster := range result.Clusters {
		tags := make(map[string]interface{})
		for _, tag := range cluster.Tags {
			tags[*tag.Key] = *tag.Value
		}

		resources = append(resources, models.Resource{
			ID:       *cluster.ClusterIdentifier,
			Name:     *cluster.ClusterIdentifier,
			Type:     "aws_redshift_cluster",
			Provider: "aws",
			Region:   extractRegionFromClusterEndpoint(cluster),
			State:    *cluster.ClusterStatus,
			Tags:     tags,
			CreatedAt: *cluster.ClusterCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"node_type":        *cluster.NodeType,
				"number_of_nodes":  *cluster.NumberOfNodes,
				"encrypted":        *cluster.Encrypted,
				"database_name":    *cluster.DBName,
				"endpoint":         cluster.Endpoint.Address,
				"port":             *cluster.Endpoint.Port,
				"version":          *cluster.ClusterVersion,
				"maintenance_track": *cluster.MaintenanceTrackName,
			},
		})
	}

	return resources, nil
}

// discoverElastiCacheClusters discovers ElastiCache clusters
func (d *DatabaseDiscovery) discoverElastiCacheClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover Redis clusters
	showNodeInfo := true
	redisResult, err := d.awsClient.elasticacheClient.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		ShowCacheNodeInfo: &showNodeInfo,
	})
	if err != nil {
		return nil, err
	}

	for _, cluster := range redisResult.CacheClusters {
		tags := make(map[string]interface{})
		
		// Get tags
		tagResult, err := d.awsClient.elasticacheClient.ListTagsForResource(ctx, &elasticache.ListTagsForResourceInput{
			ResourceName: cluster.ARN,
		})
		if err == nil && tagResult.TagList != nil {
			for _, tag := range tagResult.TagList {
				tags[*tag.Key] = *tag.Value
			}
		}

		resources = append(resources, models.Resource{
			ID:       *cluster.CacheClusterId,
			Name:     *cluster.CacheClusterId,
			Type:     "aws_elasticache_cluster",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*cluster.ARN),
			State:    *cluster.CacheClusterStatus,
			Tags:     tags,
			CreatedAt: *cluster.CacheClusterCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"engine":         *cluster.Engine,
				"engine_version": *cluster.EngineVersion,
				"node_type":      *cluster.CacheNodeType,
				"num_nodes":      *cluster.NumCacheNodes,
				"subnet_group":   *cluster.CacheSubnetGroupName,
			},
		})
	}

	return resources, nil
}

// discoverDocumentDBClusters discovers DocumentDB clusters
func (d *DatabaseDiscovery) discoverDocumentDBClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.docdbClient.DescribeDBClusters(ctx, &docdb.DescribeDBClustersInput{})
	if err != nil {
		return nil, err
	}

	for _, cluster := range result.DBClusters {
		tags := make(map[string]interface{})
		
		// Get tags
		tagResult, err := d.awsClient.docdbClient.ListTagsForResource(ctx, &docdb.ListTagsForResourceInput{
			ResourceName: cluster.DBClusterArn,
		})
		if err == nil {
			for _, tag := range tagResult.TagList {
				tags[*tag.Key] = *tag.Value
			}
		}

		resources = append(resources, models.Resource{
			ID:       *cluster.DBClusterIdentifier,
			Name:     *cluster.DBClusterIdentifier,
			Type:     "aws_documentdb_cluster",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*cluster.DBClusterArn),
			State:    *cluster.Status,
			Tags:     tags,
			CreatedAt: *cluster.ClusterCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"engine":           *cluster.Engine,
				"engine_version":   *cluster.EngineVersion,
				"backup_retention": *cluster.BackupRetentionPeriod,
				"encrypted":        *cluster.StorageEncrypted,
				"endpoint":         *cluster.Endpoint,
				"reader_endpoint":  *cluster.ReaderEndpoint,
			},
		})
	}

	return resources, nil
}

// discoverNeptuneClusters discovers Neptune graph database clusters
func (d *DatabaseDiscovery) discoverNeptuneClusters(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	result, err := d.awsClient.neptuneClient.DescribeDBClusters(ctx, &neptune.DescribeDBClustersInput{})
	if err != nil {
		return nil, err
	}

	for _, cluster := range result.DBClusters {
		tags := make(map[string]interface{})
		
		// Get tags
		tagResult, err := d.awsClient.neptuneClient.ListTagsForResource(ctx, &neptune.ListTagsForResourceInput{
			ResourceName: cluster.DBClusterArn,
		})
		if err == nil {
			for _, tag := range tagResult.TagList {
				tags[*tag.Key] = *tag.Value
			}
		}

		resources = append(resources, models.Resource{
			ID:       *cluster.DBClusterIdentifier,
			Name:     *cluster.DBClusterIdentifier,
			Type:     "aws_neptune_cluster",
			Provider: "aws",
			Region:   extractRegionFromDBARN(*cluster.DBClusterArn),
			State:    *cluster.Status,
			Tags:     tags,
			CreatedAt: *cluster.ClusterCreateTime,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"engine":           *cluster.Engine,
				"engine_version":   *cluster.EngineVersion,
				"backup_retention": *cluster.BackupRetentionPeriod,
				"encrypted":        *cluster.StorageEncrypted,
				"endpoint":         *cluster.Endpoint,
				"reader_endpoint":  *cluster.ReaderEndpoint,
			},
		})
	}

	return resources, nil
}

// discoverAzureDatabases discovers all Azure database resources
func (d *DatabaseDiscovery) discoverAzureDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover SQL databases
	sqlDatabases, err := d.discoverAzureSQLDatabases(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, sqlDatabases...)

	// Discover Cosmos DB accounts
	cosmosDatabases, err := d.discoverCosmosDBAccounts(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, cosmosDatabases...)

	// Discover MySQL servers
	mysqlServers, err := d.discoverAzureMySQLServers(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, mysqlServers...)

	// Discover PostgreSQL servers
	postgresServers, err := d.discoverAzurePostgreSQLServers(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, postgresServers...)

	// Discover Redis caches
	redisCaches, err := d.discoverAzureRedisCaches(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, redisCaches...)

	return resources, nil
}

// discoverAzureSQLDatabases discovers Azure SQL databases
func (d *DatabaseDiscovery) discoverAzureSQLDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := d.azureClient.sqlClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, server := range page.Value {
			tags := make(map[string]interface{})
			if server.Tags != nil {
				for k, v := range server.Tags {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:       *server.ID,
				Name:     *server.Name,
				Type:     "azure_sql_server",
				Provider: "azure",
				Region:   *server.Location,
				State:    string(*server.Properties.State),
				Tags:     tags,
				CreatedAt: time.Now(), // Azure doesn't provide creation time in this API
				Updated: time.Now(),
				Attributes: map[string]interface{}{
					"version":          *server.Properties.Version,
					"administrator":    *server.Properties.AdministratorLogin,
					"fqdn":             *server.Properties.FullyQualifiedDomainName,
					"public_access":    *server.Properties.PublicNetworkAccess,
				},
			})
		}
	}

	return resources, nil
}

// discoverCosmosDBAccounts discovers Cosmos DB accounts
func (d *DatabaseDiscovery) discoverCosmosDBAccounts(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := d.azureClient.cosmosClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, account := range page.Value {
			tags := make(map[string]interface{})
			if account.Tags != nil {
				for k, v := range account.Tags {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:       *account.ID,
				Name:     *account.Name,
				Type:     "azure_cosmosdb_account",
				Provider: "azure",
				Region:   *account.Location,
				State:    "active",
				Tags:     tags,
				CreatedAt: time.Now(),
				Updated: time.Now(),
				Attributes: map[string]interface{}{
					"kind":              string(*account.Kind),
					"consistency_level": string(*account.Properties.ConsistencyPolicy.DefaultConsistencyLevel),
					"enable_multiple_write": *account.Properties.EnableMultipleWriteLocations,
				},
			})
		}
	}

	return resources, nil
}

// discoverAzureMySQLServers discovers Azure MySQL servers
func (d *DatabaseDiscovery) discoverAzureMySQLServers(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := d.azureClient.mysqlClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, server := range page.Value {
			tags := make(map[string]interface{})
			if server.Tags != nil {
				for k, v := range server.Tags {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:       *server.ID,
				Name:     *server.Name,
				Type:     "azure_mysql_server",
				Provider: "azure",
				Region:   *server.Location,
				State:    string(*server.Properties.UserVisibleState),
				Tags:     tags,
				CreatedAt: time.Now(),
				Updated: time.Now(),
				Attributes: map[string]interface{}{
					"version":       string(*server.Properties.Version),
					"sku_name":      *server.SKU.Name,
					"sku_tier":      string(*server.SKU.Tier),
					"storage_mb":    *server.Properties.StorageProfile.StorageMB,
					"backup_retention": *server.Properties.StorageProfile.BackupRetentionDays,
				},
			})
		}
	}

	return resources, nil
}

// discoverAzurePostgreSQLServers discovers Azure PostgreSQL servers
func (d *DatabaseDiscovery) discoverAzurePostgreSQLServers(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := d.azureClient.postgresClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, server := range page.Value {
			tags := make(map[string]interface{})
			if server.Tags != nil {
				for k, v := range server.Tags {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:       *server.ID,
				Name:     *server.Name,
				Type:     "azure_postgresql_server",
				Provider: "azure",
				Region:   *server.Location,
				State:    string(*server.Properties.UserVisibleState),
				Tags:     tags,
				CreatedAt: time.Now(),
				Updated: time.Now(),
				Attributes: map[string]interface{}{
					"version":       string(*server.Properties.Version),
					"sku_name":      *server.SKU.Name,
					"sku_tier":      string(*server.SKU.Tier),
					"storage_mb":    *server.Properties.StorageProfile.StorageMB,
					"backup_retention": *server.Properties.StorageProfile.BackupRetentionDays,
				},
			})
		}
	}

	return resources, nil
}

// discoverAzureRedisCaches discovers Azure Redis caches
func (d *DatabaseDiscovery) discoverAzureRedisCaches(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := d.azureClient.redisClient.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cache := range page.Value {
			tags := make(map[string]interface{})
			if cache.Tags != nil {
				for k, v := range cache.Tags {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:       *cache.ID,
				Name:     *cache.Name,
				Type:     "azure_redis_cache",
				Provider: "azure",
				Region:   *cache.Location,
				State:    string(*cache.Properties.ProvisioningState),
				Tags:     tags,
				CreatedAt: time.Now(),
				Updated: time.Now(),
				Attributes: map[string]interface{}{
					"sku_name":     string(*cache.Properties.SKU.Name),
					"sku_family":   string(*cache.Properties.SKU.Family),
					"sku_capacity": *cache.Properties.SKU.Capacity,
					"hostname":     *cache.Properties.HostName,
					"port":         *cache.Properties.Port,
					"ssl_port":     *cache.Properties.SSLPort,
				},
			})
		}
	}

	return resources, nil
}

// discoverGCPDatabases discovers all GCP database resources
func (d *DatabaseDiscovery) discoverGCPDatabases(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover Cloud SQL instances
	sqlInstances, err := d.discoverCloudSQLInstances(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, sqlInstances...)

	// Discover Spanner instances
	spannerInstances, err := d.discoverSpannerInstances(ctx)
	if err != nil {
		return nil, err
	}
	resources = append(resources, spannerInstances...)

	return resources, nil
}

// discoverCloudSQLInstances discovers GCP Cloud SQL instances
func (d *DatabaseDiscovery) discoverCloudSQLInstances(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	instances, err := d.gcpClient.sqlAdminClient.Instances.List(d.gcpClient.projectID).Do()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances.Items {
		tags := make(map[string]interface{})
		for k, v := range instance.Settings.UserLabels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       instance.Name,
			Name:     instance.Name,
			Type:     "gcp_cloud_sql_instance",
			Provider: "gcp",
			Region:   instance.Region,
			State:    instance.State,
			Tags:     tags,
			CreatedAt: parseGCPTimestamp(instance.CreateTime),
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"database_version": instance.DatabaseVersion,
				"tier":             instance.Settings.Tier,
				"disk_size":        instance.Settings.DataDiskSizeGb,
				"disk_type":        instance.Settings.DataDiskType,
				"backup_enabled":   instance.Settings.BackupConfiguration.Enabled,
				"ip_addresses":     instance.IpAddresses,
			},
		})
	}

	return resources, nil
}

// discoverSpannerInstances discovers GCP Spanner instances
func (d *DatabaseDiscovery) discoverSpannerInstances(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	parent := fmt.Sprintf("projects/%s", d.gcpClient.projectID)
	instances, err := d.gcpClient.spannerClient.Projects.Instances.List(parent).Do()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances.Instances {
		tags := make(map[string]interface{})
		for k, v := range instance.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       instance.Name,
			Name:     extractResourceName(instance.Name),
			Type:     "gcp_spanner_instance",
			Provider: "gcp",
			Region:   extractLocationFromConfig(instance.Config),
			State:    instance.State,
			Tags:     tags,
			CreatedAt: parseGCPTimestamp(instance.CreateTime),
			Updated: parseGCPTimestamp(instance.UpdateTime),
			Attributes: map[string]interface{}{
				"config":      instance.Config,
				"node_count":  instance.NodeCount,
				"processing_units": instance.ProcessingUnits,
				"display_name": instance.DisplayName,
			},
		})
	}

	return resources, nil
}

// Helper functions

func extractRegionFromDBARN(arn string) string {
	// ARN format: arn:aws:service:region:account:resource
	parts := strings.Split(arn, ":")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "unknown"
}

func estimateRDSCost(instanceClass string) float64 {
	// Simplified cost estimation based on instance class
	// In production, this would use AWS Pricing API
	costMap := map[string]float64{
		"db.t3.micro":   0.017,
		"db.t3.small":   0.034,
		"db.t3.medium":  0.068,
		"db.t3.large":   0.136,
		"db.m5.large":   0.171,
		"db.m5.xlarge":  0.342,
		"db.m5.2xlarge": 0.684,
		"db.r5.large":   0.240,
		"db.r5.xlarge":  0.480,
	}

	if cost, ok := costMap[instanceClass]; ok {
		return cost * 24 * 30 // Monthly cost estimate
	}
	return 100.0 // Default estimate
}

func extractRegionFromClusterEndpoint(cluster interface{}) string {
	// Since the cluster type is from AWS SDK and may vary, we'll simplify
	// In real implementation this would use the actual redshift.Cluster type
	return "us-east-1" // Default region
}

func extractLocationFromConfig(config string) string {
	// Extract location from Spanner config
	// Format: projects/PROJECT/instanceConfigs/LOCATION
	parts := strings.Split(config, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "unknown"
}

func extractResourceName(fullName string) string {
	// Extract resource name from full path
	// Format: projects/PROJECT/instances/NAME
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

func parseGCPTimestamp(timestamp string) time.Time {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return time.Now()
	}
	return t
}