package proofs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alternayte/shipproof/internal/verification"
)

// Coverage holds the measurement pass configuration.
//
// The measurement runs beside a proof, not in place of it. The template
// carries no proof command, so it cannot be the proof. A failed measurement
// changes no recorded result, because the signal never blocks a run.
type Coverage struct {
	Command string
	Dir     string
}

// MergedProfilePath returns the merged profile location for one change.
func MergedProfilePath(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "runs", changeID, "coverage", "merged.out")
}

// CoverageDir returns the per-proof profile directory for one change.
func CoverageDir(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "runs", changeID, "coverage")
}

// Run executes every proof in the plan on its own and returns the result set.
//
// This is the attribution pass. It does not replace the repository gate. The
// gate stays the authority on whether the repository passes, because it
// catches faults that no requirement names. A green attribution never masks a
// red gate.
//
// A human proof runs no command. ShipProof records that a person owns it and
// claims nothing further.
func Run(root string, plan verification.Plan, headRev string, treeClean *bool, coverage *Coverage) (Set, error) {
	if coverage != nil {
		if err := os.MkdirAll(coverage.Dir, 0o755); err != nil {
			return Set{}, fmt.Errorf("create coverage directory: %w", err)
		}
	}
	results := []Result{}
	for _, group := range [][]verification.Item{plan.Requirements, plan.Invariants} {
		for _, item := range group {
			for index, proof := range item.Proof {
				results = append(results, runProof(root, item.ID, index, proof))
				measure(root, item.ID, index, proof, coverage)
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
// shell cannot start counts as a failure with the exit code 127, because the
// shell never started and reported no code.
//
// This is a deliberate asymmetry with the gate. A proof that cannot start is
// attributed to its requirement as a failure. A gate that cannot start returns
// an error, because no requirement owns the gate.
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

// measure runs the coverage command for one proof and writes one profile. It
// reports nothing. A measurement that fails leaves no profile, and the report
// then states that it could not judge those lines.
func measure(root, requirementID string, index int, proof verification.Proof, coverage *Coverage) {
	if coverage == nil || !proof.IsAutomated() || strings.TrimSpace(proof.Target) == "" {
		return
	}
	profilePath := filepath.Join(coverage.Dir, fmt.Sprintf("%s-%d.out", requirementID, index))
	expanded := strings.ReplaceAll(coverage.Command, "{{profile}}", profilePath)
	expanded = strings.ReplaceAll(expanded, "{{target}}", proof.Target)

	command := exec.Command("sh", "-c", expanded)
	command.Dir = root
	command.Env = os.Environ()
	_ = command.Run()
}
