// Package phase derives the current delivery phase of a ShipProof change from
// the artifacts on disk. The phase is a pure function of those artifacts.
// ShipProof stores no cursor, so an agent that acts out of band cannot
// desynchronize the answer.
package phase

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/verification"
	"github.com/alternayte/shipproof/internal/verify"
)

type Phase string

const (
	NoChange      Phase = "NO_CHANGE"
	IntentStale   Phase = "INTENT_STALE"
	NeedsPlan     Phase = "NEEDS_PLAN"
	NeedsRun      Phase = "NEEDS_RUN"
	RunStale      Phase = "RUN_STALE"
	RunFailed     Phase = "RUN_FAILED"
	NeedsEvidence Phase = "NEEDS_EVIDENCE"
	ReadyForHuman Phase = "READY_FOR_HUMAN"
)

// Result names the phase, the blocker that holds it open, the exact next
// command, and the skill that handles it.
type Result struct {
	ChangeID    string `json:"change_id"`
	Phase       Phase  `json:"phase"`
	Blocker     string `json:"blocker,omitempty"`
	NextCommand string `json:"next_command,omitempty"`
	// NextInstruction names the instruction file of Section 8.2 that covers
	// the next step. Three files exist, and this field names one of them.
	NextInstruction string `json:"next_instruction,omitempty"`
}

// Resolve returns the first phase whose condition holds. A malformed artifact
// is an error, never a phase. Reporting a corrupt record as NO_CHANGE would
// hide the corruption.
func Resolve(root, changeID string) (Result, error) {
	recordPath := change.Path(root, changeID)
	if _, err := os.Stat(recordPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				ChangeID:        changeID,
				Phase:           NoChange,
				Blocker:         "no change record exists",
				NextCommand:     fmt.Sprintf("shipproof start %s --intent <path>", changeID),
				NextInstruction: "capture-intent",
			}, nil
		}
		return Result{}, fmt.Errorf("inspect change record: %w", err)
	}

	record, err := change.Load(root, changeID)
	if err != nil {
		return Result{}, fmt.Errorf("load change record: %w", err)
	}

	staleness, err := record.Staleness(root)
	if err != nil {
		return Result{}, fmt.Errorf("check intent staleness: %w", err)
	}
	if staleness.Stale {
		return Result{
			ChangeID:        changeID,
			Phase:           IntentStale,
			Blocker:         fmt.Sprintf("source %s changed after the snapshot", record.SourcePath),
			NextCommand:     fmt.Sprintf("shipproof start %s --intent %s --force", changeID, record.SourcePath),
			NextInstruction: "capture-intent",
		}, nil
	}

	level := record.CeremonyLevel()

	if level >= 1 {
		planResult, held, err := resolvePlan(root, changeID)
		if err != nil {
			return Result{}, err
		}
		if held {
			return planResult, nil
		}
	}

	runResult, held, err := resolveRun(root, changeID)
	if err != nil {
		return Result{}, err
	}
	if held {
		return runResult, nil
	}

	if !fileExists(artifactPath(root, changeID, "evidence-pack.json")) {
		return Result{
			ChangeID:        changeID,
			Phase:           NeedsEvidence,
			Blocker:         "no evidence pack exists for the current run",
			NextCommand:     fmt.Sprintf("shipproof pack %s", changeID),
			NextInstruction: "read-evidence",
		}, nil
	}

	return Result{
		ChangeID:        changeID,
		Phase:           ReadyForHuman,
		NextInstruction: "read-evidence",
	}, nil
}

// resolveRun reports the first run phase that holds. RUN_STALE precedes
// RUN_FAILED by intent. The exit code of a stale run describes a revision that
// no longer matches the working tree.
func resolveRun(root, changeID string) (Result, bool, error) {
	runPath := filepath.Join(verify.RunDir(root, changeID), "run.json")
	data, err := os.ReadFile(runPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				ChangeID:        changeID,
				Phase:           NeedsRun,
				Blocker:         "no run record exists",
				NextCommand:     fmt.Sprintf("shipproof prove %s", changeID),
				NextInstruction: "plan-proof",
			}, true, nil
		}
		return Result{}, false, fmt.Errorf("read run result: %w", err)
	}

	var run verify.Result
	if err := json.Unmarshal(data, &run); err != nil {
		return Result{}, false, fmt.Errorf("parse run result: %w", err)
	}

	if stale, reason := runIsStale(root, run); stale {
		return Result{
			ChangeID:        changeID,
			Phase:           RunStale,
			Blocker:         reason,
			NextCommand:     fmt.Sprintf("shipproof prove %s", changeID),
			NextInstruction: "plan-proof",
		}, true, nil
	}

	if run.ExitCode != 0 {
		return Result{
			ChangeID:        changeID,
			Phase:           RunFailed,
			Blocker:         fmt.Sprintf("the newest run exited with code %d", run.ExitCode),
			NextCommand:     fmt.Sprintf("shipproof prove %s", changeID),
			NextInstruction: "plan-proof",
		}, true, nil
	}

	return Result{}, false, nil
}

// runIsStale delegates to the shared currency rule in internal/verify. One
// rule, one place; the coverage matrix asks the same question.
func runIsStale(root string, run verify.Result) (bool, string) {
	current, reason := verify.IsCurrent(root, run)
	return !current, reason
}

func artifactPath(root, changeID, name string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, name)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolvePlan reports NEEDS_PLAN when the verification plan is absent or holds
// no item. A malformed plan is an error.
func resolvePlan(root, changeID string) (Result, bool, error) {
	planPath := verification.Path(root, changeID)
	if _, err := os.Stat(planPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return needsPlan(changeID, "no verification plan exists",
				fmt.Sprintf("shipproof start %s --intent <path> --force", changeID)), true, nil
		}
		return Result{}, false, fmt.Errorf("inspect verification plan: %w", err)
	}

	plan, err := verification.Load(planPath)
	if err != nil {
		return Result{}, false, fmt.Errorf("load verification plan: %w", err)
	}
	if len(plan.Requirements)+len(plan.Invariants) == 0 {
		return needsPlan(changeID, "verification.json carries no requirement",
			fmt.Sprintf("shipproof prove %s", changeID)), true, nil
	}
	return Result{}, false, nil
}

// needsPlan names the command that works for the case at hand. An absent plan
// needs verification init. An empty plan needs the author to fill it, so the
// command names verification check, which reports what the plan lacks.
func needsPlan(changeID, blocker, nextCommand string) Result {
	return Result{
		ChangeID:        changeID,
		Phase:           NeedsPlan,
		Blocker:         blocker,
		NextCommand:     nextCommand,
		NextInstruction: "plan-proof",
	}
}

// instructionFor names the instruction file of Section 8.2 that covers one
// phase. Every phase maps onto one of the three files.
func instructionFor(name Phase) string {
	switch name {
	case NoChange, IntentStale:
		return "capture-intent"
	case NeedsPlan, NeedsRun, RunStale:
		return "plan-proof"
	default:
		return "read-evidence"
	}
}
