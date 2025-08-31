package remediation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// AWSRemediator handles AWS-specific remediation
type AWSRemediator struct {
	ec2Client *ec2.Client
	iamClient *iam.Client
	rdsClient *rds.Client
	s3Client  *s3.Client
	cfg       aws.Config
}

// NewAWSRemediator creates a new AWS remediator
func NewAWSRemediator(ctx context.Context, region string) (*AWSRemediator, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSRemediator{
		ec2Client: ec2.NewFromConfig(cfg),
		iamClient: iam.NewFromConfig(cfg),
		rdsClient: rds.NewFromConfig(cfg),
		s3Client:  s3.NewFromConfig(cfg),
		cfg:       cfg,
	}, nil
}

// Remediate performs AWS resource remediation
func (r *AWSRemediator) Remediate(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	switch drift.ResourceType {
	case "aws_instance", "AWS::EC2::Instance":
		return r.remediateEC2Instance(ctx, drift, action)
	case "aws_security_group", "AWS::EC2::SecurityGroup":
		return r.remediateSecurityGroup(ctx, drift, action)
	case "aws_s3_bucket", "AWS::S3::Bucket":
		return r.remediateS3Bucket(ctx, drift, action)
	case "aws_rds_instance", "AWS::RDS::DBInstance":
		return r.remediateRDSInstance(ctx, drift, action)
	case "aws_iam_role", "AWS::IAM::Role":
		return r.remediateIAMRole(ctx, drift, action)
	case "aws_vpc", "AWS::EC2::VPC":
		return r.remediateVPC(ctx, drift, action)
	case "aws_subnet", "AWS::EC2::Subnet":
		return r.remediateSubnet(ctx, drift, action)
	default:
		return fmt.Errorf("remediation not implemented for resource type: %s", drift.ResourceType)
	}
}

