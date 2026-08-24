package agent

import (
	"fmt"
	"sort"
	"sync"
)

// RunnerConfig holds the per-runner settings from the `agent.runners` section.
// It must never hold an API key.
type RunnerConfig struct {
	Name     string
	Settings map[string]string
}

// Setting returns one runner setting. A missing key returns an empty string.
func (config RunnerConfig) Setting(key string) string {
	if config.Settings == nil {
		return ""
	}
	return config.Settings[key]
}

// Factory builds one runner from its configuration.
type Factory func(config RunnerConfig) (AgentRunner, error)

// Registry maps a runner name to its factory.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{factories: map[string]Factory{}}
}

// Register adds a runner factory. A duplicate name returns an error.
func (registry *Registry) Register(name string, factory Factory) error {
	if name == "" {
		return fmt.Errorf("runner name is required")
	}
	if factory == nil {
		return fmt.Errorf("runner %q needs a factory", name)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.factories[name]; exists {
		return fmt.Errorf("runner %q is already registered", name)
	}
	registry.factories[name] = factory
	return nil
}

// MustRegister registers a runner and panics on a duplicate name.
func (registry *Registry) MustRegister(name string, factory Factory) {
	if err := registry.Register(name, factory); err != nil {
		panic(err)
	}
}

// Names returns every registered runner name in sorted order.
func (registry *Registry) Names() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	names := make([]string, 0, len(registry.factories))
	for name := range registry.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Build creates one runner by name.
func (registry *Registry) Build(config RunnerConfig) (AgentRunner, error) {
	registry.mu.RLock()
	factory, ok := registry.factories[config.Name]
	registry.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown runner %q", config.Name)
	}
	return factory(config)
}
