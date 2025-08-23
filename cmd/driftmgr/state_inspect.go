package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// handleStateInspect handles the "state inspect" command
func handleStateInspect(args []string) {
	var statePath string
	var format string = "summary"
	var showSensitive bool = false

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state", "-s":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--show-sensitive":
			showSensitive = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr state inspect [flags]")
			fmt.Println()
			fmt.Println("Inspect and display Terraform state file contents")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --state, -s string    Path to state file (required)")
			fmt.Println("  --format, -f string   Output format: summary, json, detailed (default: summary)")
			fmt.Println("  --show-sensitive      Show sensitive attribute values")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr state inspect --state terraform.tfstate")
			fmt.Println("  driftmgr state inspect --state s3://bucket/key --format json")
			fmt.Println("  driftmgr state inspect --state azure-test.tfstate --format detailed")
			return
		}
	}

	if statePath == "" {
		// Try to find state file in current directory
		if _, err := os.Stat("terraform.tfstate"); err == nil {
			statePath = "terraform.tfstate"
		} else {
			fmt.Println("Error: State file path required. Use --state flag or have terraform.tfstate in current directory")
			fmt.Println("Run 'driftmgr state inspect --help' for usage")
			os.Exit(1)
		}
	}

	// Load the state file
	loader := state.NewStateLoader(statePath)
	ctx := context.Background()
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state file: %v\n", err)
		os.Exit(1)
	}

	// Display based on format
	switch format {
	case "json":
		displayStateJSON(stateFile, showSensitive)
	case "detailed":
		displayStateDetailed(stateFile, showSensitive)
	default: // summary
		displayStateSummary(stateFile)
	}
}

func displayStateSummary(stateFile interface{}) {
	// Type assertion to get the actual state structure
	stateData, _ := json.Marshal(stateFile)
	var state map[string]interface{}
	json.Unmarshal(stateData, &state)

	fmt.Println("========================================")
	fmt.Println("TERRAFORM STATE FILE SUMMARY")
	fmt.Println("========================================")

	// Basic info
	if path, ok := state["path"].(string); ok && path != "" {
		fmt.Printf("Path: %s\n", path)
	}
	if version, ok := state["version"].(float64); ok {
		fmt.Printf("State Version: %v\n", int(version))
	}
	if tfVersion, ok := state["terraform_version"].(string); ok && tfVersion != "" {
		fmt.Printf("Terraform Version: %s\n", tfVersion)
	}
	if serial, ok := state["serial"].(float64); ok {
		fmt.Printf("Serial: %v\n", int(serial))
	}
	if lineage, ok := state["lineage"].(string); ok && lineage != "" {
		fmt.Printf("Lineage: %s\n", lineage)
	}

	fmt.Println()

	// Resources summary
	if resources, ok := state["resources"].([]interface{}); ok {
		fmt.Printf("Total Resources: %d\n", len(resources))

		// Count by type
		typeCounts := make(map[string]int)
		providerCounts := make(map[string]int)

		for _, r := range resources {
			if resource, ok := r.(map[string]interface{}); ok {
				if resType, ok := resource["type"].(string); ok {
					typeCounts[resType]++
				}
				if provider, ok := resource["provider"].(string); ok {
					// Extract provider name from full path
					parts := strings.Split(provider, "/")
					providerName := parts[len(parts)-1]
					providerName = strings.TrimSuffix(providerName, "]")
					providerCounts[providerName]++
				}
			}
		}

		if len(providerCounts) > 0 {
			fmt.Println("\nResources by Provider:")
			for provider, count := range providerCounts {
				fmt.Printf("  - %s: %d\n", provider, count)
			}
		}

		if len(typeCounts) > 0 {
			fmt.Println("\nResources by Type:")
			for resType, count := range typeCounts {
				fmt.Printf("  - %s: %d\n", resType, count)
			}
		}

		// List resources
		fmt.Println("\nResource List:")
		for i, r := range resources {
			if resource, ok := r.(map[string]interface{}); ok {
				resType := resource["type"].(string)
				resName := resource["name"].(string)
				mode := resource["mode"].(string)

				fmt.Printf("%d. [%s] %s.%s", i+1, strings.ToUpper(mode), resType, resName)

				// Show instance count
				if instances, ok := resource["instances"].([]interface{}); ok {
					fmt.Printf(" (%d instance%s)", len(instances), plural(len(instances)))
				}
				fmt.Println()
			}
		}
	}

	// Outputs
	if outputs, ok := state["outputs"].(map[string]interface{}); ok && len(outputs) > 0 {
		fmt.Printf("\nOutputs: %d defined\n", len(outputs))
		for key := range outputs {
			fmt.Printf("  - %s\n", key)
		}
	}
}

