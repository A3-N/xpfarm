package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Subfinder struct{}

func (s *Subfinder) Name() string {
	return "subfinder"
}

func (s *Subfinder) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("subfinder")
	_, err := exec.LookPath(path)
	return err == nil
}

func (s *Subfinder) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install subfinder: %v", err)
	}
	return nil
}

func (s *Subfinder) Run(ctx context.Context, target string) (string, error) {
	utils.LogInfo("Running subfinder on %s...", target)
	// -d target -silent
	path := utils.ResolveBinaryPath("subfinder")
	cmd := exec.CommandContext(ctx, path, "-d", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("subfinder failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
