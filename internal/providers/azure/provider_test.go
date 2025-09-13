package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockRoundTripper for testing HTTP requests
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestNewAzureProviderComplete(t *testing.T) {
	provider := NewAzureProviderComplete("test-subscription", "test-rg")
	assert.NotNil(t, provider)
	assert.Equal(t, "test-subscription", provider.subscriptionID)
	assert.Equal(t, "test-rg", provider.resourceGroup)
	assert.NotNil(t, provider.httpClient)
	assert.Equal(t, "https://management.azure.com", provider.baseURL)
	assert.NotEmpty(t, provider.apiVersion)
}

func TestAzureProviderComplete_Name(t *testing.T) {
	provider := NewAzureProviderComplete("test", "test")
	assert.Equal(t, "azure", provider.Name())
}

func TestAzureProviderComplete_Connect_ServicePrincipal(t *testing.T) {
	// Set environment variables
	os.Setenv("AZURE_TENANT_ID", "test-tenant")
	os.Setenv("AZURE_CLIENT_ID", "test-client")
	os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("AZURE_TENANT_ID")
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
	}()

	provider := NewAzureProviderComplete("test-sub", "test-rg")

	// Mock HTTP client
	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				// Check it's a token request
				if strings.Contains(req.URL.String(), "oauth2/v2.0/token") {
					tokenResp := AzureTokenResponse{
						TokenType:   "Bearer",
						AccessToken: "test-access-token",
						ExpiresIn:   "3600",
					}
					body, _ := json.Marshal(tokenResp)
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request")
			},
		},
	}

	err := provider.Connect(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "test-access-token", provider.accessToken)
}

func TestAzureProviderComplete_Connect_ManagedIdentity(t *testing.T) {
	// Clear service principal env vars to trigger MI auth
	os.Unsetenv("AZURE_TENANT_ID")
	os.Unsetenv("AZURE_CLIENT_ID")
	os.Unsetenv("AZURE_CLIENT_SECRET")

	provider := NewAzureProviderComplete("test-sub", "test-rg")

	// Mock HTTP client for managed identity
	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "169.254.169.254") {
				tokenResp := struct {
					AccessToken string `json:"access_token"`
					ExpiresOn   string `json:"expires_on"`
				}{
					AccessToken: "mi-access-token",
					ExpiresOn:   "1234567890",
				}
				body, _ := json.Marshal(tokenResp)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			return nil, fmt.Errorf("unexpected request")
			},
		},
	}

	err := provider.Connect(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "mi-access-token", provider.accessToken)
}

