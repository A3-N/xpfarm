package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Subfinder struct{}

func (s *Subfinder) Name() string {
	return "subfinder"
}

func (s *Subfinder) CheckInstalled() bool {
	_, err := exec.LookPath("subfinder")
	return err == nil
}

func (s *Subfinder) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install subfinder: %v", err)
	}
	return nil
}

func (s *Subfinder) Run(target string) (string, error) {
	utils.LogInfo("Running subfinder on %s...", target)
	// -d for domain, -silent for clean output
	cmd := exec.Command("subfinder", "-d", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("subfinder failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
