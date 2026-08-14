package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/change"
	"github.com/shipproof/shipproof/internal/evidence/pack"
)

func runEvidence(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof evidence <pack> ...")
		return 2
	}

	switch args[0] {
	case "pack":
		return runEvidencePack(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown evidence command %q\n", args[0])
		return 2
	}
}

func runEvidencePack(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: shipproof evidence pack <change-id> [--base <rev>] [--head <rev>] [--evidence <file>...]")
		return 2
	}

	changeID := args[0]
	var opts pack.Options

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--base":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--base requires a revision")
				return 2
			}
			i++
			opts.BaseRev = args[i]
		case "--head":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--head requires a revision")
				return 2
			}
			i++
			opts.HeadRev = args[i]
		case "--evidence":
			for i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				i++
				opts.EvidenceFiles = append(opts.EvidenceFiles, args[i])
			}
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[i])
			return 2
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	assembled, err := pack.Assemble(root, changeID, opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := pack.WritePack(root, assembled); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")
	rel, _ := filepath.Rel(root, packPath)
	fmt.Fprintf(stdout, "Evidence pack written: %s\n", filepath.ToSlash(rel))
	return 0
}
