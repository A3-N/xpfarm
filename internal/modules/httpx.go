package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Httpx struct{}

func (h *Httpx) Name() string {
	return "httpx"
}

func (h *Httpx) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("httpx")
	_, err := exec.LookPath(path)
	return err == nil
}

func (h *Httpx) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/httpx/cmd/httpx@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install httpx: %v", err)
	}
	return nil
}

func (h *Httpx) Run(ctx context.Context, target string) (string, error) {
	utils.LogInfo("Running httpx on %s...", target)
	// -u target -silent
	path := utils.ResolveBinaryPath("httpx")
	cmd := exec.CommandContext(ctx, path, "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("httpx failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
