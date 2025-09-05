package detector

import (
	"context"
	"fmt"
	"testing"

	"github.com/catherinevee/driftmgr/internal/state/parser"
	"github.com/catherinevee/driftmgr/pkg/models"
)

func BenchmarkDriftDetection(b *testing.B) {
	// Create mock provider
	provider := &mockProvider{
		resources: generateMockResources(100),
	}
	
	// Create mock state
	state := generateMockState(100)
	
	// Create detector
	detector := New(provider, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectDrift(ctx, state)
	}
}

func BenchmarkDriftDetectionQuickMode(b *testing.B) {
	provider := &mockProvider{
		resources: generateMockResources(1000),
	}
	
	state := generateMockState(1000)
	detector := New(provider, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectDriftWithMode(ctx, state, QuickMode)
	}
}

func BenchmarkDriftDetectionDeepMode(b *testing.B) {
	provider := &mockProvider{
		resources: generateMockResources(100),
	}
	
	state := generateMockState(100)
	detector := New(provider, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectDriftWithMode(ctx, state, DeepMode)
	}
}

func BenchmarkDriftDetectionSmartMode(b *testing.B) {
	provider := &mockProvider{
		resources: generateMockResources(500),
	}
	
	state := generateMockState(500)
	detector := New(provider, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectDriftWithMode(ctx, state, SmartMode)
	}
}

func BenchmarkParallelDriftDetection(b *testing.B) {
	provider := &mockProvider{
		resources: generateMockResources(1000),
	}
	
	state := generateMockState(1000)
	detector := New(provider, nil)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = detector.DetectDrift(ctx, state)
		}
	})
}

func BenchmarkResourceComparison(b *testing.B) {
	stateResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami":          "ami-12345",
			"tags": map[string]string{
				"Name": "Test Instance",
				"Env":  "test",
			},
		},
	}
	
	actualResource := &models.Resource{
		ID:   "test-resource",
		Type: "aws_instance",
		Name: "test",
		Attributes: map[string]interface{}{
			"instance_type": "t2.small", // Changed
			"ami":          "ami-12345",
			"tags": map[string]string{
				"Name": "Test Instance",
				"Env":  "production", // Changed
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = compareResources(stateResource, actualResource)
	}
}

func BenchmarkLargeStateProcessing(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Resources_%d", size), func(b *testing.B) {
			provider := &mockProvider{
				resources: generateMockResources(size),
			}
			
			state := generateMockState(size)
			detector := New(provider, nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = detector.DetectDrift(ctx, state)
			}
		})
	}
}

// Helper functions for benchmarks

type mockProvider struct {
	resources []*models.Resource
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockProvider) DiscoverResources(ctx context.Context) ([]*models.Resource, error) {
	return m.resources, nil
}

func (m *mockProvider) GetResource(ctx context.Context, id string) (*models.Resource, error) {
	for _, r := range m.resources {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("resource not found")
}

func (m *mockProvider) Validate() error {
	return nil
}

func (m *mockProvider) TestConnection() error {
	return nil
}

func generateMockResources(count int) []*models.Resource {
	resources := make([]*models.Resource, count)
	for i := 0; i < count; i++ {
		resources[i] = &models.Resource{
			ID:   fmt.Sprintf("resource-%d", i),
			Type: "aws_instance",
			Name: fmt.Sprintf("instance-%d", i),
			Attributes: map[string]interface{}{
				"instance_type": "t2.micro",
				"ami":          fmt.Sprintf("ami-%d", i),
			},
		}
	}
	return resources
}

func generateMockState(count int) *parser.TerraformState {
	resources := make([]parser.Resource, count)
	for i := 0; i < count; i++ {
		resources[i] = parser.Resource{
			Type: "aws_instance",
			Name: fmt.Sprintf("instance-%d", i),
			Instances: []parser.Instance{
				{
					Attributes: map[string]interface{}{
						"id":            fmt.Sprintf("resource-%d", i),
						"instance_type": "t2.micro",
						"ami":          fmt.Sprintf("ami-%d", i),
					},
				},
			},
		}
	}
	
	return &parser.TerraformState{
		Version:   4,
		Resources: resources,
	}
}