package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Interactsh struct{}

func (i *Interactsh) Name() string {
	return "interactsh"
}

func (i *Interactsh) CheckInstalled() bool {
	_, err := exec.LookPath("interactsh-client")
	return err == nil
}

func (i *Interactsh) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/interactsh/cmd/interactsh-client@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install interactsh-client: %v", err)
	}
	return nil
}

func (i *Interactsh) Run(target string) (string, error) {
	// Interactsh is usually a listener. Running it against a target might mean generating a payload?
	// Or maybe starting the client to listen.
	// For now, we'll assume we start the client for a bit or maybe this isn't the right way to use it in a scanner.
	// But as a wrapper, we'll just expose the ability to run it.
	// Maybe just `interactsh-client -v`
	// Must add imports for utils package manually if not replaced by auto-imports
	utils.LogInfo("Starting interactsh-client (not blocking)...")
	// cmd := exec.Command("interactsh-client", "-v")
	// This creates a long running process. We might not want to wait for it.
	// For the sake of "wrapper", we'll just check version or similar to verify it runs,
	// or we accept that this module might block.
	// Given the user wants a "tool wrapper", maybe they want to generate an OOB link?
	// We'll just run it as a check for now, or maybe not implement Run fully if it blocks.
	// "Tools to check if installed" - avoiding blocking run.
	return "", fmt.Errorf("interactsh is an interactive tool, standard run not supported in auto-mode yet")
}
