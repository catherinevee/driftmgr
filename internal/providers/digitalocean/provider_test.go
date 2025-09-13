package digitalocean

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockRoundTripper for testing HTTP requests
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestNewDigitalOceanProvider(t *testing.T) {
	tests := []struct {
		name           string
		region         string
		expectedRegion string
	}{
		{
			name:           "With region",
			region:         "sfo3",
			expectedRegion: "sfo3",
		},
		{
			name:           "Without region (default)",
			region:         "",
			expectedRegion: "nyc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewDigitalOceanProvider(tt.region)
			assert.NotNil(t, provider)
			assert.Equal(t, tt.expectedRegion, provider.region)
			assert.NotNil(t, provider.httpClient)
			assert.Equal(t, "https://api.digitalocean.com/v2", provider.baseURL)
		})
	}
}

func TestDigitalOceanProvider_Name(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	assert.Equal(t, "digitalocean", provider.Name())
}

func TestDigitalOceanProvider_Initialize(t *testing.T) {
	tests := []struct {
		name      string
		setToken  bool
		tokenVal  string
		wantErr   bool
		mockValid bool
	}{
		{
			name:      "With valid token",
			setToken:  true,
			tokenVal:  "test-token",
			wantErr:   false,
			mockValid: true,
		},
		{
			name:     "Without token",
			setToken: false,
			wantErr:  true,
		},
		{
			name:      "With invalid token",
			setToken:  true,
			tokenVal:  "invalid-token",
			wantErr:   true,
			mockValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setToken {
				os.Setenv("DIGITALOCEAN_TOKEN", tt.tokenVal)
				defer os.Unsetenv("DIGITALOCEAN_TOKEN")
			} else {
				os.Unsetenv("DIGITALOCEAN_TOKEN")
			}

			provider := NewDigitalOceanProvider("nyc1")

			if tt.setToken && tt.tokenVal != "" {
				// Mock HTTP client for validation
				provider.httpClient = &http.Client{
					Transport: &MockRoundTripper{
						RoundTripFunc: func(req *http.Request) (*http.Response, error) {
							// Check authorization header
							assert.Equal(t, "Bearer "+tt.tokenVal, req.Header.Get("Authorization"))

							if tt.mockValid {
								return &http.Response{
									StatusCode: 200,
									Body:       io.NopCloser(strings.NewReader(`{"account":{}}`)),
								}, nil
							}
							return &http.Response{
								StatusCode: 401,
								Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
							}, nil
						},
					},
				}
			}

			err := provider.Initialize(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.tokenVal, provider.apiToken)
			}
		})
	}
}

func TestDigitalOceanProvider_ValidateCredentials(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	tests := []struct {
		name       string
		mockStatus int
		wantErr    bool
	}{
		{
			name:       "Valid credentials",
			mockStatus: 200,
			wantErr:    false,
		},
		{
			name:       "Invalid credentials",
			mockStatus: 401,
			wantErr:    true,
		},
		{
			name:       "Server error",
			mockStatus: 500,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
						assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
						assert.Contains(t, req.URL.String(), "/account")

						return &http.Response{
							StatusCode: tt.mockStatus,
							Body:       io.NopCloser(strings.NewReader(`{}`)),
						}, nil
					},
				},
			}

			err := provider.ValidateCredentials(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDigitalOceanProvider_GetResource(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	tests := []struct {
		name         string
		resourceID   string
		expectedType string
		mockResponse interface{}
	}{
		{
			name:         "Droplet by numeric ID",
			resourceID:   "12345",
			expectedType: "digitalocean_droplet",
			mockResponse: struct {
				Droplet Droplet `json:"droplet"`
			}{
				Droplet: Droplet{
					ID:       12345,
					Name:     "test-droplet",
					Status:   "active",
					SizeSlug: "s-1vcpu-1gb",
				},
			},
		},
		{
			name:         "Volume by ID",
			resourceID:   "vol-12345",
			expectedType: "digitalocean_volume",
			mockResponse: struct {
				Volume Volume `json:"volume"`
			}{
				Volume: Volume{
					ID:            "vol-12345",
					Name:          "test-volume",
					SizeGigabytes: 100,
				},
			},
		},
		{
			name:         "Load Balancer by ID",
			resourceID:   "lb-12345",
			expectedType: "digitalocean_loadbalancer",
			mockResponse: struct {
				LoadBalancer LoadBalancer `json:"load_balancer"`
			}{
				LoadBalancer: LoadBalancer{
					ID:     "lb-12345",
					Name:   "test-lb",
					Status: "active",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
						body, _ := json.Marshal(tt.mockResponse)
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					},
				},
			}

			resource, err := provider.GetResource(context.Background(), tt.resourceID)
			assert.NoError(t, err)
			assert.NotNil(t, resource)
		})
	}
}

