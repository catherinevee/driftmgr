package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "driftmgr",
		Short: "Terraform Import Helper - Simplify importing cloud infrastructure into Terraform",
		Long: `driftmgr is a comprehensive tool that simplifies the process of importing 
existing cloud infrastructure into Terraform state. It provides an intuitive 
interface for discovering, selecting, and bulk-importing resources while ensuring 
state file integrity and optimization.

Features:
- Multi-cloud support (AWS, Azure, GCP)
- Interactive terminal UI
- Bulk import operations
- State file optimization
- Resource discovery and filtering`,
		Version: "1.0.0",
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.driftmgr.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "set log level (debug, info, warn, error)")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".driftmgr")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
