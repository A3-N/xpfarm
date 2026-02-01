package modules

import (
	"fmt"
	"os"
	"os/exec"
	"xpfarm/pkg/utils"
)

type Cvemap struct{}

func (c *Cvemap) Name() string {
	return "cvemap"
}

func (c *Cvemap) CheckInstalled() bool {
	_, err := exec.LookPath("vulnx") // Binary name is vulnx according to user
	if err != nil {
		_, err = exec.LookPath("cvemap") // Check both just in case
	}
	return err == nil
}

func (c *Cvemap) Install() error {
	cmd := exec.Command("go", "install", "github.com/projectdiscovery/cvemap/cmd/vulnx@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cvemap: %v", err)
	}
	return nil
}

func (c *Cvemap) Run(target string) (string, error) {
	utils.LogInfo("Running cvemap on %s...", target)
	// cvemap usually takes CVE IDs or search queries.
	// If target is a list of CVEs (comma separated) or a search query.
	// We'll pass it as -id if it looks like a CVE, or maybe just arguments.
	// Given the context is checking targets, cvemap might not be directly applicable to a domain target unless we found CVEs.
	// But let's assume valid usage for now or that target is a query.
	// Binary name might be `cvemap` or `vulnx`.
	bin := "vulnx"
	if _, err := exec.LookPath("cvemap"); err == nil {
		bin = "cvemap"
	} else if _, err := exec.LookPath("vulnx"); err != nil {
		return "", fmt.Errorf("cvemap/vulnx not found")
	}

	cmd := exec.Command(bin, "-id", target, "-silent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cvemap failed: %v\nOutput: %s", err, output)
	}
	return string(output), nil
}
