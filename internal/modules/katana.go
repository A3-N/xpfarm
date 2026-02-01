package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Katana struct{}

func (k *Katana) Name() string {
	return "katana"
}

func (k *Katana) CheckInstalled() bool {
	_, err := exec.LookPath("katana")
	return err == nil
}

func (k *Katana) Install() error {
	cmd := exec.Command("go", "install", "github.com/projectdiscovery/katana/cmd/katana@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install katana: %v", err)
	}
	return nil
}

func (k *Katana) Run(target string) (string, error) {
	utils.LogInfo("Running katana on %s...", target)
	// -u target -silent
	cmd := exec.Command("katana", "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("katana failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
