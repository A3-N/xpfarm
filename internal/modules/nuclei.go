package modules

import (
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

func (n *Nuclei) Run(target string) (string, error) {
	utils.LogInfo("Running nuclei on %s...", target)
	// -u target -silent
	path := utils.ResolveBinaryPath("nuclei")
	cmd := exec.Command(path, "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nuclei failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
