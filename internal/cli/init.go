package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/alternayte/shipproof/internal/harness"
	"github.com/alternayte/shipproof/internal/repository"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: shipproof init [directory]")
		return 2
	}

	target := "."
	if len(args) == 1 {
		target = args[0]
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(stderr, "resolve target directory: %v\n", err)
		return 1
	}

	result, err := repository.Initialize(abs)
	if err != nil {
		fmt.Fprintf(stderr, "initialize ShipProof repository: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Initialized ShipProof in %s\n", abs)
	if result.GateDetected {
		fmt.Fprintf(stdout, "Gate: %s. Change it in .shipproof/config.yaml.\n", result.Gate)
	} else {
		fmt.Fprintf(stdout, "Gate: %s. No build tool was found, so this command passes and proves nothing.\n", result.Gate)
		fmt.Fprintln(stdout, "      Put your test command in .shipproof/config.yaml under verification.command.")
	}
	fmt.Fprintf(stdout, "Created %d directories and %d files.\n", len(result.CreatedDirectories), len(result.CreatedFiles))
	if len(result.ExistingFiles) > 0 {
		fmt.Fprintf(stdout, "Kept %d existing files unchanged.\n", len(result.ExistingFiles))
	}
	return installInstructions(abs, stdout, stderr)
}

// installInstructions writes the ShipProof instructions into the format that
// each harness reads. A harness the repository does not use costs one unread
// directory, and that is cheaper than a missing instruction set.
func installInstructions(root string, stdout, stderr io.Writer) int {
	targets := []harness.Target{
		harness.TargetClaude,
		harness.TargetOpenCode,
		harness.TargetAgents,
	}
	for _, target := range targets {
		result, err := harness.Install(root, target, false, false)
		if err != nil {
			fmt.Fprintf(stderr, "install instructions for %s: %v\n", target, err)
			return 1
		}
		fmt.Fprintf(stdout, "Instructions for %s: %d files.\n", target, result.HarnessCreated)
	}
	return 0
}
