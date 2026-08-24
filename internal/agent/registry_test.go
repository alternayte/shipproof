package agent

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRegisterAndBuild(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register("codex", func(config RunnerConfig) (AgentRunner, error) {
		return stubRunner{status: RunnerStatus{Version: config.Setting("model")}}, nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.Register("codex", func(RunnerConfig) (AgentRunner, error) { return nil, nil }); err == nil {
		t.Fatal("duplicate registration must fail")
	}

	runner, err := registry.Build(RunnerConfig{Name: "codex", Settings: map[string]string{"model": "o4"}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if status.Version != "o4" {
		t.Fatalf("settings did not reach the factory: %+v", status)
	}

	if _, err := registry.Build(RunnerConfig{Name: "missing"}); err == nil {
		t.Fatal("unknown runner must fail")
	}
}

func TestRegistryNamesAreSorted(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"opencode", "claude", "codex"} {
		registry.MustRegister(name, func(RunnerConfig) (AgentRunner, error) { return stubRunner{}, nil })
	}
	names := registry.Names()
	want := []string{"claude", "codex", "opencode"}
	if len(names) != len(want) {
		t.Fatalf("Names() = %v", names)
	}
	for index := range want {
		if names[index] != want[index] {
			t.Fatalf("Names() = %v, want %v", names, want)
		}
	}
}

func TestRegistryRejectsEmptyNameAndNilFactory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register("", func(RunnerConfig) (AgentRunner, error) { return stubRunner{}, nil }); err == nil {
		t.Fatal("empty name must fail")
	}
	if err := registry.Register("codex", nil); err == nil {
		t.Fatal("nil factory must fail")
	}
	if errors.Is(nil, ErrNoUsableRunner) {
		t.Fatal("unreachable")
	}
}
