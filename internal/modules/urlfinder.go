package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Urlfinder struct{}

func (u *Urlfinder) Name() string {
	return "urlfinder"
}

func (u *Urlfinder) CheckInstalled() bool {
	_, err := exec.LookPath("urlfinder")
	return err == nil
}

func (u *Urlfinder) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/urlfinder/cmd/urlfinder@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install urlfinder: %v", err)
	}
	return nil
}

func (u *Urlfinder) Run(target string) (string, error) {
	utils.LogInfo("Running urlfinder on %s...", target)
	// -d domain -silent
	cmd := exec.Command("urlfinder", "-d", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("urlfinder failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
