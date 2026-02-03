package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"xpfarm/pkg/utils"
)

type Gowitness struct{}

func (g *Gowitness) Name() string {
	return "gowitness"
}

func (g *Gowitness) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("gowitness")
	_, err := exec.LookPath(path)
	return err == nil
}

func (g *Gowitness) Install() error {
	cmd := exec.Command("go", "install", "github.com/sensepost/gowitness@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install gowitness: %v", err)
	}
	return nil
}

// RunSingle captures a screenshot for a single URL
func (g *Gowitness) RunSingle(ctx context.Context, url string) (string, error) {
	utils.LogInfo("Running gowitness on %s...", url)
	path := utils.ResolveBinaryPath("gowitness")

	// Generate a safe filename
	safeName := strings.ReplaceAll(url, "://", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	// filename var not used in favor of prefix logic below

	// gowitness single -u <url> -o <dir> --no-http
	// NOTE: Flags depend on version. Assuming `single -u ... -o <dir>` or similar.
	// Actually `gowitness single` takes `-u` and usually `--screenshot-path` or just writes to db/screenshots.
	// Let's try: `gowitness single -u <url> -o screenshots/<filename>` if supported, or rely on default behavior.
	// Common modern gowitness: `gowitness single -u <url> --screenshot-path screenshots`

	// User requested: gowitness scan single -u ... --screenshot-fullpage (defaults to ./screenshots)
	cmd := exec.CommandContext(ctx, path, "scan", "single", "-u", url, "--screenshot-fullpage")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gowitness failed: %v\nOutput: %s", err, output)
	}

	// Construct the resulting filepath based on observed gowitness behavior
	// URL: http://example.com:80 -> http---example.com-80.jpeg
	// Replacements: : -> -, / -> -
	prefix := strings.ReplaceAll(url, ":", "-")
	prefix = strings.ReplaceAll(prefix, "/", "-")

	// Check common extensions
	extensions := []string{".jpeg", ".jpg", ".png"}
	var finalPath string

	// Ensure directory exists
	// But simply checking path presence is safer

	for _, ext := range extensions {
		candidate := fmt.Sprintf("screenshots/%s%s", prefix, ext)
		// Check both absolute and relative
		if _, err := os.Stat(candidate); err == nil {
			finalPath = candidate
			break
		}
	}

	// Fallback/Error if not found
	if finalPath == "" {
		return "", fmt.Errorf("screenshot file not found for %s (looked for prefix %s)", url, prefix)
	}

	return finalPath, nil
}

func (g *Gowitness) Run(ctx context.Context, target string) (string, error) {
	_, err := g.RunSingle(ctx, target)
	return "", err
}
