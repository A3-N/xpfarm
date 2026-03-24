package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ScanPluginDir walks dir and parses every plugin.yaml it finds.
//
// This is intentionally read-only — it does not execute any code.
// Use it in the UI to show plugins that exist on disk but have not been
// compiled into the current binary yet (future marketplace / hot-reload hint).
//
// Returns nil, nil when dir does not exist.
func ScanPluginDir(dir string) ([]Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("plugin: scanning %s: %w", dir, err)
	}

	var found []Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "plugin.yaml"))
		if err != nil {
			continue // no plugin.yaml — not a plugin directory
		}
		var m Manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("plugin: parsing %s/plugin.yaml: %w", e.Name(), err)
		}
		found = append(found, m)
	}
	return found, nil
}

// PrintSummary writes a human-readable inventory of all registered plugins to
// stdout. Call it after importing plugins/all to confirm what was loaded.
func PrintSummary() {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Printf("[plugin] %d tool(s) registered:\n", len(tools))
	for name := range tools {
		fmt.Printf("  • tool: %s\n", name)
	}
	fmt.Printf("[plugin] %d agent(s) registered:\n", len(agents))
	for name := range agents {
		fmt.Printf("  • agent: %s\n", name)
	}
	fmt.Printf("[plugin] %d pipeline(s) registered:\n", len(pipelines))
	for name := range pipelines {
		fmt.Printf("  • pipeline: %s\n", name)
	}
}
