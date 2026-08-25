package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
var retiredSkills = []string{"verify-change"}

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

	if !keepRetired {
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
		if entry.IsDir() || path == "README.md" || strings.HasPrefix(path, "evals/") {
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
