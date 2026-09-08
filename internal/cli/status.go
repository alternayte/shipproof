package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/phase"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

// runStatus prints the phase, the blocker, the next command, and the
// requirement coverage. It replaces the old `next`, `change status`,
// `change check`, and `coverage` verbs.
func runStatus(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof status [change-id] [--json]"

	changeID := ""
	asJSON := false
	for _, argument := range args {
		switch argument {
		case "--json":
			asJSON = true
		default:
			if strings.HasPrefix(argument, "--") {
				fmt.Fprintf(stderr, "unknown option %q\n", argument)
				return 2
			}
			if changeID != "" {
				fmt.Fprintf(stderr, "unexpected argument %q\n", argument)
				return 2
			}
			changeID = argument
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if changeID == "" {
		resolved, open, err := soleOpenChange(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			for _, candidate := range open {
				fmt.Fprintf(stderr, "  %s\n", candidate)
			}
			return 2
		}
		changeID = resolved
	}

	result, err := phase.Resolve(root, changeID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	// The matrix is best effort. A change with no plan and no run still has a
	// phase and a blocker, and those answer the user's question.
	matrix, matrixErr := readCoverage(root, changeID)

	if asJSON {
		payload := struct {
			phase.Result
			Coverage *coverage.Matrix `json:"coverage,omitempty"`
		}{Result: result}
		if matrixErr == nil {
			payload.Coverage = &matrix
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode result: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", data)
		return 0
	}

	fmt.Fprintf(stdout, "change    %s\n", result.ChangeID)
	fmt.Fprintf(stdout, "phase     %s\n", result.Phase)
	if result.Blocker != "" {
		fmt.Fprintf(stdout, "blocker   %s\n", result.Blocker)
	}
	if result.NextCommand != "" {
		fmt.Fprintf(stdout, "next      %s\n", result.NextCommand)
	}
	if result.NextSkill != "" {
		fmt.Fprintf(stdout, "skill     %s\n", result.NextSkill)
	}
	if matrixErr != nil {
		// An absent artifact is a state. A malformed artifact is a fault, and
		// a fault must never read as a clean status.
		if errors.Is(matrixErr, errNoRequirementSet) {
			fmt.Fprintf(stdout, "coverage  none. %v\n", matrixErr)
			return 0
		}
		// coverage.Read wraps ErrRequirementSet or ErrProofResults. Branch on
		// the wrapped error, so each cause names itself.
		switch {
		case errors.Is(matrixErr, coverage.ErrRequirementSet):
			fmt.Fprintf(stderr, "invalid %v\n", matrixErr)
		case errors.Is(matrixErr, coverage.ErrProofResults):
			fmt.Fprintf(stderr, "invalid %v\n", matrixErr)
		default:
			fmt.Fprintf(stderr, "%v\n", matrixErr)
		}
		return 1
	}
	fmt.Fprintln(stdout)
	printMatrix(stdout, matrix)
	return 0
}

// readCoverage builds the requirement coverage matrix. It reports the reason
// when the change does not yet hold enough state to build one.
func readCoverage(root, changeID string) (coverage.Matrix, error) {
	if !requirements.Exists(root, changeID) {
		return coverage.Matrix{}, errNoRequirementSet
	}
	plan, err := verification.Load(verification.Path(root, changeID))
	if err != nil {
		return coverage.Matrix{}, err
	}
	return coverage.Read(root, changeID, plan)
}

// errNoRequirementSet reports a change that holds no requirement set. It is a
// state, not a fault.
var errNoRequirementSet = errors.New("the change holds no requirement set")
