package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Interactsh struct{}

func (i *Interactsh) Name() string {
	return "interactsh"
}

func (i *Interactsh) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("interactsh-client")
	_, err := exec.LookPath(path)
	return err == nil
}

func (i *Interactsh) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/interactsh/cmd/interactsh-client@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install interactsh-client: %v", err)
	}
	return nil
}

func (i *Interactsh) Run(ctx context.Context, target string) (string, error) {
	// or we accept that this module might block.
	// Given the user wants a "tool wrapper", maybe they want to generate an OOB link?
	// We'll just run it as a check for now, or maybe not implement Run fully if it blocks.
	// "Tools to check if installed" - avoiding blocking run.
	return "", fmt.Errorf("interactsh is an interactive tool, standard run not supported in auto-mode yet")
}
