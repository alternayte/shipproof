package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alternayte/shipproof/internal/verification"
)

// runProve validates the verification plan, runs the repository gate, and runs
// each proof on its own. It replaces the old `verify` and `verification run`
// verbs.
func runProve(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof prove [change-id] [--gate-only|--proofs-only]"

	changeID := ""
	passthrough := []string{}
	for _, argument := range args {
		switch argument {
		case "--gate-only", "--proofs-only":
			passthrough = append(passthrough, argument)
		default:
			if strings.HasPrefix(argument, "--") || changeID != "" {
				fmt.Fprintln(stderr, usage)
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

	// The plan check runs first. A plan that cannot judge a requirement must
	// never hide behind a passing gate. A change with no plan runs the gate
	// alone, and the run states why it ran no proof.
	if _, err := os.Stat(verification.Path(root, changeID)); err == nil {
		if code := checkPlan(root, changeID, stdout, stderr); code != 0 {
			return code
		}
	}

	return runVerificationRun(append([]string{changeID}, passthrough...), stdout, stderr)
}
