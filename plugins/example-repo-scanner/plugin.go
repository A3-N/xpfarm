// Package reposcanner is the example-repo-scanner XPFarm plugin.
//
// It registers:
//   - RepoScannerTool   — static-analysis scan of a source repository
//   - RepoScannerAgent  — orchestrates the scan and attaches a severity summary
//
// The tool currently returns mock findings. To make it real, replace the body
// of RepoScannerTool.Run with an exec.Command call to Semgrep, Gitleaks, or
// any other static-analysis binary.
package reposcanner

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"xpfarm/internal/plugin"
)

//go:embed plugin.yaml
var rawManifest []byte

func init() {
	var m plugin.Manifest
	if err := yaml.Unmarshal(rawManifest, &m); err != nil {
		panic(fmt.Sprintf("example-repo-scanner: malformed plugin.yaml: %v", err))
	}
	plugin.RegisterManifest(m)
	plugin.RegisterTool(&RepoScannerTool{})
	plugin.RegisterAgent(&RepoScannerAgent{tool: &RepoScannerTool{}})
}

// -----------------------------------------------------------------------------
// RepoScannerTool
// -----------------------------------------------------------------------------

// RepoScannerTool performs a static-analysis scan of a source repository.
//
// Required input keys:
//
//	repo_url  string — URL or local path of the repository to scan
//
// Optional input keys:
//
//	branch   string — branch to scan (default: "main")
//	ruleset  string — scanner ruleset identifier (default: "auto")
type RepoScannerTool struct{}

func (t *RepoScannerTool) Name() string { return "repo-scanner" }
func (t *RepoScannerTool) Description() string {
	return "Static-analysis scan of a source repository. Returns findings with severity, rule ID, and file location."
}

// Finding represents a single scanner result.
type Finding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"` // CRITICAL | HIGH | MEDIUM | LOW
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
}

func (t *RepoScannerTool) Run(_ context.Context, input map[string]any) (map[string]any, error) {
	repoURL, _ := input["repo_url"].(string)
	if repoURL == "" {
		return nil, fmt.Errorf("repo-scanner: missing required input key 'repo_url'")
	}

	branch, _ := input["branch"].(string)
	if branch == "" {
		branch = "main"
	}
	ruleset, _ := input["ruleset"].(string)
	if ruleset == "" {
		ruleset = "auto"
	}

	// Derive a short repo name from the URL for display purposes.
	repoName := repoURL
	if idx := strings.LastIndex(strings.TrimRight(repoURL, "/"), "/"); idx >= 0 {
		repoName = repoURL[idx+1:]
	}

	// -------------------------------------------------------------------------
	// Mock findings — replace this block with a real tool invocation, e.g.:
	//
	//   cmd := exec.CommandContext(ctx, "semgrep", "--json", "--config=auto", ".")
	//   out, err := cmd.Output()
	//   // parse JSON into []Finding
	// -------------------------------------------------------------------------
	findings := []Finding{
		{
			RuleID:   "hardcoded-secret",
			Severity: "HIGH",
			File:     repoName + "/config/settings.go",
			Line:     42,
			Message:  "Hardcoded API key detected.",
		},
		{
			RuleID:   "sql-injection",
			Severity: "CRITICAL",
			File:     repoName + "/internal/db/query.go",
			Line:     87,
			Message:  "User input concatenated directly into SQL query.",
		},
		{
			RuleID:   "insecure-rand",
			Severity: "MEDIUM",
			File:     repoName + "/pkg/auth/token.go",
			Line:     15,
			Message:  "math/rand used for security-sensitive token generation; use crypto/rand.",
		},
	}
	// -------------------------------------------------------------------------

	return map[string]any{
		"repo_url":   repoURL,
		"branch":     branch,
		"ruleset":    ruleset,
		"scanned_at": time.Now().UTC().Format(time.RFC3339),
		"findings":   findings,
		"total":      len(findings),
	}, nil
}

// -----------------------------------------------------------------------------
// RepoScannerAgent
// -----------------------------------------------------------------------------

// RepoScannerAgent orchestrates a full repository security scan.
// It delegates to RepoScannerTool and enriches the result with a severity
// breakdown so callers can triage without iterating findings themselves.
type RepoScannerAgent struct {
	tool *RepoScannerTool
}

func (a *RepoScannerAgent) Name() string         { return "repo-scanner-agent" }
func (a *RepoScannerAgent) Tools() []plugin.Tool { return []plugin.Tool{a.tool} }

func (a *RepoScannerAgent) Handle(ctx context.Context, task plugin.Task) (plugin.Result, error) {
	// Promote Task.Target to repo_url when the caller omits it from Payload.
	payload := make(map[string]any, len(task.Payload)+1)
	for k, v := range task.Payload {
		payload[k] = v
	}
	if _, ok := payload["repo_url"]; !ok {
		payload["repo_url"] = task.Target
	}

	out, err := a.tool.Run(ctx, payload)
	if err != nil {
		return plugin.Result{Error: err.Error()}, err
	}

	// Build severity summary so callers can triage at a glance.
	findings, _ := out["findings"].([]Finding)
	severity := map[string]int{"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0}
	for _, f := range findings {
		if _, ok := severity[f.Severity]; ok {
			severity[f.Severity]++
		}
	}
	out["severity_summary"] = severity

	return plugin.Result{Output: out}, nil
}
