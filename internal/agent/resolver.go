package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SelectionSource names the precedence level that selected a runner.
type SelectionSource string

const (
	SourceCLI         SelectionSource = "cli"
	SourceEnvironment SelectionSource = "environment"
	SourceRepoConfig  SelectionSource = "repository-config"
	SourceUserConfig  SelectionSource = "user-config"
	SourceAuto        SelectionSource = "auto"
)

// EnvRunner is the environment variable that selects a runner.
const EnvRunner = "SHIPPROOF_RUNNER"

var (
	// ErrNoUsableRunner reports that no runner is installed and authenticated.
	ErrNoUsableRunner = errors.New("no usable agent runner found")
	// ErrAmbiguousRunner reports that several runners are usable and no
	// default is configured. ShipProof must not silently choose a vendor.
	ErrAmbiguousRunner = errors.New("several usable runners found and no default is configured")
)

// ResolveInput holds one candidate per precedence level plus the set of
// runners that a probe found usable.
type ResolveInput struct {
	CLIOverride string
	Environment string
	RepoConfig  string
	UserConfig  string
	Usable      []string
}

// Selection is the resolved runner and the level that chose it.
type Selection struct {
	Name   string
	Source SelectionSource
}

// Resolve applies the SDD Section 13.6 precedence, highest first:
// CLI override, environment, repository config, user config, then
// auto-selection when exactly one usable runner exists.
func Resolve(input ResolveInput) (Selection, error) {
	levels := []struct {
		value  string
		source SelectionSource
	}{
		{input.CLIOverride, SourceCLI},
		{input.Environment, SourceEnvironment},
		{input.RepoConfig, SourceRepoConfig},
		{input.UserConfig, SourceUserConfig},
	}
	for _, level := range levels {
		name := strings.TrimSpace(level.value)
		if name != "" {
			return Selection{Name: name, Source: level.source}, nil
		}
	}

	usable := normalize(input.Usable)
	switch len(usable) {
	case 0:
		return Selection{}, ErrNoUsableRunner
	case 1:
		return Selection{Name: usable[0], Source: SourceAuto}, nil
	default:
		return Selection{}, fmt.Errorf("%w: %s", ErrAmbiguousRunner, strings.Join(usable, ", "))
	}
}

func normalize(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
