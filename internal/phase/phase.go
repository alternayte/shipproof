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
	"github.com/alternayte/shipproof/internal/git"
	"github.com/alternayte/shipproof/internal/verification"
	"github.com/alternayte/shipproof/internal/verify"
)

type Phase string

const (
	NoChange          Phase = "NO_CHANGE"
	IntentStale       Phase = "INTENT_STALE"
	NeedsPlan         Phase = "NEEDS_PLAN"
	NeedsRun          Phase = "NEEDS_RUN"
	RunStale          Phase = "RUN_STALE"
	RunFailed         Phase = "RUN_FAILED"
	NeedsEvidence     Phase = "NEEDS_EVIDENCE"
	NeedsReviewPacket Phase = "NEEDS_REVIEW_PACKET"
	ReadyForHuman     Phase = "READY_FOR_HUMAN"
)

// Result names the phase, the blocker that holds it open, the exact next
// command, and the skill that handles it.
type Result struct {
	ChangeID    string `json:"change_id"`
	Phase       Phase  `json:"phase"`
	Blocker     string `json:"blocker,omitempty"`
	NextCommand string `json:"next_command,omitempty"`
	NextSkill   string `json:"next_skill,omitempty"`
}

// Resolve returns the first phase whose condition holds. A malformed artifact
// is an error, never a phase. Reporting a corrupt record as NO_CHANGE would
// hide the corruption.
func Resolve(root, changeID string) (Result, error) {
	recordPath := change.Path(root, changeID)
	if _, err := os.Stat(recordPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				ChangeID:    changeID,
				Phase:       NoChange,
				Blocker:     "no change record exists",
				NextCommand: fmt.Sprintf("shipproof change start %s --source <path>", changeID),
				NextSkill:   "prepare-change",
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
			ChangeID:    changeID,
			Phase:       IntentStale,
			Blocker:     fmt.Sprintf("source %s changed after the snapshot", record.SourcePath),
			NextCommand: fmt.Sprintf("shipproof change start %s --source %s", changeID, record.SourcePath),
			NextSkill:   "prepare-change",
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
			ChangeID:    changeID,
			Phase:       NeedsEvidence,
			Blocker:     "no evidence pack exists for the current run",
			NextCommand: fmt.Sprintf("shipproof evidence pack %s", changeID),
			NextSkill:   "produce-evidence",
		}, nil
	}

	if level >= 1 && !fileExists(artifactPath(root, changeID, "review-packet.json")) {
		return Result{
			ChangeID:    changeID,
			Phase:       NeedsReviewPacket,
			Blocker:     "no review packet exists",
			NextCommand: fmt.Sprintf("shipproof review prepare %s", changeID),
			NextSkill:   "prepare-human-review",
		}, nil
	}

	return Result{
		ChangeID:    changeID,
		Phase:       ReadyForHuman,
		NextCommand: fmt.Sprintf("shipproof report change %s", changeID),
		NextSkill:   "review-change",
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
				ChangeID:    changeID,
				Phase:       NeedsRun,
				Blocker:     "no run record exists",
				NextCommand: fmt.Sprintf("shipproof verification run %s", changeID),
				NextSkill:   "implement-change",
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
			ChangeID:    changeID,
			Phase:       RunStale,
			Blocker:     reason,
			NextCommand: fmt.Sprintf("shipproof verification run %s", changeID),
			NextSkill:   "implement-change",
		}, true, nil
	}

	if run.ExitCode != 0 {
		return Result{
			ChangeID:    changeID,
			Phase:       RunFailed,
			Blocker:     fmt.Sprintf("the newest run exited with code %d", run.ExitCode),
			NextCommand: fmt.Sprintf("shipproof verification run %s", changeID),
			NextSkill:   "implement-change",
		}, true, nil
	}

	return Result{}, false, nil
}

// runIsStale reports whether a recorded run still describes the working tree.
// A run with no recorded revision cannot be judged, so it is never stale. An
// unjudgeable run must not become a false alarm.
//
// The tree check excludes the ShipProof state directory. The evidence pack and
// the review packet land there as a normal part of the flow, and counting them
// would pin every change at RUN_STALE.
func runIsStale(root string, run verify.Result) (bool, string) {
	if run.HeadRev == "" {
		return false, ""
	}

	head, err := git.HeadRevision(root)
	if err != nil {
		return false, ""
	}
	if head != run.HeadRev {
		return true, fmt.Sprintf("the run verified revision %s; HEAD is %s", shortRev(run.HeadRev), shortRev(head))
	}
	if run.TreeClean != nil && !*run.TreeClean {
		return true, "the run verified a dirty working tree"
	}

	dirty, err := git.DirtyOutside(root, verify.StateDirectory)
	if err != nil {
		return false, ""
	}
	if dirty {
		return true, "the working tree changed after the run"
	}
	return false, ""
}

func shortRev(revision string) string {
	if len(revision) > 7 {
		return revision[:7]
	}
	return revision
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
			return needsPlan(changeID, "no verification plan exists"), true, nil
		}
		return Result{}, false, fmt.Errorf("inspect verification plan: %w", err)
	}

	plan, err := verification.Load(planPath)
	if err != nil {
		return Result{}, false, fmt.Errorf("load verification plan: %w", err)
	}
	if len(plan.Requirements)+len(plan.Invariants) == 0 {
		return needsPlan(changeID, "verification.json carries no requirement"), true, nil
	}
	return Result{}, false, nil
}

func needsPlan(changeID, blocker string) Result {
	return Result{
		ChangeID:    changeID,
		Phase:       NeedsPlan,
		Blocker:     blocker,
		NextCommand: fmt.Sprintf("shipproof verification init %s", changeID),
		NextSkill:   "plan-verification",
	}
}
