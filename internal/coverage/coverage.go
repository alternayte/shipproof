// Package coverage derives the requirement coverage matrix. The matrix is
// derived on demand and no agent writes it. A stored matrix would be an
// assertion; a derived matrix is a reading of the artifacts on disk.
//
// No state and no provenance in this package reads inferred. A row that
// nothing proved reads unproven with unknown provenance, and it says so.
package coverage

import (
	"fmt"

	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

// State names what the artifacts say about one requirement.
type State string

const (
	// Proven holds when at least one automated proof exists and all of them
	// passed at the current revision.
	Proven State = "proven"
	// Failed holds when any automated proof failed.
	Failed State = "failed"
	// Accepted holds when every proof is human and a person accepted each one.
	Accepted State = "accepted"
	// AwaitingHuman holds when a human proof waits for acceptance.
	AwaitingHuman State = "awaiting-human"
	// Unproven holds when no proof exists, or no proof ran at this revision.
	Unproven State = "unproven"
)

// Provenance names how ShipProof learned the state.
type Provenance string

const (
	Observed Provenance = "observed"
	Human    Provenance = "human"
	Unknown  Provenance = "unknown"
)

// Row is the state of one requirement.
type Row struct {
	RequirementID string     `json:"requirement_id"`
	Statement     string     `json:"statement,omitempty"`
	State         State      `json:"state"`
	Provenance    Provenance `json:"provenance"`
	// Detail states the reason in one sentence. It is review material.
	Detail string `json:"detail,omitempty"`
}

// Matrix holds one row per requirement in the sidecar, in sidecar order.
type Matrix struct {
	ChangeID string `json:"change_id"`
	// RunCurrent reports whether the recorded proof results still describe the
	// working tree. A false value forces every automated row to unproven.
	RunCurrent bool  `json:"run_current"`
	Rows       []Row `json:"rows"`
}

// Build derives the matrix. results is nil when no proof artifact exists.
// resultsCurrent is false when the artifact does not describe the working tree.
func Build(set requirements.Set, plan verification.Plan, results *proofs.Set, resultsCurrent bool) Matrix {
	usable := results != nil && resultsCurrent
	recorded := map[string]proofs.Result{}
	if usable {
		for _, result := range results.Results {
			recorded[key(result.RequirementID, result.ProofIndex)] = result
		}
	}

	items := map[string]verification.Item{}
	for _, group := range [][]verification.Item{plan.Requirements, plan.Invariants} {
		for _, item := range group {
			items[item.ID] = item
		}
	}

	matrix := Matrix{ChangeID: set.ChangeID, RunCurrent: usable, Rows: []Row{}}
	for _, requirement := range set.Requirements {
		item, planned := items[requirement.ID]
		row := classify(requirement.ID, item, planned, recorded)
		row.Statement = requirement.Statement
		matrix.Rows = append(matrix.Rows, row)
	}
	return matrix
}

// classify applies the state table in order. The first state that holds is the
// answer.
func classify(requirementID string, item verification.Item, planned bool, recorded map[string]proofs.Result) Row {
	if !planned {
		return Row{RequirementID: requirementID, State: Unproven, Provenance: Unknown,
			Detail: "the verification plan holds no entry for this requirement"}
	}
	if len(item.Proof) == 0 {
		return Row{RequirementID: requirementID, State: Unproven, Provenance: Unknown,
			Detail: "the plan entry names no proof"}
	}

	automated, passed, failed := 0, 0, 0
	humanProofs, accepted := 0, 0
	for index, proof := range item.Proof {
		if proof.IsHuman() {
			humanProofs++
			if proof.Accepted() {
				accepted++
			}
			continue
		}
		if !proof.IsAutomated() {
			continue
		}
		automated++
		result, ran := recorded[key(requirementID, index)]
		if !ran {
			continue
		}
		switch result.Status {
		case proofs.Pass:
			passed++
		case proofs.Fail:
			failed++
		}
	}

	switch {
	case failed > 0:
		return Row{RequirementID: requirementID, State: Failed, Provenance: Observed,
			Detail: fmt.Sprintf("%d of %d automated proofs failed at this revision", failed, automated)}
	case automated > 0 && passed == automated:
		return Row{RequirementID: requirementID, State: Proven, Provenance: Observed,
			Detail: fmt.Sprintf("%d automated proofs passed at this revision", passed)}
	case automated == 0 && humanProofs > 0 && accepted == humanProofs:
		return Row{RequirementID: requirementID, State: Accepted, Provenance: Human,
			Detail: fmt.Sprintf("a person accepted %d human proofs", accepted)}
	case automated == 0 && humanProofs > 0:
		return Row{RequirementID: requirementID, State: AwaitingHuman, Provenance: Unknown,
			Detail: fmt.Sprintf("%d of %d human proofs carry no recorded acceptance", humanProofs-accepted, humanProofs)}
	case automated > 0:
		return Row{RequirementID: requirementID, State: Unproven, Provenance: Unknown,
			Detail: fmt.Sprintf("%d of %d automated proofs have no result at this revision", automated-passed-failed, automated)}
	default:
		return Row{RequirementID: requirementID, State: Unproven, Provenance: Unknown,
			Detail: "the plan entry names no usable proof"}
	}
}

func key(requirementID string, index int) string {
	return fmt.Sprintf("%s#%d", requirementID, index)
}
