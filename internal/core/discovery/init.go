package discovery

import (
	"context"
	"fmt"
)

// InitializeService creates and configures the discovery service with all providers
func InitializeService(ctx context.Context) (*Service, error) {
	service := NewService()
	
	// Register AWS provider
	awsProvider, err := NewAWSProvider()
	if err != nil {
		fmt.Printf("Warning: Could not initialize AWS provider: %v\n", err)
	} else {
		if err := service.RegisterProvider("aws", awsProvider); err != nil {
			fmt.Printf("Warning: Could not register AWS provider: %v\n", err)
		} else {
			fmt.Println("✓ AWS provider registered")
		}
	}
	
	// Register Azure provider
	azureProvider, err := NewAzureProvider()
	if err != nil {
		fmt.Printf("Warning: Could not initialize Azure provider: %v\n", err)
	} else {
		if err := service.RegisterProvider("azure", azureProvider); err != nil {
			fmt.Printf("Warning: Could not register Azure provider: %v\n", err)
		} else {
			fmt.Println("✓ Azure provider registered")
		}
	}
	
	// Register GCP provider
	gcpProvider, err := NewGCPProvider()
	if err != nil {
		fmt.Printf("Warning: Could not initialize GCP provider: %v\n", err)
	} else {
		if err := service.RegisterProvider("gcp", gcpProvider); err != nil {
			fmt.Printf("Warning: Could not register GCP provider: %v\n", err)
		} else {
			fmt.Println("✓ GCP provider registered")
		}
	}
	
	// Register DigitalOcean provider
	doProvider, err := NewDigitalOceanProvider()
	if err != nil {
		fmt.Printf("Warning: Could not initialize DigitalOcean provider: %v\n", err)
	} else {
		if err := service.RegisterProvider("digitalocean", doProvider); err != nil {
			fmt.Printf("Warning: Could not register DigitalOcean provider: %v\n", err)
		} else {
			fmt.Println("✓ DigitalOcean provider registered")
		}
	}
	
	return service, nil
}

// InitializeServiceSilent creates the service without output
func InitializeServiceSilent(ctx context.Context) (*Service, error) {
	service := NewService()
	
	// Try to register all providers silently
	if awsProvider, err := NewAWSProvider(); err == nil {
		service.RegisterProvider("aws", awsProvider)
	}
	
	if azureProvider, err := NewAzureProvider(); err == nil {
		service.RegisterProvider("azure", azureProvider)
	}
	
	if gcpProvider, err := NewGCPProvider(); err == nil {
		service.RegisterProvider("gcp", gcpProvider)
	}
	
	if doProvider, err := NewDigitalOceanProvider(); err == nil {
		service.RegisterProvider("digitalocean", doProvider)
	}
	
	return service, nil
}