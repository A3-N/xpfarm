package modules

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"xpfarm/pkg/utils"
)

type Nuclei struct{}

func (n *Nuclei) Name() string {
	return "nuclei"
}

func (n *Nuclei) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("nuclei")
	_, err := exec.LookPath(path)
	return err == nil
}

func (n *Nuclei) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install nuclei: %v", err)
	}
	return nil
}

// Run satisfies the Module interface — runs a basic scan against a single target.
func (n *Nuclei) Run(ctx context.Context, target string) (string, error) {
	return n.RunRaw(ctx, []string{"-u", target, "-jsonl", "-silent"})
}

// RunRaw executes nuclei with the exact arguments provided.
func (n *Nuclei) RunRaw(ctx context.Context, args []string) (string, error) {
	path := utils.ResolveBinaryPath("nuclei")
	baseArgs := []string{"-c", "25", "-rl", "150", "-duc"}
	fullArgs := make([]string, 0, len(baseArgs)+len(args))
	fullArgs = append(fullArgs, baseArgs...)
	fullArgs = append(fullArgs, args...)

	utils.LogInfo("[Nuclei] Running: %s %s", path, strings.Join(fullArgs, " "))
	cmd := exec.CommandContext(ctx, path, fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("nuclei failed: %v", err)
	}
	return string(output), nil
}

// RunWithTags scans a specific host:port with service-specific tags and protocol type.
func (n *Nuclei) RunWithTags(ctx context.Context, target string, tags []string, protocolType string) (string, error) {
	args := []string{"-u", target, "-tags", strings.Join(tags, ","), "-jsonl", "-silent"}
	if protocolType != "" {
		args = append(args, "-pt", protocolType)
	}
	return n.RunRaw(ctx, args)
}

// RunAutoScan runs nuclei with -as (automatic wappalyzer-based template selection).
func (n *Nuclei) RunAutoScan(ctx context.Context, urlsFile string) (string, error) {
	return n.RunRaw(ctx, []string{"-l", urlsFile, "-as", "-jsonl", "-silent"})
}

// RunSSLScan runs nuclei with SSL protocol type templates.
func (n *Nuclei) RunSSLScan(ctx context.Context, target string) (string, error) {
	return n.RunRaw(ctx, []string{"-u", target, "-pt", "ssl", "-jsonl", "-silent"})
}

// NucleiResult represents a single finding from nuclei JSONL output.
type NucleiResult struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name        string   `json:"name"`
		Severity    string   `json:"severity"`
		Tags        []string `json:"tags"`
		Description string   `json:"description"`
		Reference   []string `json:"reference"`
	} `json:"info"`
	MatcherName      string   `json:"matcher-name"`
	Type             string   `json:"type"`
	Host             string   `json:"host"`
	Port             string   `json:"port"`
	URL              string   `json:"url"`
	MatchedAt        string   `json:"matched-at"`
	ExtractedResults []string `json:"extracted-results"`
	Timestamp        string   `json:"timestamp"`
}
