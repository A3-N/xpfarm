package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Cvemap struct{}

func (c *Cvemap) Name() string {
	return "cvemap"
}

func (c *Cvemap) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("vulnx")
	_, err := exec.LookPath(path)
	return err == nil
}

func (c *Cvemap) Install() error {
	cmd := exec.Command("go", "install", "github.com/projectdiscovery/cvemap/cmd/vulnx@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cvemap (vulnx): %v", err)
	}
	return nil
}

func (c *Cvemap) Run(ctx context.Context, target string) (string, error) {
	// Basic run: search for the target string?
	return c.Search(ctx, target)
}

// Search runs cvemap with a custom query and returns raw JSON output
func (c *Cvemap) Search(ctx context.Context, query string) (string, error) {
	// vulnx -q "query" -json -silent
	path := utils.ResolveBinaryPath("vulnx")
	utils.LogInfo("Querying cvemap (vulnx): %s", query)

	cmd := exec.CommandContext(ctx, path, "-q", query, "-json", "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cvemap query failed: %v", err)
	}
	return string(output), nil
}
