package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/cli"
	// "github.com/catherinevee/driftmgr/internal/credentials" // Package removed
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	// "github.com/catherinevee/driftmgr/internal/progress" // Package may not exist
	"github.com/catherinevee/driftmgr/internal/state"
)

// Test configuration
var (
	driftmgrPath string
	testTimeout  = 30 * time.Second
)

func init() {
	// Determine the executable path based on OS
	if runtime.GOOS == "windows" {
		driftmgrPath = "../../driftmgr.exe"
	} else {
		driftmgrPath = "../../driftmgr"
	}
}

// TestBuildExists verifies the driftmgr executable exists
func TestBuildExists(t *testing.T) {
	if _, err := os.Stat(driftmgrPath); os.IsNotExist(err) {
		t.Fatalf("DriftMgr executable not found at %s. Please build first.", driftmgrPath)
	}
}

// TestBasicCommands tests basic command functionality
func TestBasicCommands(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectError       bool
		expectContains    []string
		expectNotContains []string
	}{
		{
			name:           "Help Command",
			args:           []string{"--help"},
			expectError:    false,
			expectContains: []string{"Usage: driftmgr", "Core Commands"},
		},
		{
			name:           "Status Command",
			args:           []string{"status"},
			expectError:    false,
			expectContains: []string{"DriftMgr System Status"},
		},
		{
			name:           "Unknown Command",
			args:           []string{"unknowncommand"},
			expectError:    true,
			expectContains: []string{"Unknown command"},
		},
		{
			name:           "Invalid Flag",
			args:           []string{"--invalidflag"},
			expectError:    true,
			expectContains: []string{"Unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runCommand(tt.args...)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s' but it didn't.\nOutput: %s", expected, output)
				}
			}

			for _, notExpected := range tt.expectNotContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output NOT to contain '%s' but it did.\nOutput: %s", notExpected, output)
				}
			}
		})
	}
}

// TestCredentialDetection tests credential detection functionality
func TestCredentialDetection(t *testing.T) {
	detector := credentials.NewCredentialDetector()
	creds := detector.DetectAll()

	t.Run("Credential Detection", func(t *testing.T) {
		// At least check that detection doesn't panic
		if creds == nil {
			t.Log("No credentials detected (this is okay if no providers are configured)")
		} else {
			t.Logf("Detected %d credential(s)", len(creds))
			for _, cred := range creds {
				t.Logf("  - %s: %s", cred.Provider, cred.Status)
			}
		}
	})

	t.Run("Multiple Profiles Detection", func(t *testing.T) {
		profiles := detector.DetectMultipleProfiles()
		if len(profiles) > 0 {
			for provider, profs := range profiles {
				t.Logf("%s profiles: %v", provider, profs)
			}
		} else {
			t.Log("No multiple profiles detected")
		}
	})

	t.Run("AWS Accounts Detection", func(t *testing.T) {
		accounts := detector.DetectAWSAccounts()
		if len(accounts) > 0 {
			for accountID, profiles := range accounts {
				t.Logf("AWS Account %s: %v", accountID, profiles)
			}
		} else {
			t.Log("No AWS accounts detected")
		}
	})
}

