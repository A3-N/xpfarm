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
	// The binary installed by cvemap/cmd/vulnx is 'vulnx'
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
	utils.LogInfo("Running cvemap search for %s...", target)
	// -q query -silent (e.g., search term)
	// Binary is vulnx
	path := utils.ResolveBinaryPath("vulnx")
	cmd := exec.CommandContext(ctx, path, "-q", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cvemap failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
