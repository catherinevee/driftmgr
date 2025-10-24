package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/catherinevee/driftmgr/internal/models"
)

// IAMService handles IAM-related operations
type IAMService struct {
	client *iam.Client
	region string
}

// NewIAMService creates a new IAM service
func NewIAMService(client *iam.Client, region string) *IAMService {
	return &IAMService{
		client: client,
		region: region,
	}
}

// DiscoverRoles discovers IAM roles
func (s *IAMService) DiscoverRoles(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List roles
	result, err := s.client.ListRoles(ctx, &iam.ListRolesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	for _, role := range result.Roles {
		resource := s.convertRoleToResource(role)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverPolicies discovers IAM policies
func (s *IAMService) DiscoverPolicies(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List policies
	result, err := s.client.ListPolicies(ctx, &iam.ListPoliciesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	for _, policy := range result.Policies {
		resource := s.convertPolicyToResource(policy)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverUsers discovers IAM users
func (s *IAMService) DiscoverUsers(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List users
	result, err := s.client.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	for _, user := range result.Users {
		resource := s.convertUserToResource(user)
		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverGroups discovers IAM groups
func (s *IAMService) DiscoverGroups(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List groups
	result, err := s.client.ListGroups(ctx, &iam.ListGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	for _, group := range result.Groups {
		resource := s.convertGroupToResource(group)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertRoleToResource converts an IAM role to a CloudResource
func (s *IAMService) convertRoleToResource(role iam.Role) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("iam-role", *role.RoleName),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSIAMRole),
		Name:      *role.RoleName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the role
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"role_name":                   role.RoleName,
			"role_id":                     role.RoleId,
			"arn":                         role.Arn,
			"create_date":                 role.CreateDate,
			"assume_role_policy_document": role.AssumeRolePolicyDocument,
			"description":                 role.Description,
			"max_session_duration":        role.MaxSessionDuration,
			"permissions_boundary":        role.PermissionsBoundary,
			"tags":                        role.Tags,
			"role_last_used":              role.RoleLastUsed,
		},
		Configuration: map[string]interface{}{
			"assume_role_policy_document": role.AssumeRolePolicyDocument,
			"max_session_duration":        role.MaxSessionDuration,
			"permissions_boundary":        role.PermissionsBoundary,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// convertPolicyToResource converts an IAM policy to a CloudResource
func (s *IAMService) convertPolicyToResource(policy iam.Policy) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("iam-policy", *policy.PolicyName),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSIAMPolicy),
		Name:      *policy.PolicyName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the policy
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"policy_name":                      policy.PolicyName,
			"policy_id":                        policy.PolicyId,
			"arn":                              policy.Arn,
			"path":                             policy.Path,
			"default_version_id":               policy.DefaultVersionId,
			"attachment_count":                 policy.AttachmentCount,
			"permissions_boundary_usage_count": policy.PermissionsBoundaryUsageCount,
			"is_attachable":                    policy.IsAttachable,
			"description":                      policy.Description,
			"create_date":                      policy.CreateDate,
			"update_date":                      policy.UpdateDate,
			"tags":                             policy.Tags,
		},
		Configuration: map[string]interface{}{
			"path":               policy.Path,
			"default_version_id": policy.DefaultVersionId,
			"is_attachable":      policy.IsAttachable,
			"description":        policy.Description,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// convertUserToResource converts an IAM user to a CloudResource
func (s *IAMService) convertUserToResource(user iam.User) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("iam-user", *user.UserName),
		Provider:  models.ProviderAWS,
		Type:      "aws_iam_user",
		Name:      *user.UserName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the user
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"user_name":            user.UserName,
			"user_id":              user.UserId,
			"arn":                  user.Arn,
			"create_date":          user.CreateDate,
			"path":                 user.Path,
			"permissions_boundary": user.PermissionsBoundary,
			"tags":                 user.Tags,
			"password_last_used":   user.PasswordLastUsed,
		},
		Configuration: map[string]interface{}{
			"path":                 user.Path,
			"permissions_boundary": user.PermissionsBoundary,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}

// convertGroupToResource converts an IAM group to a CloudResource
func (s *IAMService) convertGroupToResource(group iam.Group) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("iam-group", *group.GroupName),
		Provider:  models.ProviderAWS,
		Type:      "aws_iam_group",
		Name:      *group.GroupName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the group
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"group_name":  group.GroupName,
			"group_id":    group.GroupId,
			"arn":         group.Arn,
			"create_date": group.CreateDate,
			"path":        group.Path,
		},
		Configuration: map[string]interface{}{
			"path": group.Path,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}
