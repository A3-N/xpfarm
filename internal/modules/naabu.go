package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Naabu struct{}

func (n *Naabu) Name() string {
	return "naabu"
}

func (n *Naabu) CheckInstalled() bool {
	_, err := exec.LookPath("naabu")
	return err == nil
}

func (n *Naabu) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/naabu/v2/cmd/naabu@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install naabu: %v", err)
	}
	return nil
}

func (n *Naabu) Run(target string) (string, error) {
	utils.LogInfo("Running naabu on %s...", target)
	// -host target -silent
	cmd := exec.Command("naabu", "-host", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("naabu failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