func TestAzureProviderComplete_makeAPIRequest(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		mockStatus int
		mockBody   string
		wantErr    bool
	}{
		{
			name:       "Successful GET request",
			method:     "GET",
			path:       "/test/resource",
			body:       nil,
			mockStatus: 200,
			mockBody:   `{"id":"test","name":"resource"}`,
			wantErr:    false,
		},
		{
			name:       "Successful POST request",
			method:     "POST",
			path:       "/test/resource",
			body:       map[string]string{"key": "value"},
			mockStatus: 201,
			mockBody:   `{"status":"created"}`,
			wantErr:    false,
		},
		{
			name:       "API error response",
			method:     "GET",
			path:       "/test/resource",
			body:       nil,
			mockStatus: 404,
			mockBody:   `{"error":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					// Verify authorization header
					assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

					return &http.Response{
						StatusCode: tt.mockStatus,
						Body:       io.NopCloser(strings.NewReader(tt.mockBody)),
					}, nil
					},
				},
			}

			data, err := provider.makeAPIRequest(context.Background(), tt.method, tt.path, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockBody, string(data))
			}
		})
	}
}

func TestAzureProviderComplete_SupportedResourceTypes(t *testing.T) {
	provider := NewAzureProviderComplete("test", "test")
	types := provider.SupportedResourceTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "azurerm_virtual_machine")
	assert.Contains(t, types, "azurerm_virtual_network")
	assert.Contains(t, types, "azurerm_storage_account")
	assert.Contains(t, types, "azurerm_kubernetes_cluster")
}

func TestAzureProviderComplete_ListRegions(t *testing.T) {
	provider := NewAzureProviderComplete("test", "test")
	regions, err := provider.ListRegions(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "eastus")
	assert.Contains(t, regions, "westeurope")
}

func TestAzureProviderComplete_GetResource(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	tests := []struct {
		name       string
		resourceID string
		wantType   string
	}{
		{
			name:       "Virtual Machine ID",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			wantType:   "azurerm_virtual_machine",
		},
		{
			name:       "Virtual Network ID",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet1",
			wantType:   "azurerm_virtual_network",
		},
		{
			name:       "Storage Account ID",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/storage1",
			wantType:   "azurerm_storage_account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					var mockResponse map[string]interface{}
					if strings.Contains(tt.resourceID, "virtualMachines") {
						mockResponse = map[string]interface{}{
							"name":     "vm1",
							"location": "eastus",
							"properties": map[string]interface{}{
								"hardwareProfile": map[string]interface{}{
									"vmSize": "Standard_B2s",
								},
								"provisioningState": "Succeeded",
							},
						}
					} else if strings.Contains(tt.resourceID, "virtualNetworks") {
						mockResponse = map[string]interface{}{
							"name":     "vnet1",
							"location": "eastus",
							"properties": map[string]interface{}{
								"addressSpace": map[string]interface{}{
									"addressPrefixes": []string{"10.0.0.0/16"},
								},
								"provisioningState": "Succeeded",
							},
						}
					} else if strings.Contains(tt.resourceID, "storageAccounts") {
						mockResponse = map[string]interface{}{
							"name":     "storage1",
							"location": "eastus",
							"kind":     "StorageV2",
							"sku": map[string]interface{}{
								"name": "Standard_LRS",
								"tier": "Standard",
							},
							"properties": map[string]interface{}{
								"provisioningState": "Succeeded",
								"primaryEndpoints":  map[string]interface{}{},
							},
						}
					}

					body, _ := json.Marshal(mockResponse)
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
			assert.Equal(t, tt.wantType, resource.Type)
		})
	}
}

func TestAzureProviderComplete_GetResourceByType(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		wantErr      bool
	}{
		{
			name:         "Get Virtual Machine",
			resourceType: "azurerm_virtual_machine",
			resourceID:   "test-vm",
			wantErr:      false,
		},
		{
			name:         "Get Storage Account",
			resourceType: "azurerm_storage_account",
			resourceID:   "teststorage",
			wantErr:      false,
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
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					var mockResponse map[string]interface{}

					switch tt.resourceType {
					case "azurerm_virtual_machine":
						mockResponse = map[string]interface{}{
							"name":     tt.resourceID,
							"location": "eastus",
							"properties": map[string]interface{}{
								"hardwareProfile": map[string]interface{}{
									"vmSize": "Standard_B2s",
								},
								"provisioningState": "Succeeded",
							},
						}
					case "azurerm_storage_account":
						mockResponse = map[string]interface{}{
							"name":     tt.resourceID,
							"location": "eastus",
							"kind":     "StorageV2",
							"sku": map[string]interface{}{
								"name": "Standard_LRS",
								"tier": "Standard",
							},
							"properties": map[string]interface{}{
								"provisioningState": "Succeeded",
								"primaryEndpoints":  map[string]interface{}{},
							},
						}
					default:
						return &http.Response{
							StatusCode: 404,
							Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
						}, nil
					}

					body, _ := json.Marshal(mockResponse)
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
					},
				},
			}

			resource, err := provider.GetResourceByType(context.Background(), tt.resourceType, tt.resourceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)
				assert.Equal(t, tt.resourceType, resource.Type)
			}
		})
	}
}

func TestAzureProviderComplete_ListResources(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	tests := []struct {
		name         string
		resourceType string
		mockResponse interface{}
		expectCount  int
	}{
		{
			name:         "List Virtual Machines",
			resourceType: "azurerm_virtual_machine",
			mockResponse: struct {
				Value []map[string]interface{} `json:"value"`
			}{
				Value: []map[string]interface{}{
					{
						"name":     "vm1",
						"location": "eastus",
						"properties": map[string]interface{}{
							"hardwareProfile": map[string]interface{}{
								"vmSize": "Standard_B2s",
							},
						},
					},
					{
						"name":     "vm2",
						"location": "westus",
						"properties": map[string]interface{}{
							"hardwareProfile": map[string]interface{}{
								"vmSize": "Standard_D2s_v3",
							},
						},
					},
				},
			},
			expectCount: 2,
		},
		{
			name:         "List Storage Accounts",
			resourceType: "azurerm_storage_account",
			mockResponse: struct {
				Value []map[string]interface{} `json:"value"`
			}{
				Value: []map[string]interface{}{
					{
						"name":     "storage1",
						"location": "eastus",
						"kind":     "StorageV2",
						"sku": map[string]interface{}{
							"tier": "Standard",
						},
					},
				},
			},
			expectCount: 1,
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

			resources, err := provider.ListResources(context.Background(), tt.resourceType)
			assert.NoError(t, err)
			assert.Len(t, resources, tt.expectCount)
		})
	}
}

func TestAzureProviderComplete_ResourceExists(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		mockStatus   int
		expectExists bool
		wantErr      bool
	}{
		{
			name:         "Resource exists",
			resourceType: "azurerm_virtual_machine",
			resourceID:   "test-vm",
			mockStatus:   200,
			expectExists: true,
			wantErr:      false,
		},
		{
			name:         "Resource not found",
			resourceType: "azurerm_virtual_machine",
			resourceID:   "missing-vm",
			mockStatus:   404,
			expectExists: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.httpClient = &http.Client{
				Transport: &MockRoundTripper{
					RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockStatus == 200 {
						mockResponse := map[string]interface{}{
							"name":     tt.resourceID,
							"location": "eastus",
							"properties": map[string]interface{}{
								"hardwareProfile": map[string]interface{}{
									"vmSize": "Standard_B2s",
								},
								"provisioningState": "Succeeded",
							},
						}
						body, _ := json.Marshal(mockResponse)
						return &http.Response{
							StatusCode: tt.mockStatus,
							Body:       io.NopCloser(bytes.NewReader(body)),
						}, nil
					}
					return &http.Response{
						StatusCode: tt.mockStatus,
						Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"NotFound"}}`)),
					}, nil
					},
				},
			}

			exists, err := provider.ResourceExists(context.Background(), tt.resourceType, tt.resourceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExists, exists)
			}
		})
	}
}

