package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Katana struct{}

func (k *Katana) Name() string {
	return "katana"
}

func (k *Katana) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("katana")
	_, err := exec.LookPath(path)
	return err == nil
}

func (k *Katana) Install() error {
	cmd := exec.Command("go", "install", "github.com/projectdiscovery/katana/cmd/katana@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install katana: %v", err)
	}
	return nil
}

func (k *Katana) Run(ctx context.Context, target string) (string, error) {
	utils.LogInfo("Running katana on %s...", target)
	// -u target -jc -kf all -fx -d 5 -pc -silent
	path := utils.ResolveBinaryPath("katana")
	cmd := exec.CommandContext(ctx, path, "-u", target, "-jc", "-kf", "all", "-fx", "-d", "5", "-pc", "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("katana failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}

func (k *Katana) RunCustom(ctx context.Context, target string, args []string) (string, error) {
	utils.LogInfo("Running katana custom on %s...", target)
	path := utils.ResolveBinaryPath("katana")

	// Construct args: -u <target> [custom args...]
	cmdArgs := []string{"-u", target, "-silent"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, path, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("katana custom failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
