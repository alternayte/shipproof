package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

// runCoverage implements `shipproof coverage <change-id> [--json]`.
//
// The matrix is derived on demand. This command writes nothing. It never
// fails a change, whatever the states are, because coverage is review
// material.
func runCoverage(args []string, stdout, stderr io.Writer) int {
	changeID := ""
	asJSON := false
	for _, argument := range args {
		switch {
		case argument == "--json":
			asJSON = true
		case strings.HasPrefix(argument, "--"), changeID != "":
			fmt.Fprintln(stderr, "usage: shipproof coverage <change-id> [--json]")
			return 2
		default:
			changeID = argument
		}
	}
	if changeID == "" {
		fmt.Fprintln(stderr, "usage: shipproof coverage <change-id> [--json]")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if !requirements.Exists(root, changeID) {
		fmt.Fprintf(stderr, "no requirement set for %s; run `shipproof doc adopt %s --source <path>` first\n", changeID, changeID)
		return 1
	}

	plan, err := verification.Load(verification.Path(root, changeID))
	if err != nil {
		fmt.Fprintf(stderr, "invalid verification plan: %v\n", err)
		return 1
	}

	// A run with no recorded revision is not judgeable. The phase function
	// reads it as current, so that an unjudgeable run raises no false alarm.
	// The coverage matrix takes the opposite bias. A row must not make a
	// claim it cannot support, so an empty revision reads as not current.
	matrix, err := coverage.Read(root, changeID, plan)
	if err != nil {
		// coverage.Read wraps ErrRequirementSet or ErrProofResults. Branch on
		// the wrapped error so each cause prints its own top-level message,
		// the way the code did before the extraction.
		switch {
		case errors.Is(err, coverage.ErrRequirementSet):
			fmt.Fprintf(stderr, "invalid %v\n", err)
		case errors.Is(err, coverage.ErrProofResults):
			fmt.Fprintf(stderr, "invalid %v\n", err)
		default:
			fmt.Fprintf(stderr, "%v\n", err)
		}
		return 1
	}

	if asJSON {
		data, err := json.MarshalIndent(matrix, "", "  ")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", data)
		return 0
	}

	printMatrix(stdout, matrix)
	return 0
}

// printMatrix renders the matrix for a person.
func printMatrix(stdout io.Writer, matrix coverage.Matrix) {
	fmt.Fprintf(stdout, "Coverage — %s\n\n", matrix.ChangeID)

	width := 0
	for _, row := range matrix.Rows {
		if len(row.RequirementID) > width {
			width = len(row.RequirementID)
		}
	}

	counts := map[coverage.State]int{}
	for _, row := range matrix.Rows {
		counts[row.State]++
		fmt.Fprintf(stdout, "  %-*s  %-14s  %-8s  %s\n", width, row.RequirementID, row.State, row.Provenance, row.Detail)
	}

	fmt.Fprintln(stdout)
	for _, state := range []coverage.State{coverage.Proven, coverage.Failed, coverage.Accepted, coverage.AwaitingHuman, coverage.Unproven} {
		if counts[state] > 0 {
			fmt.Fprintf(stdout, "  %d %s\n", counts[state], state)
		}
	}
	if !matrix.RunCurrent {
		fmt.Fprintln(stdout, "\n  No proof result describes the working tree. Run `shipproof verification run`.")
	}
}
