package cmd

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/tui"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive terminal UI",
	Long: `Launch the interactive terminal user interface for discovering, selecting, 
and importing resources. The TUI provides a user-friendly way to manage your 
infrastructure import process with real-time updates and progress tracking.`,
	RunE: runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func runInteractive(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Launching interactive mode...")

	app := tui.NewApp()
	return app.Run()
}
