package modules

import (
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Naabu struct{}

func (n *Naabu) Name() string {
	return "naabu"
}

func (n *Naabu) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("naabu")
	_, err := exec.LookPath(path)
	return err == nil
}

func (n *Naabu) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/naabu/v2/cmd/naabu@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install naabu: %v", err)
	}
	return nil
}

func (n *Naabu) Run(target string) (string, error) {
	utils.LogInfo("Running naabu on %s...", target)
	// -host target -silent
	path := utils.ResolveBinaryPath("naabu")
	cmd := exec.Command(path, "-host", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("naabu failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
