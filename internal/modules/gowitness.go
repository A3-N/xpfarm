package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Gowitness struct{}

func (g *Gowitness) Name() string {
	return "gowitness"
}

func (g *Gowitness) CheckInstalled() bool {
	_, err := exec.LookPath("gowitness")
	return err == nil
}

func (g *Gowitness) Install() error {
	cmd := exec.Command("go", "install", "github.com/sensepost/gowitness@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install gowitness: %v", err)
	}
	return nil
}

func (g *Gowitness) Run(target string) (string, error) {
	utils.LogInfo("Running gowitness on %s...", target)
	// single scan: single -u target
	cmd := exec.Command("gowitness", "single", "-u", target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gowitness failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