// TestDiscoveryCommands tests discovery command functionality
func TestDiscoveryCommands(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		skipIfNoCreds bool
	}{
		{
			name:        "Discovery Help",
			args:        []string{"discover", "--help"},
			expectError: false,
		},
		{
			name:        "Discovery with Invalid Provider",
			args:        []string{"discover", "--provider", "invalid"},
			expectError: true,
		},
		{
			name:          "Discovery with JSON Format",
			args:          []string{"discover", "--format", "json"},
			expectError:   false,
			skipIfNoCreds: true,
		},
		{
			name:          "Discovery with Auto Flag",
			args:          []string{"discover", "--auto"},
			expectError:   false,
			skipIfNoCreds: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoCreds && !hasCredentials() {
				t.Skip("Skipping test - no credentials configured")
			}

			output, err := runCommand(tt.args...)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				// Allow "no credentials" errors
				if !strings.Contains(output, "No cloud credentials") {
					t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

// TestColorSupport tests color functionality
func TestColorSupport(t *testing.T) {
	t.Run("Color Functions", func(t *testing.T) {
		// Test that color functions don't panic
		tests := []struct {
			name string
			fn   func(string) string
			text string
		}{
			// TODO: Uncomment when color functions are implemented in cli package
			// {"AWS Color", cli.AWS, "AWS Provider"},
			// {"Azure Color", cli.Azure, "Azure Provider"},
			// {"GCP Color", cli.GCP, "GCP Provider"},
			// {"Success Color", cli.Success, "Success"},
			// {"Error Color", cli.Error, "Error"},
			// {"Warning Color", cli.Warning, "Warning"},
			// {"Info Color", cli.Info, "Info"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.fn(tt.text)
				if result == "" {
					t.Errorf("%s returned empty string", tt.name)
				}
			})
		}
	})

	t.Run("NO_COLOR Environment", func(t *testing.T) {
		// Save and restore NO_COLOR
		oldNoColor := os.Getenv("NO_COLOR")
		defer os.Setenv("NO_COLOR", oldNoColor)

		os.Setenv("NO_COLOR", "1")
		output, _ := runCommand("status")

		// Check that output doesn't contain ANSI codes
		if strings.Contains(output, "\033[") {
			t.Error("Output contains ANSI codes when NO_COLOR is set")
		}
	})
}

// TestProgressIndicators tests progress indicator functionality
func TestProgressIndicators(t *testing.T) {
	t.Run("Spinner Creation", func(t *testing.T) {
		spinner := progress.NewSpinner("Test spinner")
		if spinner == nil {
			t.Error("Failed to create spinner")
		}
	})

	t.Run("Progress Bar Creation", func(t *testing.T) {
		bar := progress.NewBar(100, "Test progress")
		if bar == nil {
			t.Error("Failed to create progress bar")
		}

		// Test update
		bar.Update(50)
		bar.Complete()
	})

	t.Run("Loading Animation Creation", func(t *testing.T) {
		loading := progress.NewLoadingAnimation("Test loading")
		if loading == nil {
			t.Error("Failed to create loading animation")
		}
	})
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Missing Required Arguments",
			args: []string{"discover", "--provider"},
		},
		{
			name: "Invalid File Path",
			args: []string{"export", "--output", "/invalid:/path/file.json"},
		},
		{
			name: "Very Long Argument",
			args: []string{"discover", "--provider", strings.Repeat("a", 10000)},
		},
		{
			name: "Special Characters",
			args: []string{"export", "--output", "test file with spaces.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We expect these to error, but shouldn't panic
			_, _ = runCommand(tt.args...)
			// If we get here without panic, test passes
		})
	}
}

// TestConfigurationFiles tests configuration file handling
func TestConfigurationFiles(t *testing.T) {
	configFiles := []string{
		"configs/config.yaml",
		"configs/smart-defaults.yaml",
		"configs/driftmgr.yaml",
	}

	for _, file := range configFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			// Check from project root
			configPath := filepath.Join("../..", file)
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Logf("Config file not found: %s (may be expected)", file)
			} else {
				t.Logf("Config file exists: %s", file)
			}
		})
	}
}

// TestStateFileOperations tests state file operations
func TestStateFileOperations(t *testing.T) {
	t.Run("State Loader Creation", func(t *testing.T) {
		loader := state.NewStateLoader("test.tfstate")
		if loader == nil {
			t.Error("Failed to create state loader")
		}
	})

	t.Run("State Discovery Command", func(t *testing.T) {
		output, _ := runCommand("state", "discover")
		// Should at least not panic
		t.Logf("State discover output length: %d", len(output))
	})
}

