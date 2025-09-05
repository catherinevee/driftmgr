package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// InitConfig represents the initial configuration for DriftMgr
type InitConfig struct {
	Provider    string                 `yaml:"provider"`
	Regions     []string               `yaml:"regions"`
	Credentials map[string]string      `yaml:"credentials,omitempty"`
	Settings    map[string]interface{} `yaml:"settings"`
}

// HandleInit handles the init command to initialize DriftMgr configuration
func HandleInit(args []string) {
	provider := ""
	regions := []string{}
	configPath := ".driftmgr/config.yaml"
	interactive := true

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--region", "-r":
			if i+1 < len(args) {
				regions = strings.Split(args[i+1], ",")
				i++
			}
		case "--config", "-c":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case "--non-interactive":
			interactive = false
		case "--help", "-h":
			showInitHelp()
			return
		}
	}

	// Interactive mode
	if interactive && provider == "" {
		fmt.Println("Welcome to DriftMgr initialization!")
		fmt.Println()
		fmt.Println("Select your primary cloud provider:")
		fmt.Println("1. AWS")
		fmt.Println("2. Azure")
		fmt.Println("3. GCP")
		fmt.Println("4. DigitalOcean")
		fmt.Println("5. Multi-cloud")
		fmt.Print("\nChoice (1-5): ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			provider = "aws"
		case 2:
			provider = "azure"
		case 3:
			provider = "gcp"
		case 4:
			provider = "digitalocean"
		case 5:
			provider = "multi"
		default:
			fmt.Println("Invalid choice")
			os.Exit(1)
		}
	}

	// Set default regions if not provided
	if len(regions) == 0 {
		switch provider {
		case "aws":
			regions = []string{"us-east-1", "us-west-2"}
		case "azure":
			regions = []string{"eastus", "westus2"}
		case "gcp":
			regions = []string{"us-central1", "us-east1"}
		case "digitalocean":
			regions = []string{"nyc1", "sfo3"}
		case "multi":
			regions = []string{"global"}
		}
	}

	// Create configuration
	config := InitConfig{
		Provider: provider,
		Regions:  regions,
		Settings: map[string]interface{}{
			"auto_discovery":   true,
			"parallel_workers": 10,
			"cache_ttl":        "1h",
			"drift_detection": map[string]interface{}{
				"enabled":  true,
				"interval": "15m",
			},
			"remediation": map[string]interface{}{
				"enabled":           false,
				"dry_run":           true,
				"approval_required": true,
			},
			"database": map[string]interface{}{
				"enabled": true,
				"path":    "~/.driftmgr/driftmgr.db",
			},
			"logging": map[string]interface{}{
				"level": "info",
				"file":  "~/.driftmgr/driftmgr.log",
			},
		},
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Marshal configuration to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// Write configuration file
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Configuration initialized at %s\n", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("1. Configure %s credentials (environment variables or credential files)\n", provider)
	fmt.Println("2. Run 'driftmgr validate' to verify your configuration")
	fmt.Println("3. Run 'driftmgr discover' to start discovering resources")
	fmt.Println()
	fmt.Println("For more information, run 'driftmgr --help'")
}

func showInitHelp() {
	fmt.Println("Usage: driftmgr init [flags]")
	fmt.Println()
	fmt.Println("Initialize DriftMgr configuration")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --provider, -p     Cloud provider (aws, azure, gcp, digitalocean, multi)")
	fmt.Println("  --region, -r       Comma-separated list of regions")
	fmt.Println("  --config, -c       Path to configuration file (default: .driftmgr/config.yaml)")
	fmt.Println("  --non-interactive  Run in non-interactive mode")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  driftmgr init")
	fmt.Println("  driftmgr init --provider aws --region us-east-1,us-west-2")
	fmt.Println("  driftmgr init --provider azure --non-interactive")
}
