package plugin

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	tools     = make(map[string]Tool)
	agents    = make(map[string]Agent)
	pipelines = make(map[string][]PipelineStep)
	manifests []Manifest
)

// RegisterTool adds a Tool to the global registry.
// Panics on duplicate names so mistakes surface immediately at startup.
func RegisterTool(t Tool) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := tools[t.Name()]; exists {
		panic(fmt.Sprintf("plugin: duplicate tool name %q", t.Name()))
	}
	tools[t.Name()] = t
}

// RegisterAgent adds an Agent to the global registry.
// Panics on duplicate names.
func RegisterAgent(a Agent) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := agents[a.Name()]; exists {
		panic(fmt.Sprintf("plugin: duplicate agent name %q", a.Name()))
	}
	agents[a.Name()] = a
}

// RegisterPipeline stores a named sequence of PipelineSteps.
// Calling RegisterPipeline twice with the same name overwrites silently.
func RegisterPipeline(name string, steps []PipelineStep) {
	mu.Lock()
	defer mu.Unlock()
	pipelines[name] = steps
}

// RegisterManifest records a plugin's parsed YAML metadata.
func RegisterManifest(m Manifest) {
	mu.Lock()
	defer mu.Unlock()
	manifests = append(manifests, m)
}

// --- Getters -----------------------------------------------------------------

func GetTool(name string) (Tool, bool) {
	mu.RLock()
	defer mu.RUnlock()
	t, ok := tools[name]
	return t, ok
}

func GetAgent(name string) (Agent, bool) {
	mu.RLock()
	defer mu.RUnlock()
	a, ok := agents[name]
	return a, ok
}

func GetPipeline(name string) ([]PipelineStep, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := pipelines[name]
	return p, ok
}

func AllTools() []Tool {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, t)
	}
	return out
}

func AllAgents() []Agent {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Agent, 0, len(agents))
	for _, a := range agents {
		out = append(out, a)
	}
	return out
}

// AllPipelines returns a copy of the pipeline map.
func AllPipelines() map[string][]PipelineStep {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string][]PipelineStep, len(pipelines))
	for k, v := range pipelines {
		out[k] = v
	}
	return out
}

func AllManifests() []Manifest {
	mu.RLock()
	defer mu.RUnlock()
	cp := make([]Manifest, len(manifests))
	copy(cp, manifests)
	return cp
}
