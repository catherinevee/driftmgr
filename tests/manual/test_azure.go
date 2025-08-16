package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func main() {
	fmt.Println("Testing Azure discovery...")

	// Test Azure CLI availability
	cmd := exec.Command("az", "--version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Azure CLI not available: %v\n", err)
		return
	}
	fmt.Printf("Azure CLI version: %s\n", string(output))

	// Test Azure account
	cmd = exec.Command("az", "account", "show", "--query", "name", "--output", "tsv")
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("Azure account not accessible: %v\n", err)
		return
	}
	fmt.Printf("Azure account: %s\n", string(output))

	// Test VM discovery
	cmd = exec.Command("az", "vm", "list", "--query", "[].{id:id, name:name, location:location}", "--output", "json")
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("Failed to list VMs: %v\n", err)
		return
	}

	var vms []map[string]interface{}
	if err := json.Unmarshal(output, &vms); err != nil {
		fmt.Printf("Failed to parse VM list: %v\n", err)
		return
	}

	fmt.Printf("Found %d VMs:\n", len(vms))
	for i, vm := range vms {
		if i < 5 { // Only show first 5
			if name, ok := vm["name"].(string); ok {
				if location, ok := vm["location"].(string); ok {
					fmt.Printf("  - %s in %s\n", name, location)
				}
			}
		}
	}

	fmt.Println("Azure discovery test completed successfully!")
}
