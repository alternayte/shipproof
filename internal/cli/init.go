package cli

import (
	"fmt"
	"io"
	"path/filepath"

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
	fmt.Fprintf(stdout, "Created %d directories and %d files.\n", len(result.CreatedDirectories), len(result.CreatedFiles))
	if len(result.ExistingFiles) > 0 {
		fmt.Fprintf(stdout, "Kept %d existing files unchanged.\n", len(result.ExistingFiles))
	}
	return 0
}
