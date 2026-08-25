package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type InitResult struct {
	CreatedDirectories []string
	CreatedFiles       []string
	ExistingFiles      []string
}

var initialFiles = map[string]string{
	".shipproof/config.yaml": `version: 1
schema_version: "0.1"
verification:
  command: just verify
  # coverage:
  #   command: go test -coverpkg=./... -coverprofile={{profile}} ./{{target}}/
  #   format: go
  # unexplained_ignore:
  #   - "docs/**"
evidence:
  capture: metadata
language:
  profile: ste-assisted
  procedural_sentence_max_words: 20
  descriptive_sentence_max_words: 25
`,
	".shipproof/glossary.yaml": `technical_names:
  - ShipProof
technical_verbs:
  - verify
  - deploy
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

	for relative, contents := range initialFiles {
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
