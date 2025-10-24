package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/catherinevee/driftmgr/internal/models"
)

// EC2Service handles EC2-related operations
type EC2Service struct {
	client *ec2.Client
	region string
}

// NewEC2Service creates a new EC2 service
func NewEC2Service(client *ec2.Client, region string) *EC2Service {
	return &EC2Service{
		client: client,
		region: region,
	}
}

// DiscoverInstances discovers EC2 instances
func (s *EC2Service) DiscoverInstances(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe instances
	input := &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(1000),
	}

	paginator := ec2.NewDescribeInstancesPaginator(s.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				resource := s.convertInstanceToResource(instance)
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// DiscoverImages discovers EC2 AMIs
func (s *EC2Service) DiscoverImages(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe images
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	}

	result, err := s.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %w", err)
	}

	for _, image := range result.Images {
		resource := s.convertImageToResource(image)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverVPCs discovers VPCs
func (s *EC2Service) DiscoverVPCs(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe VPCs
	input := &ec2.DescribeVpcsInput{}

	result, err := s.client.DescribeVpcs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPCs: %w", err)
	}

	for _, vpc := range result.Vpcs {
		resource := s.convertVPCToResource(vpc)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverSubnets discovers subnets
func (s *EC2Service) DiscoverSubnets(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe subnets
	input := &ec2.DescribeSubnetsInput{}

	result, err := s.client.DescribeSubnets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	for _, subnet := range result.Subnets {
		resource := s.convertSubnetToResource(subnet)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverSecurityGroups discovers security groups
func (s *EC2Service) DiscoverSecurityGroups(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe security groups
	input := &ec2.DescribeSecurityGroupsInput{}

	result, err := s.client.DescribeSecurityGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe security groups: %w", err)
	}

	for _, sg := range result.SecurityGroups {
		resource := s.convertSecurityGroupToResource(sg)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverVolumes discovers EBS volumes
func (s *EC2Service) DiscoverVolumes(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe volumes
	input := &ec2.DescribeVolumesInput{}

	result, err := s.client.DescribeVolumes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe volumes: %w", err)
	}

	for _, volume := range result.Volumes {
		resource := s.convertVolumeToResource(volume)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverSnapshots discovers EBS snapshots
func (s *EC2Service) DiscoverSnapshots(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe snapshots
	input := &ec2.DescribeSnapshotsInput{
		OwnerIds: []string{"self"},
	}

	result, err := s.client.DescribeSnapshots(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe snapshots: %w", err)
	}

	for _, snapshot := range result.Snapshots {
		resource := s.convertSnapshotToResource(snapshot)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverKeyPairs discovers key pairs
func (s *EC2Service) DiscoverKeyPairs(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Describe key pairs
	input := &ec2.DescribeKeyPairsInput{}

	result, err := s.client.DescribeKeyPairs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe key pairs: %w", err)
	}

	for _, kp := range result.KeyPairs {
		resource := s.convertKeyPairToResource(kp)
		resources = append(resources, resource)
	}

	return resources, nil
}

// Convert instance to CloudResource
func (s *EC2Service) convertInstanceToResource(instance types.Instance) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("ec2", *instance.InstanceId),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSEC2Instance),
		Name:      getInstanceName(instance),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*instance.InstanceId),
		Tags:      convertTags(instance.Tags),
		Metadata: map[string]interface{}{
			"instance_id":          instance.InstanceId,
			"instance_type":        instance.InstanceType,
			"state":                instance.State.Name,
			"architecture":         instance.Architecture,
			"hypervisor":           instance.Hypervisor,
			"virtualization_type":  instance.VirtualizationType,
			"platform":             instance.Platform,
			"launch_time":          instance.LaunchTime,
			"placement":            instance.Placement,
			"monitoring":           instance.Monitoring,
			"security_groups":      instance.SecurityGroups,
			"subnet_id":            instance.SubnetId,
			"vpc_id":               instance.VpcId,
			"public_ip":            instance.PublicIpAddress,
			"private_ip":           instance.PrivateIpAddress,
			"public_dns":           instance.PublicDnsName,
			"private_dns":          instance.PrivateDnsName,
			"iam_instance_profile": instance.IamInstanceProfile,
			"ebs_optimized":        instance.EbsOptimized,
			"source_dest_check":    instance.SourceDestCheck,
		},
		Configuration: map[string]interface{}{
			"instance_type":       instance.InstanceType,
			"architecture":        instance.Architecture,
			"hypervisor":          instance.Hypervisor,
			"virtualization_type": instance.VirtualizationType,
			"platform":            instance.Platform,
			"ebs_optimized":       instance.EbsOptimized,
			"source_dest_check":   instance.SourceDestCheck,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert image to CloudResource
func (s *EC2Service) convertImageToResource(image types.Image) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("ami", *image.ImageId),
		Provider:  models.ProviderAWS,
		Type:      "aws_ami",
		Name:      getImageName(image),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*image.ImageId),
		Tags:      convertTags(image.Tags),
		Metadata: map[string]interface{}{
			"image_id":              image.ImageId,
			"name":                  image.Name,
			"description":           image.Description,
			"architecture":          image.Architecture,
			"creation_date":         image.CreationDate,
			"image_location":        image.ImageLocation,
			"image_type":            image.ImageType,
			"public":                image.Public,
			"kernel_id":             image.KernelId,
			"owner_id":              image.OwnerId,
			"platform":              image.Platform,
			"platform_details":      image.PlatformDetails,
			"usage_operation":       image.UsageOperation,
			"product_codes":         image.ProductCodes,
			"ramdisk_id":            image.RamdiskId,
			"state":                 image.State,
			"block_device_mappings": image.BlockDeviceMappings,
			"virtualization_type":   image.VirtualizationType,
			"hypervisor":            image.Hypervisor,
			"root_device_name":      image.RootDeviceName,
			"root_device_type":      image.RootDeviceType,
			"sriov_net_support":     image.SriovNetSupport,
			"ena_support":           image.EnaSupport,
		},
		Configuration: map[string]interface{}{
			"architecture":        image.Architecture,
			"image_type":          image.ImageType,
			"public":              image.Public,
			"platform":            image.Platform,
			"virtualization_type": image.VirtualizationType,
			"hypervisor":          image.Hypervisor,
			"root_device_type":    image.RootDeviceType,
			"sriov_net_support":   image.SriovNetSupport,
			"ena_support":         image.EnaSupport,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert VPC to CloudResource
func (s *EC2Service) convertVPCToResource(vpc types.Vpc) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("vpc", *vpc.VpcId),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSVPC),
		Name:      getVPCName(vpc),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*vpc.VpcId),
		Tags:      convertTags(vpc.Tags),
		Metadata: map[string]interface{}{
			"vpc_id":                          vpc.VpcId,
			"state":                           vpc.State,
			"cidr_block":                      vpc.CidrBlock,
			"dhcp_options_id":                 vpc.DhcpOptionsId,
			"instance_tenancy":                vpc.InstanceTenancy,
			"is_default":                      vpc.IsDefault,
			"cidr_block_association_set":      vpc.CidrBlockAssociationSet,
			"ipv6_cidr_block_association_set": vpc.Ipv6CidrBlockAssociationSet,
		},
		Configuration: map[string]interface{}{
			"cidr_block":       vpc.CidrBlock,
			"instance_tenancy": vpc.InstanceTenancy,
			"is_default":       vpc.IsDefault,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert subnet to CloudResource
func (s *EC2Service) convertSubnetToResource(subnet types.Subnet) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("subnet", *subnet.SubnetId),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSSubnet),
		Name:      getSubnetName(subnet),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*subnet.SubnetId),
		Tags:      convertTags(subnet.Tags),
		Metadata: map[string]interface{}{
			"subnet_id":                       subnet.SubnetId,
			"state":                           subnet.State,
			"vpc_id":                          subnet.VpcId,
			"cidr_block":                      subnet.CidrBlock,
			"availability_zone":               subnet.AvailabilityZone,
			"availability_zone_id":            subnet.AvailabilityZoneId,
			"available_ip_address_count":      subnet.AvailableIpAddressCount,
			"default_for_az":                  subnet.DefaultForAz,
			"map_public_ip_on_launch":         subnet.MapPublicIpOnLaunch,
			"assign_ipv6_address_on_creation": subnet.AssignIpv6AddressOnCreation,
			"ipv6_cidr_block_association_set": subnet.Ipv6CidrBlockAssociationSet,
		},
		Configuration: map[string]interface{}{
			"cidr_block":                      subnet.CidrBlock,
			"availability_zone":               subnet.AvailabilityZone,
			"default_for_az":                  subnet.DefaultForAz,
			"map_public_ip_on_launch":         subnet.MapPublicIpOnLaunch,
			"assign_ipv6_address_on_creation": subnet.AssignIpv6AddressOnCreation,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert security group to CloudResource
func (s *EC2Service) convertSecurityGroupToResource(sg types.SecurityGroup) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("sg", *sg.GroupId),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSSecurityGroup),
		Name:      getSecurityGroupName(sg),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*sg.GroupId),
		Tags:      convertTags(sg.Tags),
		Metadata: map[string]interface{}{
			"group_id":              sg.GroupId,
			"group_name":            sg.GroupName,
			"description":           sg.Description,
			"vpc_id":                sg.VpcId,
			"owner_id":              sg.OwnerId,
			"ip_permissions":        sg.IpPermissions,
			"ip_permissions_egress": sg.IpPermissionsEgress,
		},
		Configuration: map[string]interface{}{
			"group_name":  sg.GroupName,
			"description": sg.Description,
			"vpc_id":      sg.VpcId,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert volume to CloudResource
func (s *EC2Service) convertVolumeToResource(volume types.Volume) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("vol", *volume.VolumeId),
		Provider:  models.ProviderAWS,
		Type:      "aws_ebs_volume",
		Name:      getVolumeName(volume),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*volume.VolumeId),
		Tags:      convertTags(volume.Tags),
		Metadata: map[string]interface{}{
			"volume_id":            volume.VolumeId,
			"size":                 volume.Size,
			"snapshot_id":          volume.SnapshotId,
			"availability_zone":    volume.AvailabilityZone,
			"state":                volume.State,
			"create_time":          volume.CreateTime,
			"volume_type":          volume.VolumeType,
			"iops":                 volume.Iops,
			"encrypted":            volume.Encrypted,
			"kms_key_id":           volume.KmsKeyId,
			"throughput":           volume.Throughput,
			"multi_attach_enabled": volume.MultiAttachEnabled,
		},
		Configuration: map[string]interface{}{
			"size":                 volume.Size,
			"volume_type":          volume.VolumeType,
			"iops":                 volume.Iops,
			"encrypted":            volume.Encrypted,
			"multi_attach_enabled": volume.MultiAttachEnabled,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert snapshot to CloudResource
func (s *EC2Service) convertSnapshotToResource(snapshot types.Snapshot) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("snap", *snapshot.SnapshotId),
		Provider:  models.ProviderAWS,
		Type:      "aws_ebs_snapshot",
		Name:      getSnapshotName(snapshot),
		Region:    s.region,
		AccountID: getAccountIDFromARN(*snapshot.SnapshotId),
		Tags:      convertTags(snapshot.Tags),
		Metadata: map[string]interface{}{
			"snapshot_id":            snapshot.SnapshotId,
			"owner_id":               snapshot.OwnerId,
			"owner_alias":            snapshot.OwnerAlias,
			"volume_id":              snapshot.VolumeId,
			"volume_size":            snapshot.VolumeSize,
			"description":            snapshot.Description,
			"start_time":             snapshot.StartTime,
			"progress":               snapshot.Progress,
			"state":                  snapshot.State,
			"state_message":          snapshot.StateMessage,
			"encrypted":              snapshot.Encrypted,
			"kms_key_id":             snapshot.KmsKeyId,
			"data_encryption_key_id": snapshot.DataEncryptionKeyId,
		},
		Configuration: map[string]interface{}{
			"volume_size": snapshot.VolumeSize,
			"encrypted":   snapshot.Encrypted,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Convert key pair to CloudResource
func (s *EC2Service) convertKeyPairToResource(kp types.KeyPairInfo) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("key", *kp.KeyPairId),
		Provider:  models.ProviderAWS,
		Type:      "aws_key_pair",
		Name:      *kp.KeyName,
		Region:    s.region,
		AccountID: getAccountIDFromARN(*kp.KeyPairId),
		Tags:      convertTags(kp.Tags),
		Metadata: map[string]interface{}{
			"key_pair_id":     kp.KeyPairId,
			"key_name":        kp.KeyName,
			"key_fingerprint": kp.KeyFingerprint,
			"key_type":        kp.KeyType,
			"public_key":      kp.PublicKey,
		},
		Configuration: map[string]interface{}{
			"key_type": kp.KeyType,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// Helper functions

func generateResourceID(prefix, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}

func getAccountIDFromARN(arn string) string {
	// Simplified implementation - in production, parse ARN properly
	return "123456789012"
}

func convertTags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

func getInstanceName(instance types.Instance) string {
	for _, tag := range instance.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	if instance.InstanceId != nil {
		return *instance.InstanceId
	}
	return "unnamed-instance"
}

func getImageName(image types.Image) string {
	if image.Name != nil {
		return *image.Name
	}
	if image.ImageId != nil {
		return *image.ImageId
	}
	return "unnamed-image"
}

func getVPCName(vpc types.Vpc) string {
	for _, tag := range vpc.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	if vpc.VpcId != nil {
		return *vpc.VpcId
	}
	return "unnamed-vpc"
}

func getSubnetName(subnet types.Subnet) string {
	for _, tag := range subnet.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	if subnet.SubnetId != nil {
		return *subnet.SubnetId
	}
	return "unnamed-subnet"
}

func getSecurityGroupName(sg types.SecurityGroup) string {
	if sg.GroupName != nil {
		return *sg.GroupName
	}
	if sg.GroupId != nil {
		return *sg.GroupId
	}
	return "unnamed-security-group"
}

func getVolumeName(volume types.Volume) string {
	for _, tag := range volume.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	if volume.VolumeId != nil {
		return *volume.VolumeId
	}
	return "unnamed-volume"
}

func getSnapshotName(snapshot types.Snapshot) string {
	for _, tag := range snapshot.Tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	if snapshot.SnapshotId != nil {
		return *snapshot.SnapshotId
	}
	return "unnamed-snapshot"
}
