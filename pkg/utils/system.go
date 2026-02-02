package utils

import (
	"os"
	"os/exec"
	"path/filepath"
)

// ResolveBinaryPath checks if a binary exists in the system PATH.
// If not, it checks standard Go bin locations ($HOME/go/bin, $GOPATH/bin).
// It returns the absolute path if found in a fallback location, or the tool name if found in PATH (or not found at all).
func ResolveBinaryPath(toolName string) string {
	// 1. Check system PATH
	if _, err := exec.LookPath(toolName); err == nil {
		return toolName
	}

	// 2. Check Fallback Locations
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// $HOME/go/bin/toolName
		defaultGoBin := filepath.Join(homeDir, "go", "bin", toolName)
		if _, err := os.Stat(defaultGoBin); err == nil {
			return defaultGoBin
		}
	}

	goPath := os.Getenv("GOPATH")
	if goPath != "" {
		// $GOPATH/bin/toolName
		goPathBin := filepath.Join(goPath, "bin", toolName)
		if _, err := os.Stat(goPathBin); err == nil {
			return goPathBin
		}
	}

	// Not found in fallbacks, return original name (which will likely fail later, but that's expected)
	return toolName
}
