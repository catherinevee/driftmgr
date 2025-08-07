package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long: `Manage configuration settings for driftmgr including cloud provider credentials,
default regions, and tool preferences.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file with default settings.`,
	RunE:  runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE:  runConfigList,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".driftmgr.yaml")

	defaultConfig := `# Driftmgr Configuration File
defaults:
  provider: aws
  region: us-east-1
  parallel_imports: 5
  retry_attempts: 3

aws:
  profile: default
  assume_role_arn: null

azure:
  subscription_id: ""
  tenant_id: ""

import:
  dry_run: false
  generate_config: true
  validate_after_import: true

ui:
  theme: dark
  show_progress: true
  log_level: info
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ Configuration file created at %s\n", configPath)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	viper.Set(key, value)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := viper.GetString(key)

	if value == "" {
		fmt.Printf("Key '%s' not found\n", key)
		return nil
	}

	fmt.Printf("%s = %s\n", key, value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	fmt.Println("Current configuration:")

	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("  %s = %v\n", key, value)
	}

	return nil
}
