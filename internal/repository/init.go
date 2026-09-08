package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InitResult struct {
	CreatedDirectories []string
	CreatedFiles       []string
	ExistingFiles      []string
	// Gate is the verification command that init chose for this repository.
	Gate string
	// GateDetected reports whether a build tool in the repository named the
	// command. A false value means that init fell back to a command that
	// always passes, and the user must replace it.
	GateDetected bool
}

var initialFiles = map[string]string{
	".shipproof/config.yaml": `version: 1
schema_version: "0.1"
verification:
  command: {{gate}}
  # coverage:
  #   command: go test -coverpkg=./... -coverprofile={{profile}} ./{{target}}/
  #   format: go
  # unexplained_ignore:
  #   - "docs/**"
evidence:
  capture: metadata
`,
	".shipproof/templates/prd.md": `# <Product or feature name>

## Problem
Describe the observed problem, pain, or explicit hypothesis.

## Users and desired outcome
Name the affected actor. State the observable outcome.

## Scope and appetite
State material boundaries. Add an appetite when it helps constrain the solution.

## Requirements and acceptance
Describe important behavior and how it can be evaluated.

## Assumptions, risks, and unknowns
Record only material uncertainty. Remove this section when none remains.
`,
	".shipproof/templates/sdd.md": `# <Design title>

## Intent and context
Link the design to approved product intent and the affected system boundary.

## Design
Describe the smallest design that satisfies the intent.

## Decisions and rationale
Explain material choices and trade-offs. Do not catalogue patterns.

## Relevant failure and operational behavior
Include only the failure, data, concurrency, security, migration, or operational concerns that apply.

## Verification
Explain how important requirements and invariants can be proven or inspected.

## Assumptions, risks, and unknowns
Record remaining material uncertainty.
`,
}

func Initialize(root string) (InitResult, error) {
	var result InitResult

	info, err := os.Stat(root)
	switch {
	case err == nil && !info.IsDir():
		return result, fmt.Errorf("target exists and is not a directory: %s", root)
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(root, 0o755); err != nil {
			return result, fmt.Errorf("create target directory: %w", err)
		}
	case err != nil:
		return result, fmt.Errorf("inspect target directory: %w", err)
	}

	for _, path := range directoryPaths(root) {
		_, statErr := os.Stat(path)
		existed := statErr == nil
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return result, fmt.Errorf("inspect directory %s: %w", path, statErr)
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return result, fmt.Errorf("create directory %s: %w", path, err)
		}
		if !existed {
			result.CreatedDirectories = append(result.CreatedDirectories, path)
		}
	}

	gate, detected := DetectGate(root)
	result.Gate = gate
	result.GateDetected = detected

	for relative, contents := range initialFiles {
		contents = strings.ReplaceAll(contents, "{{gate}}", gate)
		path := filepath.Join(root, filepath.FromSlash(relative))
		if _, err := os.Stat(path); err == nil {
			result.ExistingFiles = append(result.ExistingFiles, path)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("inspect file %s: %w", path, err)
		}

		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			return result, fmt.Errorf("write file %s: %w", path, err)
		}
		result.CreatedFiles = append(result.CreatedFiles, path)
	}

	return result, nil
}

// gateCandidate names a build tool, the file that proves the repository uses
// it, and the command that runs its checks.
type gateCandidate struct {
	Marker  string
	Command string
}

// gateCandidates are tried in order. The first marker that exists wins.
var gateCandidates = []gateCandidate{
	{"justfile", "just verify"},
	{"Justfile", "just verify"},
	{"Makefile", "make test"},
	{"package.json", "npm test"},
	{"go.mod", "go test ./..."},
	{"Cargo.toml", "cargo test"},
	{"pyproject.toml", "pytest"},
}

// DetectGate picks a verification command that the repository can actually
// run. The first verdict a new user sees must never read FAILED because of a
// build tool they do not use.
//
// A directory that matches nothing gets a command that always passes. It
// proves nothing, and the second result reports that honestly, which is better
// than a failure the user cannot act on.
func DetectGate(root string) (command string, detected bool) {
	for _, candidate := range gateCandidates {
		if _, err := os.Stat(filepath.Join(root, candidate.Marker)); err == nil {
			return candidate.Command, true
		}
	}
	return "true", false
}
