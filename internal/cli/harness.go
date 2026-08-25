package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alternayte/shipproof/internal/harness"
)

func runHarness(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || args[0] != "install" {
		fmt.Fprintln(stderr, "usage: shipproof harness install <claude|cursor|codex|opencode|agents> [directory] [--force] [--keep-retired]")
		return 2
	}
	target, err := harness.ParseTarget(args[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	root := "."
	force := false
	keepRetired := false
	for _, arg := range args[2:] {
		if arg == "--force" {
			force = true
			continue
		}
		if arg == "--keep-retired" {
			keepRetired = true
			continue
		}
		if root != "." {
			fmt.Fprintln(stderr, "only one directory can be specified")
			return 2
		}
		root = arg
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve directory: %v\n", err)
		return 1
	}
	if _, err := os.Stat(filepath.Join(abs, ".shipproof")); err != nil {
		fmt.Fprintln(stderr, "target is not initialized; run shipproof init first")
		return 1
	}
	result, err := harness.Install(abs, target, force, keepRetired)
	if err != nil {
		fmt.Fprintf(stderr, "install skills: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Installed ShipProof skills for %s. Canonical files: %d, harness files: %d.\n", target, result.CanonicalCreated, result.HarnessCreated)
	if len(result.Existing) > 0 {
		fmt.Fprintf(stdout, "Kept %d identical existing files.\n", len(result.Existing))
	}
	for _, directory := range result.Retired {
		fmt.Fprintf(stdout, "Removed retired skill: %s\n", directory)
	}
	return 0
}