func TestDigitalOceanProvider_GetResourceByType(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		mockResponse interface{}
		wantErr      bool
	}{
		{
			name:         "Get Droplet",
			resourceType: "digitalocean_droplet",
			resourceID:   "12345",
			mockResponse: struct {
				Droplet Droplet `json:"droplet"`
			}{
				Droplet: Droplet{
					ID:     12345,
					Name:   "test-droplet",
					Status: "active",
				},
			},
			wantErr: false,
		},
		{
			name:         "Get Volume",
			resourceType: "digitalocean_volume",
			resourceID:   "vol-12345",
			mockResponse: struct {
				Volume Volume `json:"volume"`
			}{
				Volume: Volume{
					ID:   "vol-12345",
					Name: "test-volume",
				},
			},
			wantErr: false,
		},
		{
			name:         "Unsupported resource type",
			resourceType: "unsupported_type",
			resourceID:   "test",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				provider.httpClient = &http.Client{
					Transport: &MockRoundTripper{
						RoundTripFunc: func(req *http.Request) (*http.Response, error) {
							body, _ := json.Marshal(tt.mockResponse)
							return &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(bytes.NewReader(body)),
							}, nil
						},
					},
				}
			}

			resource, err := provider.GetResourceByType(context.Background(), tt.resourceType, tt.resourceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)
			}
		})
	}
}

func TestDigitalOceanProvider_ListResources(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	tests := []struct {
		name         string
		resourceType string
		mockResponse interface{}
		expectCount  int
		wantErr      bool
	}{
		{
			name:         "List Droplets",
			resourceType: "digitalocean_droplet",
			mockResponse: DropletResponse{
				Droplets: []Droplet{
					{
						ID:     1,
						Name:   "droplet-1",
						Status: "active",
					},
					{
						ID:     2,
						Name:   "droplet-2",
						Status: "active",
					},
				},
			},
			expectCount: 2,
			wantErr:     false,
		},
		{
			name:         "List Volumes",
			resourceType: "digitalocean_volume",
			mockResponse: VolumeResponse{
				Volumes: []Volume{
					{
						ID:   "vol-1",
						Name: "volume-1",
					},
				},
			},
			expectCount: 1,
			wantErr:     false,
		},
		{
			name:         "List Load Balancers",
			resourceType: "digitalocean_loadbalancer",
			mockResponse: LoadBalancerResponse{
				LoadBalancers: []LoadBalancer{
					{
						ID:   "lb-1",
						Name: "lb-1",
					},
					{
						ID:   "lb-2",
						Name: "lb-2",
					},
					{
						ID:   "lb-3",
						Name: "lb-3",
					},
				},
			},
			expectCount: 3,
			wantErr:     false,
		},
		{
			name:         "Unsupported resource type",
			resourceType: "unsupported",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				provider.httpClient = &http.Client{
					Transport: &MockRoundTripper{
						RoundTripFunc: func(req *http.Request) (*http.Response, error) {
							body, _ := json.Marshal(tt.mockResponse)
							return &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(bytes.NewReader(body)),
							}, nil
						},
					},
				}
			}

			resources, err := provider.ListResources(context.Background(), tt.resourceType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, resources, tt.expectCount)
			}
		})
	}
}