// TestDriftDetection tests drift detection functionality
func TestDriftDetection(t *testing.T) {
	t.Run("Smart Defaults Creation", func(t *testing.T) {
		smartDefaults := drift.NewSmartDefaults("configs/smart-defaults.yaml")
		if smartDefaults == nil {
			t.Log("Smart defaults not created (config may not exist)")
		}
	})

	t.Run("Drift Detect Command", func(t *testing.T) {
		output, _ := runCommand("drift", "detect", "--help")
		if !strings.Contains(output, "detect") {
			t.Error("Drift detect help doesn't contain expected text")
		}
	})
}

// TestPerformance tests performance requirements
func TestPerformance(t *testing.T) {
	t.Run("Help Command Performance", func(t *testing.T) {
		start := time.Now()
		_, _ = runCommand("--help")
		elapsed := time.Since(start)

		if elapsed > 1*time.Second {
			t.Errorf("Help command took too long: %v (expected < 1s)", elapsed)
		} else {
			t.Logf("Help command completed in %v", elapsed)
		}
	})

	t.Run("Status Command Performance", func(t *testing.T) {
		start := time.Now()
		_, _ = runCommand("status")
		elapsed := time.Since(start)

		if elapsed > 5*time.Second {
			t.Errorf("Status command took too long: %v (expected < 5s)", elapsed)
		} else {
			t.Logf("Status command completed in %v", elapsed)
		}
	})
}

// TestJSONOutput tests JSON output parsing
func TestJSONOutput(t *testing.T) {
	t.Run("Export JSON Format", func(t *testing.T) {
		output, err := runCommand("export", "--format", "json")
		if err == nil && len(output) > 0 && strings.HasPrefix(strings.TrimSpace(output), "{") {
			// Try to parse as JSON
			var data interface{}
			if err := json.Unmarshal([]byte(output), &data); err != nil {
				t.Logf("Output is not valid JSON (may be expected if no resources): %v", err)
			} else {
				t.Log("Successfully parsed JSON output")
			}
		}
	})
}

// TestIntegration tests integration between components
func TestIntegration(t *testing.T) {
	t.Run("Discovery Engine Creation", func(t *testing.T) {
		engine, err := discovery.NewEnhancedEngine()
		if err != nil {
			t.Logf("Failed to create discovery engine: %v (may be expected without credentials)", err)
		} else if engine != nil {
			t.Log("Successfully created discovery engine")
		}
	})

	t.Run("Command Chaining", func(t *testing.T) {
		// Run status first
		statusOutput, _ := runCommand("status")

		// If configured, try discovery
		if strings.Contains(statusOutput, "Configured") {
			discoverOutput, _ := runCommand("discover", "--auto")
			if len(discoverOutput) > 0 {
				t.Log("Command chaining successful")
			}
		} else {
			t.Skip("No providers configured for command chaining test")
		}
	})
}

// Helper functions

func runCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, driftmgrPath, args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()

	// Combine stdout and stderr for complete output
	output := out.String() + errOut.String()

	return output, err
}

func hasCredentials() bool {
	detector := credentials.NewCredentialDetector()
	creds := detector.DetectAll()

	for _, cred := range creds {
		if cred.Status == "configured" {
			return true
		}
	}

	return false
}

// Benchmark tests

func BenchmarkHelpCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = runCommand("--help")
	}
}

func BenchmarkStatusCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = runCommand("status")
	}
}

func BenchmarkCredentialDetection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detector := credentials.NewCredentialDetector()
		_ = detector.DetectAll()
	}
}

// TestMain allows setup/teardown
func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Starting DriftMgr Comprehensive Tests")

	// Check if executable exists
	if _, err := os.Stat(driftmgrPath); os.IsNotExist(err) {
		fmt.Printf("Building DriftMgr executable...\n")
		buildCmd := exec.Command("go", "build", "-o", driftmgrPath, "./cmd/driftmgr")
		buildCmd.Dir = "../.."
		if err := buildCmd.Run(); err != nil {
			fmt.Printf("Failed to build: %v\n", err)
			os.Exit(1)
		}
	}

	// Run tests
	code := m.Run()

	// Teardown
	fmt.Println("Tests completed")

	os.Exit(code)
}
