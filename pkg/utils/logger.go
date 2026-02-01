package utils

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgHiBlue).SprintFunc() // Lighter blue for better visibility
	cyan   = color.New(color.FgCyan).SprintFunc()
)

func LogInfo(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", blue("[*]"), fmt.Sprintf(format, a...))
}

func LogSuccess(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", green("[+]"), fmt.Sprintf(format, a...))
}

func LogError(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", red("[!]"), fmt.Sprintf(format, a...))
}

func LogWarning(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", yellow("[-]"), fmt.Sprintf(format, a...))
}

func LogDebug(format string, a ...interface{}) {
	fmt.Printf("%s %s\n", cyan("[?]"), fmt.Sprintf(format, a...))
}
