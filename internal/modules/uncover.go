package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Uncover struct{}

func (u *Uncover) Name() string {
	return "uncover"
}

func (u *Uncover) CheckInstalled() bool {
	_, err := exec.LookPath("uncover")
	return err == nil
}

func (u *Uncover) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/uncover/cmd/uncover@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install uncover: %v", err)
	}
	return nil
}

func (u *Uncover) Run(target string) (string, error) {
	utils.LogInfo("Running uncover on %s...", target)
	// uncover usually needs queries, but for a default run we might just check basic info or skip
	// For this wrapper, we'll assume target is a query or we might need to adjust logic later.
	// As a placeholder, we'll run it with -q (query) if target looks like a query, or throw error.
	// However, the user flow implies "targets" are companies/domains. Uncover uses API keys to search shodan/censys etc.
	// We'll assume target is a domain and search for it.
	cmd := exec.Command("uncover", "-q", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("uncover failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
