package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Nuclei struct{}

func (n *Nuclei) Name() string {
	return "nuclei"
}

func (n *Nuclei) CheckInstalled() bool {
	_, err := exec.LookPath("nuclei")
	return err == nil
}

func (n *Nuclei) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install nuclei: %v", err)
	}
	return nil
}

func (n *Nuclei) Run(target string) (string, error) {
	utils.LogInfo("Running nuclei on %s...", target)
	// -u target -silent
	cmd := exec.Command("nuclei", "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nuclei failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
