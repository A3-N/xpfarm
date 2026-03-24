// Package plugin is the XPFarm Plugin SDK.
//
// Community plugins implement Tool and/or Agent, call the Register* functions
// inside an init(), and are compiled into the binary via a blank import in
// plugins/all/all.go.  No runtime .so loading, no CGO — just clean Go.
package plugin

import "context"

// Tool is the smallest unit of work a plugin can expose.
// It receives a free-form input map and returns a free-form output map so the
// interface stays stable as the platform evolves.
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input map[string]any) (map[string]any, error)
}

// Agent owns a set of Tools and orchestrates them to accomplish a higher-level
// Task.  Agents may call multiple tools, chain results, or apply LLM reasoning.
type Agent interface {
	Name() string
	Tools() []Tool
	Handle(ctx context.Context, task Task) (Result, error)
}

// PipelineStep is one node in a named Pipeline.
// Set Agent XOR Tool — not both.
type PipelineStep struct {
	Name   string         `yaml:"name"`
	Agent  string         `yaml:"agent,omitempty"`
	Tool   string         `yaml:"tool,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`
}

// Task is the unit of work passed into an Agent.
type Task struct {
	ID      string
	Target  string
	Payload map[string]any
}

// Result is what an Agent returns after handling a Task.
// When Error is non-empty the caller should treat the run as failed.
type Result struct {
	Output map[string]any
	Error  string
}

// Manifest mirrors the fields in a plugin's plugin.yaml metadata file.
type Manifest struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
	Agents      []string `yaml:"agents"`
}
