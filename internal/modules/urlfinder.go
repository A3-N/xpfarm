package modules

import (
	"context"
	"fmt"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Urlfinder struct{}

func (u *Urlfinder) Name() string {
	return "urlfinder"
}

func (u *Urlfinder) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("urlfinder")
	_, err := exec.LookPath(path)
	return err == nil
}

func (u *Urlfinder) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/urlfinder/cmd/urlfinder@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install urlfinder: %v", err)
	}
	return nil
}

func (u *Urlfinder) Run(ctx context.Context, target string) (string, error) {
	utils.LogInfo("Running urlfinder on %s...", target)
	// -d target -silent -all
	path := utils.ResolveBinaryPath("urlfinder")
	cmd := exec.CommandContext(ctx, path, "-d", target, "-silent", "-all")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("urlfinder failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