func displayStateDetailed(stateFile interface{}, showSensitive bool) {
	stateData, _ := json.Marshal(stateFile)
	var state map[string]interface{}
	json.Unmarshal(stateData, &state)

	fmt.Println("========================================")
	fmt.Println("TERRAFORM STATE FILE DETAILED VIEW")
	fmt.Println("========================================")

	displayStateSummary(stateFile)

	// Show detailed resource information
	if resources, ok := state["resources"].([]interface{}); ok {
		fmt.Println("\n========================================")
		fmt.Println("RESOURCE DETAILS")
		fmt.Println("========================================")

		for i, r := range resources {
			if resource, ok := r.(map[string]interface{}); ok {
				fmt.Printf("\n%d. %s.%s\n", i+1, resource["type"], resource["name"])
				fmt.Println(strings.Repeat("-", 40))

				// Show provider
				if provider, ok := resource["provider"].(string); ok {
					fmt.Printf("Provider: %s\n", provider)
				}

				// Show instances
				if instances, ok := resource["instances"].([]interface{}); ok {
					for j, inst := range instances {
						if instance, ok := inst.(map[string]interface{}); ok {
							fmt.Printf("\nInstance %d:\n", j+1)

							// Show attributes
							if attrs, ok := instance["attributes"].(map[string]interface{}); ok {
								fmt.Println("  Attributes:")
								for key, value := range attrs {
									// Check if sensitive
									isSensitive := false
									if sensitiveAttrs, ok := instance["sensitive_attributes"].([]interface{}); ok {
										for _, sa := range sensitiveAttrs {
											if sa == key {
												isSensitive = true
												break
											}
										}
									}

									if isSensitive && !showSensitive {
										fmt.Printf("    %s: <sensitive>\n", key)
									} else {
										// Format value based on type
										switch v := value.(type) {
										case string:
											if len(v) > 50 {
												fmt.Printf("    %s: %s...\n", key, v[:50])
											} else {
												fmt.Printf("    %s: %s\n", key, v)
											}
										case []interface{}:
											fmt.Printf("    %s: [%d items]\n", key, len(v))
										case map[string]interface{}:
											fmt.Printf("    %s: {%d fields}\n", key, len(v))
										default:
											fmt.Printf("    %s: %v\n", key, value)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func displayStateJSON(stateFile interface{}, showSensitive bool) {
	// If not showing sensitive data, filter it out
	if !showSensitive {
		// Marshal and unmarshal to work with the data
		stateData, _ := json.Marshal(stateFile)
		var state map[string]interface{}
		json.Unmarshal(stateData, &state)

		// Filter sensitive attributes
		if resources, ok := state["resources"].([]interface{}); ok {
			for _, r := range resources {
				if resource, ok := r.(map[string]interface{}); ok {
					if instances, ok := resource["instances"].([]interface{}); ok {
						for _, inst := range instances {
							if instance, ok := inst.(map[string]interface{}); ok {
								if sensitiveAttrs, ok := instance["sensitive_attributes"].([]interface{}); ok {
									if attrs, ok := instance["attributes"].(map[string]interface{}); ok {
										for _, sa := range sensitiveAttrs {
											if saStr, ok := sa.(string); ok {
												attrs[saStr] = "<sensitive>"
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		stateFile = state
	}

	// Pretty print JSON
	data, err := json.MarshalIndent(stateFile, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
