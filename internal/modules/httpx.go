package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Httpx struct{}

func (h *Httpx) Name() string {
	return "httpx"
}

func (h *Httpx) CheckInstalled() bool {
	_, err := exec.LookPath("httpx")
	return err == nil
}

func (h *Httpx) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/httpx/cmd/httpx@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install httpx: %v", err)
	}
	return nil
}

func (h *Httpx) Run(target string) (string, error) {
	utils.LogInfo("Running httpx on %s...", target)
	// -u target -silent
	cmd := exec.Command("httpx", "-u", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("httpx failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
