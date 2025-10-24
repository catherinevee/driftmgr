package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/catherinevee/driftmgr/internal/models"
)

// CloudFormationService handles CloudFormation-related operations
type CloudFormationService struct {
	client *cloudformation.Client
	region string
}

// NewCloudFormationService creates a new CloudFormation service
func NewCloudFormationService(client *cloudformation.Client, region string) *CloudFormationService {
	return &CloudFormationService{
		client: client,
		region: region,
	}
}

// DiscoverStacks discovers CloudFormation stacks
func (s *CloudFormationService) DiscoverStacks(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List stacks
	result, err := s.client.ListStacks(ctx, &cloudformation.ListStacksInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}

	for _, stack := range result.StackSummaries {
		resource := s.convertStackToResource(stack)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertStackToResource converts a CloudFormation stack to a CloudResource
func (s *CloudFormationService) convertStackToResource(stack cloudformation.StackSummary) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("cf-stack", *stack.StackName),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSCloudFormation),
		Name:      *stack.StackName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the stack
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"stack_name":          stack.StackName,
			"stack_id":            stack.StackId,
			"stack_status":        stack.StackStatus,
			"stack_status_reason": stack.StackStatusReason,
			"creation_time":       stack.CreationTime,
			"deletion_time":       stack.DeletionTime,
			"last_updated_time":   stack.LastUpdatedTime,
			"parent_id":           stack.ParentId,
			"root_id":             stack.RootId,
			"drift_information":   stack.DriftInformation,
		},
		Configuration: map[string]interface{}{
			"stack_status":        stack.StackStatus,
			"stack_status_reason": stack.StackStatusReason,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}
