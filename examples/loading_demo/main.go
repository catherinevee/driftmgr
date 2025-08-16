package main

import (
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/utils"
)

func main() {
	fmt.Println("DriftMgr Loading Bar Demo")
	fmt.Println("=========================")
	fmt.Println()

	// Demo 1: Progress Bar
	fmt.Println("1. Progress Bar Demo:")
	utils.GlobalLoadingManager.ShowProgress("Processing resources", func(update func(float64)) {
		for i := 0; i <= 100; i++ {
			update(float64(i) / 100.0)
			time.Sleep(50 * time.Millisecond)
		}
	})
	fmt.Println()

	// Demo 2: Spinner
	fmt.Println("2. Spinner Demo:")
	utils.GlobalLoadingManager.ShowSpinner("Discovering cloud resources", func() {
		time.Sleep(3 * time.Second)
	})
	fmt.Println()

	// Demo 3: Simple Message
	fmt.Println("3. Simple Message Demo:")
	utils.GlobalLoadingManager.ShowSimpleMessage("Validating credentials")
	time.Sleep(1 * time.Second)
	utils.GlobalLoadingManager.CompleteMessage()

	// Demo 4: Error Message
	fmt.Println("4. Error Message Demo:")
	utils.GlobalLoadingManager.ErrorMessage(fmt.Errorf("connection timeout"))

	fmt.Println("\nDemo completed!")
}
