package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	skillassets "github.com/alternayte/shipproof/skills"
)

type Target string

const (
	TargetClaude   Target = "claude"
	TargetCursor   Target = "cursor"
	TargetCodex    Target = "codex"
	TargetOpenCode Target = "opencode"
	TargetAgents   Target = "agents"
)

type InstallResult struct {
	CanonicalCreated int
	HarnessCreated   int
	Existing         []string
	Retired          []string
}

// retiredSkills name skill directories that ShipProof no longer ships. Install
// removes them by default. A retired skill left in a harness directory still
// answers to an agent, which is worse than the loss of a local edit to a skill
// that no longer exists.
var retiredSkills = []string{
	// SP-033 replaced these six with the three instruction files of Section
	// 8.2.
	"prepare-change",
	"plan-verification",
	"implement-change",
	"produce-evidence",
	"review-change",
	"prepare-human-review",
	"verify-change",
	"shape-prd",
	"shape-sdd",
	"review-prd",
	"review-sdd",
	"decompose-plan",
	"triage-change",
	"record-decision",
	"benchmark-run",
}

// retiredLooseFiles name the instruction files that SP-033 wrote directly into
// a skills directory. SP-041 replaced each with a package, and a harness never
// read the loose form.
var retiredLooseFiles = []string{
	"capture-intent.md",
	"plan-proof.md",
	"read-evidence.md",
}

func ParseTarget(value string) (Target, error) {
	switch Target(value) {
	case TargetClaude, TargetCursor, TargetCodex, TargetOpenCode, TargetAgents:
		return Target(value), nil
	default:
		return "", fmt.Errorf("unsupported harness %q; use claude, cursor, codex, opencode, or agents", value)
	}
}

func targetDirectory(root string, target Target) string {
	switch target {
	case TargetClaude:
		return filepath.Join(root, ".claude", "skills")
	case TargetOpenCode:
		return filepath.Join(root, ".opencode", "skills")
	case TargetCursor, TargetCodex, TargetAgents:
		return filepath.Join(root, ".agents", "skills")
	default:
		return ""
	}
}

func Install(root string, target Target, force bool, keepRetired bool) (InstallResult, error) {
	var result InstallResult
	canonicalRoot := filepath.Join(root, ".shipproof", "skills")
	harnessRoot := targetDirectory(root, target)
	if harnessRoot == "" {
		return result, errors.New("invalid harness target")
	}
	if err := os.MkdirAll(canonicalRoot, 0o755); err != nil {
		return result, fmt.Errorf("create canonical skills directory: %w", err)
	}
	if err := os.MkdirAll(harnessRoot, 0o755); err != nil {
		return result, fmt.Errorf("create harness skills directory: %w", err)
	}

	// SP-041 moved each instruction into its own package. A loose Markdown
	// file from an earlier version is worse than no file: no harness reads it
	// and it looks installed.
	if !keepRetired {
		for _, name := range retiredLooseFiles {
			for _, base := range []string{canonicalRoot, harnessRoot} {
				path := filepath.Join(base, name)
				if _, err := os.Stat(path); err != nil {
					continue
				}
				if err := os.Remove(path); err != nil {
					return result, fmt.Errorf("remove the loose instruction %s: %w", name, err)
				}
				result.Retired = append(result.Retired, path)
			}
		}
		for _, name := range retiredSkills {
			for _, base := range []string{canonicalRoot, harnessRoot} {
				directory := filepath.Join(base, name)
				if _, err := os.Stat(directory); err != nil {
					continue
				}
				if err := os.RemoveAll(directory); err != nil {
					return result, fmt.Errorf("remove retired skill %s: %w", name, err)
				}
				result.Retired = append(result.Retired, directory)
			}
		}
	}

	var sourceFiles []string
	if err := fs.WalkDir(skillassets.Catalog, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || path == "README.md" {
			return nil
		}
		sourceFiles = append(sourceFiles, path)
		return nil
	}); err != nil {
		return result, fmt.Errorf("walk built-in skills: %w", err)
	}
	sort.Strings(sourceFiles)

	for _, source := range sourceFiles {
		contents, err := skillassets.Catalog.ReadFile(source)
		if err != nil {
			return result, fmt.Errorf("read built-in skill %s: %w", source, err)
		}
		for _, destinationRoot := range []struct {
			path      string
			canonical bool
		}{{canonicalRoot, true}, {harnessRoot, false}} {
			destination := filepath.Join(destinationRoot.path, filepath.FromSlash(source))
			created, existing, err := writeManagedFile(destination, contents, force)
			if err != nil {
				return result, err
			}
			if existing {
				result.Existing = append(result.Existing, destination)
			}
			if created {
				if destinationRoot.canonical {
					result.CanonicalCreated++
				} else {
					result.HarnessCreated++
				}
			}
		}
	}
	return result, nil
}

func writeManagedFile(path string, contents []byte, force bool) (created bool, existing bool, err error) {
	current, readErr := os.ReadFile(path)
	if readErr == nil {
		if string(current) == string(contents) {
			return false, true, nil
		}
		if !force {
			return false, false, fmt.Errorf("refusing to overwrite modified skill file %s; rerun with --force after reviewing the difference", path)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return false, false, fmt.Errorf("inspect skill file %s: %w", path, readErr)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, false, fmt.Errorf("create skill directory: %w", err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return false, false, fmt.Errorf("write skill file %s: %w", path, err)
	}
	return true, false, nil
}