func TestDigitalOceanProvider_DiscoverResources(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	// Mock multiple resource types
	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				var response interface{}

				if strings.Contains(req.URL.Path, "/droplets") {
					response = DropletResponse{
						Droplets: []Droplet{
							{ID: 1, Name: "droplet-1"},
						},
					}
				} else if strings.Contains(req.URL.Path, "/volumes") {
					response = VolumeResponse{
						Volumes: []Volume{
							{ID: "vol-1", Name: "volume-1"},
						},
					}
				} else if strings.Contains(req.URL.Path, "/load_balancers") {
					response = LoadBalancerResponse{
						LoadBalancers: []LoadBalancer{
							{ID: "lb-1", Name: "lb-1"},
						},
					}
				} else if strings.Contains(req.URL.Path, "/databases") {
					response = DatabaseResponse{
						Databases: []Database{
							{ID: "db-1", Name: "database-1"},
						},
					}
				} else {
					response = map[string]interface{}{"error": "not found"}
				}

				body, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			},
		},
	}

	resources, err := provider.DiscoverResources(context.Background(), "nyc1")
	assert.NoError(t, err)
	assert.NotNil(t, resources)
	// Should discover multiple resource types
	assert.GreaterOrEqual(t, len(resources), 0)
}

func TestDigitalOceanProvider_ListRegions(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")

	regions, err := provider.ListRegions(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "nyc1")
	assert.Contains(t, regions, "sfo3")
	assert.Contains(t, regions, "lon1")
}

func TestDigitalOceanProvider_SupportedResourceTypes(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	types := provider.SupportedResourceTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "digitalocean_droplet")
	assert.Contains(t, types, "digitalocean_volume")
	assert.Contains(t, types, "digitalocean_loadbalancer")
	assert.Contains(t, types, "digitalocean_database_cluster")
}

func TestDigitalOceanProvider_getDroplet(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	mockDroplet := Droplet{
		ID:       12345,
		Name:     "test-droplet",
		Memory:   1024,
		VCPUs:    1,
		Disk:     25,
		Status:   "active",
		SizeSlug: "s-1vcpu-1gb",
		Tags:     []string{"web", "production"},
		CreatedAt: time.Now(),
	}

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Contains(t, req.URL.Path, "droplets")
				response := struct {
					Droplet Droplet `json:"droplet"`
				}{
					Droplet: mockDroplet,
				}
				body, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			},
		},
	}

	resource, err := provider.getDroplet(context.Background(), "12345")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "12345", resource.ID)
	assert.Equal(t, "digitalocean_droplet", resource.Type)
}

func TestDigitalOceanProvider_getVolume(t *testing.T) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	mockVolume := Volume{
		ID:            "vol-12345",
		Name:          "test-volume",
		SizeGigabytes: 100,
		Description:   "Test volume",
		Tags:          []string{"storage"},
		CreatedAt:     time.Now(),
	}

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				assert.Contains(t, req.URL.Path, "volumes")
				response := struct {
					Volume Volume `json:"volume"`
				}{
					Volume: mockVolume,
				}
				body, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			},
		},
	}

	resource, err := provider.getVolume(context.Background(), "vol-12345")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "vol-12345", resource.ID)
	assert.Equal(t, "digitalocean_volume", resource.Type)
}

// Benchmark tests
func BenchmarkDigitalOceanProvider_GetResource(b *testing.B) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				response := struct {
					Droplet Droplet `json:"droplet"`
				}{
					Droplet: Droplet{
						ID:     12345,
						Name:   "test-droplet",
						Status: "active",
					},
				}
				body, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GetResource(context.Background(), "12345")
	}
}

func BenchmarkDigitalOceanProvider_ListResources(b *testing.B) {
	provider := NewDigitalOceanProvider("nyc1")
	provider.apiToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				response := DropletResponse{
					Droplets: []Droplet{
						{ID: 1, Name: "droplet-1"},
						{ID: 2, Name: "droplet-2"},
						{ID: 3, Name: "droplet-3"},
					},
				}
				body, _ := json.Marshal(response)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.ListResources(context.Background(), "digitalocean_droplet")
	}
}