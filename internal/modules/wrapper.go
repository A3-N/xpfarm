package modules

// Module defines the interface that all tool wrappers must implement.
type Module interface {
	// Name returns the name of the tool (e.g., "nuclei")
	Name() string

	// CheckInstalled returns true if the tool is available in PATH
	CheckInstalled() bool

	// Install attempts to install the tool (e.g., via go install)
	Install() error

	// Run executes the tool against a specific target.
	// target can be a domain, URL, or IP.
	// It returns the raw output or a path to the output file, and an error.
	Run(target string) (string, error)
}
