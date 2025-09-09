package detector

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/providers/testprovider"
	"github.com/catherinevee/driftmgr/pkg/models"
)

func TestDriftDetector(t *testing.T) {
	// Create test provider with real data
	testResources := []models.Resource{
		{
			ID:       "instance-1",
			Type:     "instance",
			Provider: "test",
			Region:   "us-east-1",
			Name:     "web-server",
			Properties: map[string]interface{}{
				"instance_type": "t2.micro",
				"state":         "running",
			},
		},
		{
			ID:       "bucket-1",
			Type:     "bucket",
			Provider: "test",
			Region:   "us-east-1",
			Name:     "data-bucket",
			Properties: map[string]interface{}{
				"versioning": true,
			},
		},
	}

	provider := testprovider.NewTestProviderWithData(testResources)
	providers := map[string]providers.CloudProvider{
		"test": provider,
	}
	detector := NewDriftDetector(providers)

	t.Run("DetectResourceDrift", func(t *testing.T) {
		ctx := context.Background()

		result, err := detector.DetectResourceDrift(ctx, testResources[0])

		if err != nil {
			t.Fatalf("Failed to detect drift: %v", err)
		}

		if result == nil {
			t.Fatal("Expected drift result")
		}
	})
}
