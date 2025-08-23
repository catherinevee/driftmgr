package discovery

import (
	"os"
	"testing"
)

func TestNewDigitalOceanDiscoverer(t *testing.T) {
	// Skip if no token is set
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping test")
	}

	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{
			name:    "Create discoverer with no region",
			region:  "",
			wantErr: false,
		},
		{
			name:    "Create discoverer with specific region",
			region:  "nyc3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer, err := NewDigitalOceanDiscoverer(tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDigitalOceanDiscoverer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && discoverer == nil {
				t.Error("NewDigitalOceanDiscoverer() returned nil discoverer")
			}
		})
	}
}

func TestDigitalOceanDiscoverer_IsAvailable(t *testing.T) {
	// Save current token
	originalToken := os.Getenv("DIGITALOCEAN_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("DIGITALOCEAN_TOKEN", originalToken)
		}
	}()

	tests := []struct {
		name      string
		setupFunc func()
		want      bool
	}{
		{
			name: "Token in DIGITALOCEAN_TOKEN",
			setupFunc: func() {
				os.Setenv("DIGITALOCEAN_TOKEN", "test-token")
				os.Unsetenv("DO_TOKEN")
				os.Unsetenv("DIGITALOCEAN_ACCESS_TOKEN")
			},
			want: true,
		},
		{
			name: "Token in DO_TOKEN",
			setupFunc: func() {
				os.Unsetenv("DIGITALOCEAN_TOKEN")
				os.Setenv("DO_TOKEN", "test-token")
				os.Unsetenv("DIGITALOCEAN_ACCESS_TOKEN")
			},
			want: true,
		},
		{
			name: "Token in DIGITALOCEAN_ACCESS_TOKEN",
			setupFunc: func() {
				os.Unsetenv("DIGITALOCEAN_TOKEN")
				os.Unsetenv("DO_TOKEN")
				os.Setenv("DIGITALOCEAN_ACCESS_TOKEN", "test-token")
			},
			want: true,
		},
		{
			name: "No token set",
			setupFunc: func() {
				os.Unsetenv("DIGITALOCEAN_TOKEN")
				os.Unsetenv("DO_TOKEN")
				os.Unsetenv("DIGITALOCEAN_ACCESS_TOKEN")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			// Create a discoverer (might fail if no token)
			discoverer, _ := NewDigitalOceanDiscoverer("")
			if discoverer == nil {
				// If we can't create discoverer, check directly
				got := CheckDigitalOceanCredentials()
				if got != tt.want {
					t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
				}
			} else {
				if got := discoverer.IsAvailable(); got != tt.want {
					t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestDigitalOceanDiscoverer_Discover(t *testing.T) {
	// Skip if no token is set
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		t.Skip("DIGITALOCEAN_TOKEN not set, skipping test")
	}

	discoverer, err := NewDigitalOceanDiscoverer("")
	if err != nil {
		t.Fatalf("Failed to create discoverer: %v", err)
	}

	resources, err := discoverer.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// We don't know what resources exist, just check it doesn't error
	t.Logf("Discovered %d resources", len(resources))

	// Check resource types if any exist
	resourceTypes := make(map[string]int)
	for _, r := range resources {
		resourceTypes[r.Type]++
	}

	for rType, count := range resourceTypes {
		t.Logf("  %s: %d", rType, count)
	}
}

func TestCheckDigitalOceanCredentials(t *testing.T) {
	// This test requires a valid token to actually verify
	token := os.Getenv("DIGITALOCEAN_TOKEN")

	if token == "" {
		// Test that it returns false with no token
		if CheckDigitalOceanCredentials() {
			t.Error("CheckDigitalOceanCredentials() should return false with no token")
		}
	} else {
		// Test that it returns true with a token (might still be invalid)
		// This is more of an integration test
		result := CheckDigitalOceanCredentials()
		t.Logf("CheckDigitalOceanCredentials() with token = %v", result)
	}
}
