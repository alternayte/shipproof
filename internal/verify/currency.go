package verify

import (
	"fmt"

	"github.com/alternayte/shipproof/internal/git"
)

// StateDirectory is the ShipProof state directory. Tree checks exclude it,
// because ShipProof writes its own artifacts there as a normal part of the
// flow. A check that counted those writes would mark every run stale the
// moment the evidence pack landed.
const StateDirectory = ".shipproof"

// TreeState reports the Git revision and the cleanliness of root. A directory
// that is not a Git repository yields an empty revision and a nil tree state.
// The caller must treat both as unknown, never as clean.
func TreeState(root string) (string, *bool) {
	head, err := git.HeadRevision(root)
	if err != nil {
		return "", nil
	}
	dirty, err := git.DirtyOutside(root, StateDirectory)
	if err != nil {
		return head, nil
	}
	clean := !dirty
	return head, &clean
}

// IsCurrent reports whether a recorded run still describes the working tree,
// and states the reason when it does not.
//
// A run with no recorded revision cannot be judged, so it counts as current.
// An unjudgeable run must not become a false alarm.
func IsCurrent(root string, run Result) (bool, string) {
	if run.HeadRev == "" {
		return true, ""
	}

	head, err := git.HeadRevision(root)
	if err != nil {
		return true, ""
	}
	if head != run.HeadRev {
		return false, fmt.Sprintf("the run verified revision %s; HEAD is %s", ShortRevision(run.HeadRev), ShortRevision(head))
	}
	if run.TreeClean != nil && !*run.TreeClean {
		return false, "the run verified a dirty working tree"
	}

	dirty, err := git.DirtyOutside(root, StateDirectory)
	if err != nil {
		return true, ""
	}
	if dirty {
		return false, "the working tree changed after the run"
	}
	return true, ""
}

// ShortRevision abbreviates a revision for a human-facing message.
func ShortRevision(revision string) string {
	if len(revision) > 7 {
		return revision[:7]
	}
	return revision
}
