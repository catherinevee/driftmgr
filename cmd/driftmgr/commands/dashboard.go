package commands

import (
	"fmt"
	// "log"
	"os"
	"os/signal"
	"syscall"
	// "github.com/catherinevee/driftmgr/internal/app/api"
)

// HandleDashboard handles the dashboard command
func HandleDashboard(args []string) {
	var port string = "8081"

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: driftmgr dashboard [flags]")
			fmt.Println()
			fmt.Println("Start the DriftMgr web dashboard")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --port, -p string    Port to run dashboard on (default: 8081)")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr dashboard")
			fmt.Println("  driftmgr dashboard --port 8080")
			return
		}
	}

	fmt.Printf("Starting DriftMgr Dashboard Server on port %s\n", port)
	fmt.Printf("Open your browser at http://localhost:%s\n", port)
	fmt.Println("\nPress Ctrl+C to stop the server")

	// Create dashboard server
	// server := dashboard.NewDashboardServer(port)
	fmt.Println("Dashboard functionality temporarily disabled")

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down dashboard server...")
		os.Exit(0)
	}()

	// Start server
	// if err := server.Start(); err != nil {
	// 	log.Fatal("Failed to start dashboard server:", err)
	// }

	// Keep running until interrupted
	select {}
}
