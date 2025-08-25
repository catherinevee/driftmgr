package main

import (
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/progress"
)

func main() {
	fmt.Println("DriftMgr Progress Indicators Demo")
	fmt.Println("==================================\n")

	// Demo 1: Basic Spinner
	fmt.Println("1. Basic Spinner:")
	spinner := progress.NewSpinner("Processing data")
	spinner.Start()
	time.Sleep(3 * time.Second)
	spinner.Success("Data processed successfully")

	// Demo 2: Dot Spinner
	fmt.Println("\n2. Dot Spinner:")
	dotSpinner := progress.NewDotSpinner("Loading configuration")
	dotSpinner.Start()
	time.Sleep(2 * time.Second)
	dotSpinner.Success("Configuration loaded")

	// Demo 3: Bar Spinner
	fmt.Println("\n3. Bar Spinner:")
	barSpinner := progress.NewBarSpinner("Connecting to cloud provider")
	barSpinner.Start()
	time.Sleep(2 * time.Second)
	barSpinner.UpdateMessage("Authenticating...")
	time.Sleep(1 * time.Second)
	barSpinner.Success("Connected to AWS")

	// Demo 4: Progress Bar
	fmt.Println("\n4. Progress Bar:")
	bar := progress.NewBar(100, "Discovering resources")
	for i := 0; i <= 100; i++ {
		bar.Update(i)
		time.Sleep(50 * time.Millisecond)
	}

	// Demo 5: Loading Animation
	fmt.Println("\n5. Loading Animation:")
	loading := progress.NewLoadingAnimation("Scanning for drift")
	loading.Start()
	time.Sleep(3 * time.Second)
	loading.Complete("Drift scan complete")

	// Demo 6: Multi-Progress
	fmt.Println("\n6. Multi-Progress Display:")
	multi := progress.NewMultiProgress()

	item1 := multi.AddItem("AWS Resources")
	item2 := multi.AddItem("Azure Resources")
	item3 := multi.AddItem("GCP Resources")

	// Simulate processing
	item1.status = "running"
	multi.Render()
	time.Sleep(1 * time.Second)

	item1.status = "success"
	item2.status = "running"
	multi.Render()
	time.Sleep(1 * time.Second)

	item2.status = "success"
	item3.status = "running"
	multi.Render()
	time.Sleep(1 * time.Second)

	item3.status = "success"
	multi.Render()

	fmt.Println("\n\nDemo complete!")
}
