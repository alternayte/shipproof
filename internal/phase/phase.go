// Package phase derives the current delivery phase of a ShipProof change from
// the artifacts on disk. The phase is a pure function of those artifacts.
// ShipProof stores no cursor, so an agent that acts out of band cannot
// desynchronize the answer.
package phase

import (
	"errors"
	"fmt"
	"os"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/verification"
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

	return Result{ChangeID: changeID, Phase: NeedsRun}, nil
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
