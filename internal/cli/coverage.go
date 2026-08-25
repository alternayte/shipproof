package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
	"github.com/alternayte/shipproof/internal/verify"
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
	set, err := requirements.Load(root, changeID)
	if err != nil {
		fmt.Fprintf(stderr, "invalid requirement set: %v\n", err)
		return 1
	}

	plan, err := verification.Load(verification.Path(root, changeID))
	if err != nil {
		fmt.Fprintf(stderr, "invalid verification plan: %v\n", err)
		return 1
	}

	var results *proofs.Set
	current := false
	if proofs.Exists(root, changeID) {
		loaded, err := proofs.Load(root, changeID)
		if err != nil {
			fmt.Fprintf(stderr, "invalid proof results: %v\n", err)
			return 1
		}
		results = &loaded
		current, _ = verify.IsCurrent(root, verify.Result{HeadRev: loaded.HeadRev, TreeClean: loaded.TreeClean})
	}

	matrix := coverage.Build(set, plan, results, current)

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
