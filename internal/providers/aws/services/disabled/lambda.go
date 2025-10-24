package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/catherinevee/driftmgr/internal/models"
)

// LambdaService handles Lambda-related operations
type LambdaService struct {
	client *lambda.Client
	region string
}

// NewLambdaService creates a new Lambda service
func NewLambdaService(client *lambda.Client, region string) *LambdaService {
	return &LambdaService{
		client: client,
		region: region,
	}
}

// DiscoverFunctions discovers Lambda functions
func (s *LambdaService) DiscoverFunctions(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List functions
	result, err := s.client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	for _, function := range result.Functions {
		resource := s.convertFunctionToResource(function)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertFunctionToResource converts a Lambda function to a CloudResource
func (s *LambdaService) convertFunctionToResource(function lambda.FunctionConfiguration) models.CloudResource {
	resource := models.CloudResource{
		ID:        generateResourceID("lambda", *function.FunctionName),
		Provider:  models.ProviderAWS,
		Type:      string(models.ResourceTypeAWSLambdaFunction),
		Name:      *function.FunctionName,
		Region:    s.region,
		AccountID: "123456789012",          // This would be extracted from the function
		Tags:      make(map[string]string), // Tags would be fetched separately
		Metadata: map[string]interface{}{
			"function_name":                  function.FunctionName,
			"function_arn":                   function.FunctionArn,
			"runtime":                        function.Runtime,
			"role":                           function.Role,
			"handler":                        function.Handler,
			"code_size":                      function.CodeSize,
			"description":                    function.Description,
			"timeout":                        function.Timeout,
			"memory_size":                    function.MemorySize,
			"last_modified":                  function.LastModified,
			"code_sha256":                    function.CodeSha256,
			"version":                        function.Version,
			"vpc_config":                     function.VpcConfig,
			"dead_letter_config":             function.DeadLetterConfig,
			"environment":                    function.Environment,
			"kms_key_arn":                    function.KMSKeyArn,
			"tracing_config":                 function.TracingConfig,
			"master_arn":                     function.MasterArn,
			"revision_id":                    function.RevisionId,
			"layers":                         function.Layers,
			"state":                          function.State,
			"state_reason":                   function.StateReason,
			"state_reason_code":              function.StateReasonCode,
			"last_update_status":             function.LastUpdateStatus,
			"last_update_status_reason":      function.LastUpdateStatusReason,
			"last_update_status_reason_code": function.LastUpdateStatusReasonCode,
			"file_system_configs":            function.FileSystemConfigs,
			"package_type":                   function.PackageType,
			"image_config_response":          function.ImageConfigResponse,
			"signing_profile_version_arn":    function.SigningProfileVersionArn,
			"signing_job_arn":                function.SigningJobArn,
			"architectures":                  function.Architectures,
			"ephemeral_storage":              function.EphemeralStorage,
			"snap_start":                     function.SnapStart,
			"runtime_version_config":         function.RuntimeVersionConfig,
		},
		Configuration: map[string]interface{}{
			"runtime":                function.Runtime,
			"handler":                function.Handler,
			"timeout":                function.Timeout,
			"memory_size":            function.MemorySize,
			"vpc_config":             function.VpcConfig,
			"dead_letter_config":     function.DeadLetterConfig,
			"environment":            function.Environment,
			"tracing_config":         function.TracingConfig,
			"layers":                 function.Layers,
			"file_system_configs":    function.FileSystemConfigs,
			"package_type":           function.PackageType,
			"architectures":          function.Architectures,
			"ephemeral_storage":      function.EphemeralStorage,
			"snap_start":             function.SnapStart,
			"runtime_version_config": function.RuntimeVersionConfig,
		},
		LastDiscovered: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource
}
