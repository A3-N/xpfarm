// Package echoplugin is the example-echo XPFarm plugin.
//
// It registers:
//   - EchoTool   — reflects its input map back as output
//   - EchoAgent  — wraps EchoTool and handles a Task
//   - echo-pipeline — a single-step pipeline backed by EchoAgent
//
// Copy this package as a starting point for your own plugin.
package echoplugin

import (
	"context"
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
	"xpfarm/internal/plugin"
)

//go:embed plugin.yaml
var rawManifest []byte

func init() {
	var m plugin.Manifest
	if err := yaml.Unmarshal(rawManifest, &m); err != nil {
		panic(fmt.Sprintf("example-echo: malformed plugin.yaml: %v", err))
	}
	plugin.RegisterManifest(m)
	plugin.RegisterTool(&EchoTool{})
	plugin.RegisterAgent(&EchoAgent{tool: &EchoTool{}})
	plugin.RegisterPipeline("echo-pipeline", []plugin.PipelineStep{
		{
			Name:   "reflect-input",
			Agent:  "echo-agent",
			Params: map[string]any{"mode": "passthrough"},
		},
	})
}

// -----------------------------------------------------------------------------
// EchoTool
// -----------------------------------------------------------------------------

// EchoTool returns its entire input under the key "echo".
// It demonstrates the minimum viable Tool implementation.
type EchoTool struct{}

func (t *EchoTool) Name() string { return "echo" }
func (t *EchoTool) Description() string {
	return "Reflects input back as output. Useful for pipeline testing."
}

func (t *EchoTool) Run(_ context.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{"echo": input}, nil
}

// -----------------------------------------------------------------------------
// EchoAgent
// -----------------------------------------------------------------------------

// EchoAgent wraps EchoTool and demonstrates the Agent interface.
type EchoAgent struct {
	tool *EchoTool
}

func (a *EchoAgent) Name() string         { return "echo-agent" }
func (a *EchoAgent) Tools() []plugin.Tool { return []plugin.Tool{a.tool} }

func (a *EchoAgent) Handle(ctx context.Context, task plugin.Task) (plugin.Result, error) {
	// Merge Task.Target into the payload so callers can omit it from Payload.
	input := make(map[string]any, len(task.Payload)+1)
	for k, v := range task.Payload {
		input[k] = v
	}
	input["target"] = task.Target

	out, err := a.tool.Run(ctx, input)
	if err != nil {
		return plugin.Result{Error: err.Error()}, err
	}
	return plugin.Result{Output: out}, nil
}
