package modules

import (
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

func (k *Katana) Run(target string) (string, error) {
	utils.LogInfo("Running katana on %s...", target)
	// -u target -silent
	path := utils.ResolveBinaryPath("katana")
	cmd := exec.Command(path, "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("katana failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
