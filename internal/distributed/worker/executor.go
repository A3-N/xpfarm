package worker

import (
	"context"
	"fmt"
	"strings"

	"xpfarm/internal/modules"
)

// ExecuteTool runs a named tool against the target specified in payload.
// payload["target"] is the primary scan target (domain, IP, URL).
// payload["args"] is an optional []string of extra arguments (currently ignored;
// modules handle their own arg construction).
// Returns a result map or an error.
func ExecuteTool(ctx context.Context, toolName string, payload map[string]any) (map[string]any, error) {
	target, ok := payload["target"].(string)
	if !ok || strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("executor: payload must contain a non-empty 'target' string")
	}

	// Try the modules registry first (built-in Go tools)
	mod := modules.Get(toolName)
	if mod == nil {
		// Try common aliases (tool names may differ between job API and module names)
		if aliased := toolAlias(toolName); aliased != "" {
			mod = modules.Get(aliased)
		}
	}
	if mod == nil {
		return nil, fmt.Errorf("executor: tool %q is not registered on this worker", toolName)
	}

	if !mod.CheckInstalled() {
		return nil, fmt.Errorf("executor: tool %q is registered but not installed on this worker", toolName)
	}

	output, err := mod.Run(ctx, target)
	result := map[string]any{
		"tool":   toolName,
		"target": target,
		"output": output,
	}
	if err != nil {
		result["warning"] = err.Error()
	}
	return result, nil
}

// ListAvailableTools returns the names of all installed tools on this worker.
func ListAvailableTools() []string {
	var out []string
	for _, mod := range modules.GetAll() {
		if mod.CheckInstalled() {
			out = append(out, mod.Name())
		}
	}
	return out
}

// toolAlias maps job API tool names to module registry names.
var toolAlias = func(name string) string {
	aliases := map[string]string{
		"subfinder_enum": "subfinder",
		"port_scan":      "naabu",
		"http_probe":     "httpx",
		"vuln_scan":      "nuclei",
		"crawl":          "katana",
		"screenshot":     "gowitness",
		"tech_detect":    "wappalyzer",
		"url_discovery":  "urlfinder",
		"cve_lookup":     "cvemap",
		"service_detect": "nmap",
		"nmap_scan":      "nmap",
	}
	return aliases[name]
}
