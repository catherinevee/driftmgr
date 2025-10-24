package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/catherinevee/driftmgr/internal/models"
)

// S3Service handles S3-related operations
type S3Service struct {
	client *s3.Client
	region string
}

// NewS3Service creates a new S3 service
func NewS3Service(client *s3.Client, region string) *S3Service {
	return &S3Service{
		client: client,
		region: region,
	}
}

// DiscoverBuckets discovers S3 buckets
func (s *S3Service) DiscoverBuckets(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List buckets
	result, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	for _, bucket := range result.Buckets {
		resource := s.convertBucketToResource(bucket)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertBucketToResource converts an S3 bucket to a CloudResource
func (s *S3Service) convertBucketToResource(bucket types.Bucket) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("s3", *bucket.Name),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSS3Bucket),
		Name:      *bucket.Name,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the bucket
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"bucket_name":   bucket.Name,
			"creation_date": bucket.CreationDate,
		},
		Configuration: map[string]interface{}{
			"bucket_name": bucket.Name,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}