// remediateEC2Instance handles EC2 instance remediation
func (r *AWSRemediator) remediateEC2Instance(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	instanceID := drift.ResourceID
	if !strings.HasPrefix(instanceID, "i-") {
		// Extract instance ID from ARN if necessary
		parts := strings.Split(instanceID, "/")
		if len(parts) > 0 {
			instanceID = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		// Update instance attributes based on drift
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			// Update tags
			var ec2Tags []types.Tag
			for key, value := range tags {
				ec2Tags = append(ec2Tags, types.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{instanceID},
				Tags:      ec2Tags,
			})
			if err != nil {
				return fmt.Errorf("failed to update EC2 instance tags: %w", err)
			}
		}

		// Update instance type if changed
		if instanceType, ok := action.Parameters["instance_type"].(string); ok {
			// First stop the instance
			_, err := r.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
				InstanceIds: []string{instanceID},
			})
			if err != nil {
				return fmt.Errorf("failed to stop instance: %w", err)
			}

			// Wait for instance to stop
			waiter := ec2.NewInstanceStoppedWaiter(r.ec2Client)
			err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{instanceID},
			}, 5*60) // 5 minutes timeout
			if err != nil {
				return fmt.Errorf("error waiting for instance to stop: %w", err)
			}

			// Modify instance attribute
			_, err = r.ec2Client.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
				InstanceId: aws.String(instanceID),
				InstanceType: &types.AttributeValue{
					Value: aws.String(instanceType),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to modify instance type: %w", err)
			}

			// Start the instance again
			_, err = r.ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
				InstanceIds: []string{instanceID},
			})
			if err != nil {
				return fmt.Errorf("failed to start instance: %w", err)
			}
		}

		return nil

	case "delete":
		// Terminate the instance
		_, err := r.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil {
			return fmt.Errorf("failed to terminate EC2 instance: %w", err)
		}
		return nil

	case "create":
		// Create instance based on expected configuration
		// This would need full instance configuration from drift.Details
		return fmt.Errorf("EC2 instance creation not implemented - requires full configuration")

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateSecurityGroup handles security group remediation
func (r *AWSRemediator) remediateSecurityGroup(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	sgID := drift.ResourceID
	if !strings.HasPrefix(sgID, "sg-") {
		// Extract SG ID from ARN if necessary
		parts := strings.Split(sgID, "/")
		if len(parts) > 0 {
			sgID = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		// Update security group rules
		if ingressRules, ok := action.Parameters["ingress"].([]interface{}); ok {
			// First, revoke existing rules
			describeResp, err := r.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
				GroupIds: []string{sgID},
			})
			if err != nil {
				return fmt.Errorf("failed to describe security group: %w", err)
			}

			if len(describeResp.SecurityGroups) > 0 {
				sg := describeResp.SecurityGroups[0]
				
				// Revoke existing ingress rules
				if len(sg.IpPermissions) > 0 {
					_, err = r.ec2Client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
						GroupId:       aws.String(sgID),
						IpPermissions: sg.IpPermissions,
					})
					if err != nil && !strings.Contains(err.Error(), "InvalidPermission.NotFound") {
						return fmt.Errorf("failed to revoke ingress rules: %w", err)
					}
				}

				// Add new ingress rules
				var newPermissions []types.IpPermission
				for _, rule := range ingressRules {
					if ruleMap, ok := rule.(map[string]interface{}); ok {
						permission := types.IpPermission{}
						
						if fromPort, ok := ruleMap["from_port"].(float64); ok {
							permission.FromPort = aws.Int32(int32(fromPort))
						}
						if toPort, ok := ruleMap["to_port"].(float64); ok {
							permission.ToPort = aws.Int32(int32(toPort))
						}
						if protocol, ok := ruleMap["protocol"].(string); ok {
							permission.IpProtocol = aws.String(protocol)
						}
						if cidr, ok := ruleMap["cidr_blocks"].([]interface{}); ok {
							for _, c := range cidr {
								permission.IpRanges = append(permission.IpRanges, types.IpRange{
									CidrIp: aws.String(fmt.Sprintf("%v", c)),
								})
							}
						}
						
						newPermissions = append(newPermissions, permission)
					}
				}

				if len(newPermissions) > 0 {
					_, err = r.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
						GroupId:       aws.String(sgID),
						IpPermissions: newPermissions,
					})
					if err != nil {
						return fmt.Errorf("failed to authorize ingress rules: %w", err)
					}
				}
			}
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			var ec2Tags []types.Tag
			for key, value := range tags {
				ec2Tags = append(ec2Tags, types.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{sgID},
				Tags:      ec2Tags,
			})
			if err != nil {
				return fmt.Errorf("failed to update security group tags: %w", err)
			}
		}

		return nil

	case "delete":
		// Delete the security group
		_, err := r.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})
		if err != nil {
			return fmt.Errorf("failed to delete security group: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateS3Bucket handles S3 bucket remediation
func (r *AWSRemediator) remediateS3Bucket(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	bucketName := drift.ResourceID
	// Remove any ARN prefix
	if strings.Contains(bucketName, ":::") {
		parts := strings.Split(bucketName, ":::")
		if len(parts) > 1 {
			bucketName = parts[1]
		}
	}

	switch action.Action {
	case "update":
		// Update bucket configuration
		if versioning, ok := action.Parameters["versioning"].(bool); ok {
			versioningConfig := &s3.PutBucketVersioningInput{
				Bucket: aws.String(bucketName),
				VersioningConfiguration: &s3types.VersioningConfiguration{
					Status: s3types.BucketVersioningStatusSuspended,
				},
			}
			if versioning {
				versioningConfig.VersioningConfiguration.Status = s3types.BucketVersioningStatusEnabled
			}
			
			_, err := r.s3Client.PutBucketVersioning(ctx, versioningConfig)
			if err != nil {
				return fmt.Errorf("failed to update bucket versioning: %w", err)
			}
		}

		// Update bucket encryption
		if encryption, ok := action.Parameters["encryption"].(map[string]interface{}); ok {
			if enabled, ok := encryption["enabled"].(bool); ok && enabled {
				_, err := r.s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
					Bucket: aws.String(bucketName),
					ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
						Rules: []s3types.ServerSideEncryptionRule{
							{
								ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
									SSEAlgorithm: s3types.ServerSideEncryptionAes256,
								},
							},
						},
					},
				})
				if err != nil {
					return fmt.Errorf("failed to update bucket encryption: %w", err)
				}
			}
		}

		// Update bucket tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			var s3Tags []s3types.Tag
			for key, value := range tags {
				s3Tags = append(s3Tags, s3types.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
				Bucket: aws.String(bucketName),
				Tagging: &s3types.Tagging{
					TagSet: s3Tags,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update bucket tags: %w", err)
			}
		}

		return nil

	case "delete":
		// Delete the bucket (must be empty)
		_, err := r.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("failed to delete S3 bucket: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateRDSInstance handles RDS instance remediation
func (r *AWSRemediator) remediateRDSInstance(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	dbInstanceID := drift.ResourceID
	// Extract DB instance identifier from ARN if necessary
	if strings.Contains(dbInstanceID, ":db:") {
		parts := strings.Split(dbInstanceID, ":")
		if len(parts) > 0 {
			dbInstanceID = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		modifyInput := &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbInstanceID),
			ApplyImmediately:     aws.Bool(false), // Apply during maintenance window
		}

		// Update instance class if changed
		if instanceClass, ok := action.Parameters["instance_class"].(string); ok {
			modifyInput.DBInstanceClass = aws.String(instanceClass)
		}

		// Update allocated storage
		if storage, ok := action.Parameters["allocated_storage"].(float64); ok {
			modifyInput.AllocatedStorage = aws.Int32(int32(storage))
		}

		// Update backup retention
		if retention, ok := action.Parameters["backup_retention_period"].(float64); ok {
			modifyInput.BackupRetentionPeriod = aws.Int32(int32(retention))
		}

		// Apply modifications
		_, err := r.rdsClient.ModifyDBInstance(ctx, modifyInput)
		if err != nil {
			return fmt.Errorf("failed to modify RDS instance: %w", err)
		}

		return nil

	case "delete":
		// Delete the RDS instance
		_, err := r.rdsClient.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbInstanceID),
			SkipFinalSnapshot:    aws.Bool(false),
			FinalDBSnapshotIdentifier: aws.String(fmt.Sprintf("%s-final-snapshot-%d", 
				dbInstanceID, 
				time.Now().Unix())),
		})
		if err != nil {
			return fmt.Errorf("failed to delete RDS instance: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateIAMRole handles IAM role remediation
func (r *AWSRemediator) remediateIAMRole(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	roleName := drift.ResourceID
	// Extract role name from ARN if necessary
	if strings.Contains(roleName, ":role/") {
		parts := strings.Split(roleName, "/")
		if len(parts) > 0 {
			roleName = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		// Update assume role policy
		if assumePolicy, ok := action.Parameters["assume_role_policy"].(string); ok {
			_, err := r.iamClient.UpdateAssumeRolePolicy(ctx, &iam.UpdateAssumeRolePolicyInput{
				RoleName:       aws.String(roleName),
				PolicyDocument: aws.String(assumePolicy),
			})
			if err != nil {
				return fmt.Errorf("failed to update assume role policy: %w", err)
			}
		}

		// Update role tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			var iamTags []iamtypes.Tag
			for key, value := range tags {
				iamTags = append(iamTags, iamtypes.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.iamClient.TagRole(ctx, &iam.TagRoleInput{
				RoleName: aws.String(roleName),
				Tags:     iamTags,
			})
			if err != nil {
				return fmt.Errorf("failed to update IAM role tags: %w", err)
			}
		}

		return nil

	case "delete":
		// First detach all policies
		listPoliciesResp, err := r.iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(roleName),
		})
		if err == nil {
			for _, policy := range listPoliciesResp.AttachedPolicies {
				_, _ = r.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
					RoleName:  aws.String(roleName),
					PolicyArn: policy.PolicyArn,
				})
			}
		}

		// Delete the role
		_, err = r.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return fmt.Errorf("failed to delete IAM role: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateVPC handles VPC remediation
func (r *AWSRemediator) remediateVPC(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	vpcID := drift.ResourceID
	if !strings.HasPrefix(vpcID, "vpc-") {
		// Extract VPC ID from ARN if necessary
		parts := strings.Split(vpcID, "/")
		if len(parts) > 0 {
			vpcID = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		// Update VPC attributes
		if enableDNS, ok := action.Parameters["enable_dns_support"].(bool); ok {
			_, err := r.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
				VpcId: aws.String(vpcID),
				EnableDnsSupport: &types.AttributeBooleanValue{
					Value: aws.Bool(enableDNS),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update VPC DNS support: %w", err)
			}
		}

		if enableDNSHostnames, ok := action.Parameters["enable_dns_hostnames"].(bool); ok {
			_, err := r.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
				VpcId: aws.String(vpcID),
				EnableDnsHostnames: &types.AttributeBooleanValue{
					Value: aws.Bool(enableDNSHostnames),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update VPC DNS hostnames: %w", err)
			}
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			var ec2Tags []types.Tag
			for key, value := range tags {
				ec2Tags = append(ec2Tags, types.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags:      ec2Tags,
			})
			if err != nil {
				return fmt.Errorf("failed to update VPC tags: %w", err)
			}
		}

		return nil

	case "delete":
		// Delete the VPC (must have no dependencies)
		_, err := r.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(vpcID),
		})
		if err != nil {
			return fmt.Errorf("failed to delete VPC: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateSubnet handles subnet remediation
func (r *AWSRemediator) remediateSubnet(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	subnetID := drift.ResourceID
	if !strings.HasPrefix(subnetID, "subnet-") {
		// Extract subnet ID from ARN if necessary
		parts := strings.Split(subnetID, "/")
		if len(parts) > 0 {
			subnetID = parts[len(parts)-1]
		}
	}

	switch action.Action {
	case "update":
		// Update subnet attributes
		if mapPublicIP, ok := action.Parameters["map_public_ip_on_launch"].(bool); ok {
			_, err := r.ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
				SubnetId: aws.String(subnetID),
				MapPublicIpOnLaunch: &types.AttributeBooleanValue{
					Value: aws.Bool(mapPublicIP),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update subnet public IP mapping: %w", err)
			}
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			var ec2Tags []types.Tag
			for key, value := range tags {
				ec2Tags = append(ec2Tags, types.Tag{
					Key:   aws.String(key),
					Value: aws.String(fmt.Sprintf("%v", value)),
				})
			}
			
			_, err := r.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{subnetID},
				Tags:      ec2Tags,
			})
			if err != nil {
				return fmt.Errorf("failed to update subnet tags: %w", err)
			}
		}

		return nil

	case "delete":
		// Delete the subnet
		_, err := r.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnetID),
		})
		if err != nil {
			return fmt.Errorf("failed to delete subnet: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}