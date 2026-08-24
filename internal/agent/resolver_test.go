package agent

import (
	"errors"
	"testing"
)

// P2: every precedence level of SDD Section 13.6 is covered.
func TestResolvePrecedence(t *testing.T) {
	full := ResolveInput{
		CLIOverride: "codex",
		Environment: "claude",
		RepoConfig:  "opencode",
		UserConfig:  "claude",
		Usable:      []string{"codex", "claude"},
	}

	cases := []struct {
		name       string
		input      ResolveInput
		wantName   string
		wantSource SelectionSource
	}{
		{"level 1 cli override", full, "codex", SourceCLI},
		{"level 2 environment", withoutCLI(full), "claude", SourceEnvironment},
		{"level 3 repository config", withoutEnvironment(withoutCLI(full)), "opencode", SourceRepoConfig},
		{"level 4 user config", ResolveInput{UserConfig: "claude", Usable: []string{"codex", "claude"}}, "claude", SourceUserConfig},
		{"level 5 auto selection", ResolveInput{Usable: []string{"codex"}}, "codex", SourceAuto},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			selection, err := Resolve(testCase.input)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if selection.Name != testCase.wantName || selection.Source != testCase.wantSource {
				t.Fatalf("Resolve() = %+v, want %s from %s", selection, testCase.wantName, testCase.wantSource)
			}
		})
	}
}

func TestResolveNoUsableRunner(t *testing.T) {
	if _, err := Resolve(ResolveInput{}); !errors.Is(err, ErrNoUsableRunner) {
		t.Fatalf("want ErrNoUsableRunner, got %v", err)
	}
}

func TestResolveAmbiguousRunner(t *testing.T) {
	_, err := Resolve(ResolveInput{Usable: []string{"codex", "claude"}})
	if !errors.Is(err, ErrAmbiguousRunner) {
		t.Fatalf("want ErrAmbiguousRunner, got %v", err)
	}
}

func TestResolveIgnoresBlankLevels(t *testing.T) {
	selection, err := Resolve(ResolveInput{CLIOverride: "   ", Environment: "  ", RepoConfig: "codex"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if selection.Source != SourceRepoConfig {
		t.Fatalf("Resolve() = %+v, want repository-config", selection)
	}
}

func withoutCLI(input ResolveInput) ResolveInput {
	input.CLIOverride = ""
	return input
}

func withoutEnvironment(input ResolveInput) ResolveInput {
	input.Environment = ""
	return input
}
