package modules

import (
	"context"
	"fmt"
	"os/exec"
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

func (n *Nuclei) Run(ctx context.Context, target string) (string, error) {
	// Default run
	return n.RunCustom(ctx, target, []string{})
}

func (n *Nuclei) RunCustom(ctx context.Context, target string, customArgs []string) (string, error) {
	args := []string{"-u", target, "-silent"}
	args = append(args, customArgs...)
	return n.RunRaw(ctx, args)
}

// RunRaw executes nuclei with exact arguments provided
func (n *Nuclei) RunRaw(ctx context.Context, args []string) (string, error) {
	utils.LogInfo("Running nuclei with args: %v", args)
	path := utils.ResolveBinaryPath("nuclei")

	// Add concurrency optimization flags
	baseArgs := []string{"-c", "25", "-rl", "150"}
	fullArgs := append(baseArgs, args...)

	cmd := exec.CommandContext(ctx, path, fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("nuclei failed: %v", err)
	}
	return string(output), nil
}
