package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockRoundTripper for testing HTTP requests
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// MockTokenSource for testing
type MockTokenSource struct{}

func (m *MockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: "mock-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}, nil
}

func TestNewGCPProviderComplete(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	assert.NotNil(t, provider)
	assert.Equal(t, "test-project", provider.projectID)
	assert.Equal(t, "us-central1", provider.region)
	assert.Equal(t, "us-central1-a", provider.zone)
	assert.NotNil(t, provider.httpClient)
	assert.NotEmpty(t, provider.baseURLs)
	assert.Equal(t, "https://compute.googleapis.com/compute/v1", provider.baseURLs["compute"])
	assert.Equal(t, "https://storage.googleapis.com/storage/v1", provider.baseURLs["storage"])
}

func TestGCPProviderComplete_Name(t *testing.T) {
	provider := NewGCPProviderComplete("test")
	assert.Equal(t, "gcp", provider.Name())
}

func TestGCPProviderComplete_Connect_ServiceAccount(t *testing.T) {
	// Create a temporary service account key file
	tempFile, err := ioutil.TempFile("", "gcp-key-*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	serviceAccountKey := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  "test-project",
		"private_key_id":              "key-id",
		"private_key":                 "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----\n",
		"client_email":                "test@test-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com",
	}

	keyData, _ := json.Marshal(serviceAccountKey)
	_, err = tempFile.Write(keyData)
	require.NoError(t, err)
	tempFile.Close()

	// Set environment variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tempFile.Name())
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	provider := NewGCPProviderComplete("")

	// Note: This will fail with actual authentication but we're testing the flow
	err = provider.Connect(context.Background())
	// We expect an error here because the test key is not valid
	assert.Error(t, err)
}