func TestAzureProviderComplete_ValidateCredentials(t *testing.T) {
	os.Setenv("AZURE_TENANT_ID", "test-tenant")
	os.Setenv("AZURE_CLIENT_ID", "test-client")
	os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("AZURE_TENANT_ID")
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
	}()

	provider := NewAzureProviderComplete("test-sub", "test-rg")

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			tokenResp := AzureTokenResponse{
				TokenType:   "Bearer",
				AccessToken: "valid-token",
				ExpiresIn:   "3600",
			}
			body, _ := json.Marshal(tokenResp)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	err := provider.ValidateCredentials(context.Background())
	assert.NoError(t, err)
}

func TestAzureProviderComplete_DiscoverResources(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	resources, err := provider.DiscoverResources(context.Background(), "eastus")
	assert.NoError(t, err)
	assert.NotNil(t, resources)
	// Currently returns empty list - would need implementation
	assert.Empty(t, resources)
}

// Test specific resource getters
func TestAzureProviderComplete_getVirtualMachine(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "virtualMachines")
			mockVM := map[string]interface{}{
				"name":     "test-vm",
				"location": "eastus",
				"tags": map[string]string{
					"Environment": "Test",
				},
				"zones": []string{"1"},
				"properties": map[string]interface{}{
					"hardwareProfile": map[string]interface{}{
						"vmSize": "Standard_B2s",
					},
					"provisioningState": "Succeeded",
				},
			}
			body, _ := json.Marshal(mockVM)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getVirtualMachine(context.Background(), "test-vm")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-vm", resource.ID)
	assert.Equal(t, "azurerm_virtual_machine", resource.Type)
	assert.Equal(t, "Standard_B2s", resource.Attributes["vm_size"])
}

func TestAzureProviderComplete_getStorageAccount(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "storageAccounts")
			mockStorage := map[string]interface{}{
				"name":     "teststorage123",
				"location": "eastus",
				"kind":     "StorageV2",
				"sku": map[string]interface{}{
					"name": "Standard_LRS",
					"tier": "Standard",
				},
				"properties": map[string]interface{}{
					"provisioningState": "Succeeded",
					"primaryEndpoints": map[string]interface{}{
						"blob": "https://teststorage123.blob.core.windows.net/",
					},
				},
			}
			body, _ := json.Marshal(mockStorage)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getStorageAccount(context.Background(), "teststorage123")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "teststorage123", resource.ID)
	assert.Equal(t, "azurerm_storage_account", resource.Type)
	assert.Equal(t, "Standard", resource.Attributes["account_tier"])
	assert.Equal(t, "LRS", resource.Attributes["account_replication_type"])
}

func TestAzureProviderComplete_getKubernetesCluster(t *testing.T) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "managedClusters")
			mockAKS := map[string]interface{}{
				"name":     "test-aks",
				"location": "eastus",
				"properties": map[string]interface{}{
					"kubernetesVersion": "1.27.3",
					"dnsPrefix":         "test-aks-dns",
					"fqdn":              "test-aks-dns.hcp.eastus.azmk8s.io",
					"nodeResourceGroup": "MC_test-rg_test-aks_eastus",
					"enableRBAC":        true,
					"provisioningState": "Succeeded",
				},
			}
			body, _ := json.Marshal(mockAKS)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	resource, err := provider.getKubernetesCluster(context.Background(), "test-aks")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "test-aks", resource.ID)
	assert.Equal(t, "azurerm_kubernetes_cluster", resource.Type)
	assert.Equal(t, "1.27.3", resource.Attributes["kubernetes_version"])
	assert.Equal(t, true, resource.Attributes["enable_rbac"])
}

// Benchmark tests
func BenchmarkAzureProviderComplete_makeAPIRequest(b *testing.B) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

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
		_, _ = provider.makeAPIRequest(context.Background(), "GET", "/test", nil)
	}
}

func BenchmarkAzureProviderComplete_GetResource(b *testing.B) {
	provider := NewAzureProviderComplete("test-sub", "test-rg")
	provider.accessToken = "test-token"

	provider.httpClient = &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			mockVM := map[string]interface{}{
				"name":     "test-vm",
				"location": "eastus",
				"properties": map[string]interface{}{
					"hardwareProfile": map[string]interface{}{
						"vmSize": "Standard_B2s",
					},
					"provisioningState": "Succeeded",
				},
			}
			body, _ := json.Marshal(mockVM)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GetResource(context.Background(), "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1")
	}
}