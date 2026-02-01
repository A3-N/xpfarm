package main

import (
	"fmt"
	"log"
	"os"

	"xpfarm/internal/database"
	"xpfarm/internal/modules"
	"xpfarm/internal/ui"
	"xpfarm/pkg/utils"
)

func main() {
	fmt.Println("XPFarm - Automated Pentest Tool")
	fmt.Println("-------------------------------")

	// 1. Initialize Database
	utils.LogInfo("Initializing Database...")
	database.InitDB()

	// 2. Register Modules
	modules.InitModules()

	// 3. Health Checks & Installation
	utils.LogInfo("Checking Dependencies...")
	allModules := modules.GetAll()
	missingCount := 0

	for _, mod := range allModules {
		if !mod.CheckInstalled() {
			utils.LogWarning("Tool '%s' not found. Attempting install...", mod.Name())
			if err := mod.Install(); err != nil {
				utils.LogError("Failed to install '%s': %v", mod.Name(), err)
				missingCount++
			} else {
				utils.LogSuccess("Successfully installed '%s'", mod.Name())
			}
		} else {
			// Optional: Verbose mode could show installed tools
			// utils.LogSuccess("'%s' is installed.", mod.Name())
		}
	}

	if missingCount > 0 {
		utils.LogError("%d tools failed to install. The tool might not function correctly.", missingCount)
		// We can decide to exit here or continue.
		// User said "if it fails it will error out".
		utils.LogError("Exiting due to missing dependencies.")
		os.Exit(1)
	}

	utils.LogSuccess("All dependencies satisfied.")

	// 4. Start Web Server
	port := "8888"
	utils.LogSuccess("Starting Web Interface on port %s...", port)
	utils.LogSuccess("Access at http://localhost:%s", port)

	// Open browser? Maybe later.

	if err := ui.StartServer(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