func TestGCPProviderComplete_makeAPIRequest(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	tests := []struct {
		name       string
		method     string
		url        string
		body       interface{}
		mockStatus int
		mockBody   string
		wantErr    bool
	}{
		{
			name:       "Successful GET request",
			method:     "GET",
			url:        "https://compute.googleapis.com/compute/v1/projects/test/zones/us-central1-a/instances/test",
			body:       nil,
			mockStatus: 200,
			mockBody:   `{"id":"test","name":"instance"}`,
			wantErr:    false,
		},
		{
			name:       "Successful POST request",
			method:     "POST",
			url:        "https://compute.googleapis.com/compute/v1/projects/test/zones/us-central1-a/instances",
			body:       map[string]string{"name": "new-instance"},
			mockStatus: 201,
			mockBody:   `{"status":"created"}`,
			wantErr:    false,
		},
		{
			name:       "API error response",
			method:     "GET",
			url:        "https://compute.googleapis.com/compute/v1/projects/test/zones/us-central1-a/instances/missing",
			body:       nil,
			mockStatus: 404,
			mockBody:   `{"error":{"code":404,"message":"Instance not found","status":"NOT_FOUND"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock HTTP client
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					// Verify authorization header exists
					authHeader := req.Header.Get("Authorization")
					assert.NotEmpty(t, authHeader)
					assert.Contains(t, authHeader, "Bearer")

					if tt.body != nil {
						assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					}

					return &http.Response{
						StatusCode: tt.mockStatus,
						Body:       io.NopCloser(strings.NewReader(tt.mockBody)),
					}, nil
					},
				},
			}

			data, err := provider.makeAPIRequest(context.Background(), tt.method, tt.url, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockBody, string(data))
			}
		})
	}
}

func TestGCPProviderComplete_SupportedResourceTypes(t *testing.T) {
	provider := NewGCPProviderComplete("test")
	types := provider.SupportedResourceTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "google_compute_instance")
	assert.Contains(t, types, "google_compute_network")
	assert.Contains(t, types, "google_storage_bucket")
	assert.Contains(t, types, "google_container_cluster")
	assert.Contains(t, types, "google_cloud_function")
}

func TestGCPProviderComplete_ListRegions(t *testing.T) {
	provider := NewGCPProviderComplete("test")
	regions, err := provider.ListRegions(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "us-central1")
	assert.Contains(t, regions, "europe-west1")
	assert.Contains(t, regions, "asia-east1")
}

func TestGCPProviderComplete_GetResource(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	tests := []struct {
		name         string
		resourceID   string
		mockResponse map[string]interface{}
		wantType     string
	}{
		{
			name:       "Get Compute Instance",
			resourceID: "test-instance",
			mockResponse: map[string]interface{}{
				"name":        "test-instance",
				"machineType": "zones/us-central1-a/machineTypes/n1-standard-1",
				"status":      "RUNNING",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"network": "global/networks/default",
					},
				},
				"disks": []interface{}{
					map[string]interface{}{
						"source": "zones/us-central1-a/disks/test-disk",
					},
				},
			},
			wantType: "google_compute_instance",
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
			// Since GetResource tries multiple resource types, it will match one
		})
	}
}

func TestGCPProviderComplete_GetResourceByType(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		mockResponse map[string]interface{}
		wantErr      bool
	}{
		{
			name:         "Get Compute Instance",
			resourceType: "google_compute_instance",
			resourceID:   "test-instance",
			mockResponse: map[string]interface{}{
				"name":        "test-instance",
				"machineType": "zones/us-central1-a/machineTypes/n1-standard-1",
				"status":      "RUNNING",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"network": "global/networks/default",
					},
				},
				"disks": []interface{}{},
			},
			wantErr: false,
		},
		{
			name:         "Get Storage Bucket",
			resourceType: "google_storage_bucket",
			resourceID:   "test-bucket",
			mockResponse: map[string]interface{}{
				"name":         "test-bucket",
				"location":     "US",
				"storageClass": "STANDARD",
				"versioning": map[string]interface{}{
					"enabled": true,
				},
			},
			wantErr: false,
		},
		{
			name:         "Get GKE Cluster",
			resourceType: "google_container_cluster",
			resourceID:   "test-cluster",
			mockResponse: map[string]interface{}{
				"name":                 "test-cluster",
				"location":             "us-central1",
				"initialNodeCount":     3,
				"status":               "RUNNING",
				"currentMasterVersion": "1.27.3-gke.100",
				"currentNodeVersion":   "1.27.3-gke.100",
			},
			wantErr: false,
		},
		{
			name:         "Unsupported Resource Type",
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
				assert.Equal(t, tt.resourceType, resource.Type)
				assert.Equal(t, tt.resourceID, resource.ID)
			}
		})
	}
}

func TestGCPProviderComplete_ValidateCredentials(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")

	// Without proper credentials, this will fail
	err := provider.ValidateCredentials(context.Background())
	assert.Error(t, err) // Expected to fail without real credentials
}

func TestGCPProviderComplete_DiscoverResources(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	resources, err := provider.DiscoverResources(context.Background(), "us-west1")
	assert.NoError(t, err)
	assert.NotNil(t, resources)
	// Currently returns empty list - would need implementation
	assert.Empty(t, resources)
	// Verify region was updated
	assert.Equal(t, "us-west1-a", provider.zone)
}

// Test specific resource getters
func TestGCPProviderComplete_getComputeInstance(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "instances")
			mockInstance := map[string]interface{}{
				"name":        "test-instance",
				"machineType": "zones/us-central1-a/machineTypes/n1-standard-1",
				"status":      "RUNNING",
				"zone":        "us-central1-a",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"network":       "global/networks/default",
						"networkIP":     "10.0.0.2",
						"accessConfigs": []interface{}{},
					},
				},
				"disks": []interface{}{
					map[string]interface{}{
						"source":    "zones/us-central1-a/disks/test-disk",
						"boot":      true,
						"autoDelete": true,
					},
				},
				"labels": map[string]string{
					"environment": "test",
				},
				"tags": map[string]interface{}{
					"items": []string{"http-server", "https-server"},
				},
				"creationTimestamp": "2024-01-01T00:00:00.000-07:00",
			}
			body, _ := json.Marshal(mockInstance)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getComputeInstance(context.Background(), "test-instance")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-instance", resource.ID)
	assert.Equal(t, "google_compute_instance", resource.Type)
	assert.Equal(t, "RUNNING", resource.Attributes["status"])
}

func TestGCPProviderComplete_getStorageBucket(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "/b/")
			mockBucket := map[string]interface{}{
				"name":         "test-bucket",
				"location":     "US",
				"storageClass": "STANDARD",
				"versioning": map[string]interface{}{
					"enabled": true,
				},
				"lifecycle": map[string]interface{}{
					"rule": []interface{}{
						map[string]interface{}{
							"action": map[string]interface{}{
								"type": "Delete",
							},
							"condition": map[string]interface{}{
								"age": 30,
							},
						},
					},
				},
				"labels": map[string]string{
					"environment": "test",
					"project":     "test-project",
				},
				"encryption": map[string]interface{}{
					"defaultKmsKeyName": "projects/test/locations/us/keyRings/test/cryptoKeys/test",
				},
				"iamConfiguration": map[string]interface{}{
					"uniformBucketLevelAccess": map[string]interface{}{
						"enabled": true,
					},
				},
				"timeCreated": "2024-01-01T00:00:00.000Z",
			}
			body, _ := json.Marshal(mockBucket)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getStorageBucket(context.Background(), "test-bucket")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-bucket", resource.ID)
	assert.Equal(t, "google_storage_bucket", resource.Type)
	assert.Equal(t, "US", resource.Attributes["location"])
	assert.Equal(t, "STANDARD", resource.Attributes["storage_class"])
}

func TestGCPProviderComplete_getGKECluster(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "clusters")
			mockCluster := map[string]interface{}{
				"name":             "test-cluster",
				"location":         "us-central1",
				"initialNodeCount": 3,
				"nodeConfig": map[string]interface{}{
					"machineType": "n1-standard-2",
					"diskSizeGb":  100,
					"diskType":    "pd-standard",
				},
				"masterAuth": map[string]interface{}{
					"clusterCaCertificate": "LS0tLS1CRUdJTi...",
				},
				"network":                "default",
				"subnetwork":             "default",
				"clusterIpv4Cidr":        "10.4.0.0/14",
				"servicesIpv4Cidr":       "10.8.0.0/20",
				"status":                 "RUNNING",
				"currentMasterVersion":   "1.27.3-gke.100",
				"currentNodeVersion":     "1.27.3-gke.100",
				"resourceLabels": map[string]string{
					"environment": "test",
				},
			}
			body, _ := json.Marshal(mockCluster)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getGKECluster(context.Background(), "test-cluster")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-cluster", resource.ID)
	assert.Equal(t, "google_container_cluster", resource.Type)
	assert.Equal(t, "RUNNING", resource.Attributes["status"])
	assert.Equal(t, "1.27.3-gke.100", resource.Attributes["current_master_version"])
}

func TestGCPProviderComplete_getSQLInstance(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "instances")
			mockSQL := map[string]interface{}{
				"name":            "test-sql",
				"databaseVersion": "MYSQL_8_0",
				"region":          "us-central1",
				"state":           "RUNNABLE",
				"settings": map[string]interface{}{
					"tier":            "db-n1-standard-1",
					"dataDiskSizeGb":  "100",
					"dataDiskType":    "PD_SSD",
					"availabilityType": "ZONAL",
					"backupConfiguration": map[string]interface{}{
						"enabled":   true,
						"startTime": "03:00",
					},
					"ipConfiguration": map[string]interface{}{
						"ipv4Enabled": true,
					},
				},
			}
			body, _ := json.Marshal(mockSQL)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getSQLInstance(context.Background(), "test-sql")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-sql", resource.ID)
	assert.Equal(t, "google_sql_database_instance", resource.Type)
	assert.Equal(t, "MYSQL_8_0", resource.Attributes["database_version"])
	assert.Equal(t, "RUNNABLE", resource.Attributes["state"])
}

func TestGCPProviderComplete_getPubSubTopic(t *testing.T) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "topics")
			mockTopic := map[string]interface{}{
				"name": "projects/test-project/topics/test-topic",
				"labels": map[string]string{
					"environment": "test",
				},
				"messageRetentionDuration": "604800s",
				"kmsKeyName":               "projects/test/locations/us/keyRings/test/cryptoKeys/test",
			}
			body, _ := json.Marshal(mockTopic)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getPubSubTopic(context.Background(), "test-topic")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-topic", resource.ID)
	assert.Equal(t, "google_pubsub_topic", resource.Type)
}

// Benchmark tests
func BenchmarkGCPProviderComplete_makeAPIRequest(b *testing.B) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			}, nil
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.makeAPIRequest(context.Background(), "GET", "https://compute.googleapis.com/compute/v1/projects/test/zones/us-central1-a/instances/test", nil)
	}
}

func BenchmarkGCPProviderComplete_GetResource(b *testing.B) {
	provider := NewGCPProviderComplete("test-project")
	provider.tokenSource = &MockTokenSource{}
	provider.httpClient = oauth2.NewClient(context.Background(), provider.tokenSource)

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			mockInstance := map[string]interface{}{
				"name":        "test-instance",
				"machineType": "zones/us-central1-a/machineTypes/n1-standard-1",
				"status":      "RUNNING",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"network": "global/networks/default",
					},
				},
				"disks": []interface{}{},
			}
			body, _ := json.Marshal(mockInstance)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GetResource(context.Background(), "test-instance")
	}
}