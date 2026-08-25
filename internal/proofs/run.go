package proofs

import (
	"os"
	"os/exec"
	"time"

	"github.com/alternayte/shipproof/internal/verification"
)

// Run executes every proof in the plan on its own and returns the result set.
//
// This is the attribution pass. It does not replace the repository gate. The
// gate stays the authority on whether the repository passes, because it
// catches faults that no requirement names. A green attribution never masks a
// red gate.
//
// A human proof runs no command. ShipProof records that a person owns it and
// claims nothing further.
func Run(root string, plan verification.Plan, headRev string, treeClean *bool) (Set, error) {
	results := []Result{}
	for _, group := range [][]verification.Item{plan.Requirements, plan.Invariants} {
		for _, item := range group {
			for index, proof := range item.Proof {
				results = append(results, runProof(root, item.ID, index, proof))
			}
		}
	}
	return Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      plan.ChangeID,
		HeadRev:       headRev,
		TreeClean:     treeClean,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Results:       results,
	}, nil
}

// runProof executes one proof and records what it observed. A command that the
// shell cannot start counts as a failure with the exit code the shell reports.
func runProof(root, requirementID string, index int, proof verification.Proof) Result {
	if !proof.IsAutomated() {
		return Result{RequirementID: requirementID, ProofIndex: index, Status: Human}
	}

	start := time.Now()
	command := exec.Command("sh", "-c", proof.Command)
	command.Dir = root
	command.Env = os.Environ()
	err := command.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 127
		}
	}

	status := Pass
	if exitCode != 0 {
		status = Fail
	}
	return Result{
		RequirementID: requirementID,
		ProofIndex:    index,
		Command:       proof.Command,
		ExitCode:      exitCode,
		DurationMs:    duration.Milliseconds(),
		Status:        status,
	}
}
