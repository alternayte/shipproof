package verdict

import (
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/phase"
	"github.com/alternayte/shipproof/internal/schema"
)

func provenMatrix() coverage.Matrix {
	return coverage.Matrix{
		ChangeID:   "SP-030",
		RunCurrent: true,
		Rows: []coverage.Row{
			{RequirementID: "R1", State: coverage.Proven, Provenance: coverage.Observed},
			{RequirementID: "R2", State: coverage.Accepted, Provenance: coverage.Human},
		},
	}
}

func zero() *int { count := 0; return &count }

func TestProvenNeedsEveryRowAndAKnownZeroCount(t *testing.T) {
	block := Decide(Input{
		ChangeID:    "SP-030",
		Phase:       phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
		Matrix:      provenMatrix(),
		HasMatrix:   true,
		Unexplained: zero(),
	})
	if block.Verdict != Proven {
		t.Fatalf("verdict = %q, want %q; reason %q", block.Verdict, Proven, block.Reason)
	}
}

func TestUnknownCountBlocksProven(t *testing.T) {
	block := Decide(Input{
		ChangeID:  "SP-030",
		Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
		Matrix:    provenMatrix(),
		HasMatrix: true,
		// Unexplained stays nil. No tool measured the count.
	})
	if block.Verdict != NotProven {
		t.Fatalf("verdict = %q, want %q", block.Verdict, NotProven)
	}
	if !strings.Contains(block.Reason, "not known") {
		t.Fatalf("reason = %q, want it to name the unknown count", block.Reason)
	}
}

func TestAFailedProofFails(t *testing.T) {
	matrix := provenMatrix()
	matrix.Rows[1].State = coverage.Failed
	block := Decide(Input{
		ChangeID:    "SP-030",
		Phase:       phase.Result{ChangeID: "SP-030", Phase: phase.NeedsEvidence},
		Matrix:      matrix,
		HasMatrix:   true,
		Unexplained: zero(),
	})
	if block.Verdict != Failed {
		t.Fatalf("verdict = %q, want %q", block.Verdict, Failed)
	}
}

func TestAFailedGateFails(t *testing.T) {
	block := Decide(Input{
		ChangeID:    "SP-030",
		Phase:       phase.Result{ChangeID: "SP-030", Phase: phase.RunFailed, NextCommand: "shipproof prove SP-030"},
		Matrix:      provenMatrix(),
		HasMatrix:   true,
		Unexplained: zero(),
	})
	if block.Verdict != Failed {
		t.Fatalf("verdict = %q, want %q", block.Verdict, Failed)
	}
}

func TestUnexplainedLinesBlockProven(t *testing.T) {
	count := 14
	block := Decide(Input{
		ChangeID:    "SP-030",
		Phase:       phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
		Matrix:      provenMatrix(),
		HasMatrix:   true,
		Unexplained: &count,
	})
	if block.Verdict != NotProven {
		t.Fatalf("verdict = %q, want %q", block.Verdict, NotProven)
	}
	if !strings.Contains(block.Reason, "14 changed lines match no requirement") {
		t.Fatalf("reason = %q", block.Reason)
	}
}

func TestNoMatrixIsNotProven(t *testing.T) {
	block := Decide(Input{
		ChangeID: "SP-030",
		Phase:    phase.Result{ChangeID: "SP-030", Phase: phase.NeedsPlan, NextCommand: "shipproof prove SP-030"},
	})
	if block.Verdict != NotProven {
		t.Fatalf("verdict = %q, want %q", block.Verdict, NotProven)
	}
}

func TestTheBlockHoldsExactlyThreeLines(t *testing.T) {
	inputs := []Input{
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.ReadyForHuman}, Matrix: provenMatrix(), HasMatrix: true, Unexplained: zero()},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.NeedsPlan, NextCommand: "shipproof prove SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.RunFailed, NextCommand: "shipproof prove SP-030"}},
	}
	for _, input := range inputs {
		text := Decide(input).String()
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("block %q holds %d lines, want 3", text, len(lines))
		}
		if !strings.HasPrefix(lines[0], "VERDICT: ") {
			t.Fatalf("line 1 = %q", lines[0])
		}
		if strings.TrimSpace(strings.TrimPrefix(lines[0], "VERDICT: ")) != string(Decide(input).Verdict) {
			t.Fatalf("line 1 = %q, want the verdict word alone", lines[0])
		}
		if !strings.HasPrefix(lines[2], "NEXT: ") {
			t.Fatalf("line 3 = %q", lines[2])
		}
		if strings.TrimSpace(lines[1]) == "" || strings.TrimSpace(lines[2]) == "" {
			t.Fatalf("block %q holds an empty line", text)
		}
	}
}

func TestTheBlockNamesARunnableCommand(t *testing.T) {
	block := Decide(Input{
		ChangeID: "SP-030",
		Phase:    phase.Result{ChangeID: "SP-030", Phase: phase.NeedsPlan, NextCommand: "shipproof prove SP-030"},
	})
	if !strings.Contains(block.Next, "shipproof ") {
		t.Fatalf("next = %q, want a shipproof command", block.Next)
	}
}

// TestNoJargonWordAppears is the proof for row U2 of the definition of done.
func TestNoJargonWordAppears(t *testing.T) {
	count := 14
	inputs := []Input{
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.ReadyForHuman}, Matrix: provenMatrix(), HasMatrix: true, Unexplained: zero()},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.ReadyForHuman}, Matrix: provenMatrix(), HasMatrix: true},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.ReadyForHuman}, Matrix: provenMatrix(), HasMatrix: true, Unexplained: &count},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.NeedsPlan, NextCommand: "shipproof prove SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.NeedsRun, NextCommand: "shipproof prove SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.RunStale, NextCommand: "shipproof prove SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.RunFailed, NextCommand: "shipproof prove SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.NeedsEvidence, NextCommand: "shipproof pack SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.IntentStale, NextCommand: "shipproof start SP-030"}},
		{ChangeID: "SP-030", Phase: phase.Result{Phase: phase.NoChange, NextCommand: "shipproof start SP-030"}},
	}
	for _, input := range inputs {
		text := strings.ToLower(Decide(input).String())
		for _, word := range JargonWords {
			if strings.Contains(text, word) {
				t.Fatalf("block %q holds the jargon word %q", text, word)
			}
		}
	}
}

func TestEveryPhaseProducesANextCommand(t *testing.T) {
	all := []phase.Phase{phase.NoChange, phase.IntentStale, phase.NeedsPlan,
		phase.NeedsRun, phase.RunStale, phase.RunFailed, phase.NeedsEvidence,
		phase.ReadyForHuman}
	for _, name := range all {
		block := Decide(Input{ChangeID: "SP-030", Phase: phase.Result{ChangeID: "SP-030", Phase: name}})
		if !strings.Contains(block.Next, "shipproof ") {
			t.Fatalf("phase %s produced next %q", name, block.Next)
		}
	}
}

// TestAClaimedCheckNeverProvesARequirement is the proof for row H2 of the
// definition of done.
func TestAClaimedCheckNeverProvesARequirement(t *testing.T) {
	matrix := coverage.Matrix{
		ChangeID:   "SP-030",
		RunCurrent: true,
		Rows: []coverage.Row{
			{RequirementID: "R1", State: coverage.Proven, Provenance: coverage.Observed},
			{RequirementID: "R2", State: coverage.Unproven, Provenance: coverage.Unknown},
		},
	}
	for _, label := range []schema.ProvenanceKind{schema.ProvenanceDerived, schema.ProvenanceInferred, "guessed"} {
		block := Decide(Input{
			ChangeID:  "SP-030",
			Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
			Matrix:    matrix,
			HasMatrix: true,
			Checks: []schema.Check{
				{ID: "R2", Status: "pass", Source: "agent", Provenance: label},
			},
			Unexplained: zero(),
		})
		if block.Verdict != NotProven {
			t.Fatalf("a %q check produced %q, want %q", label, block.Verdict, NotProven)
		}
		if !strings.Contains(block.Reason, "1 of 2 requirements have no proof") {
			t.Fatalf("reason = %q", block.Reason)
		}
	}
}

// TestAFailedProofCheckFails holds requirement R6 of SP-031. Section 5 says
// that a proof which ran and failed produces FAILED. A parsed test report is
// such a proof.
func TestAFailedProofCheckFails(t *testing.T) {
	for _, id := range []string{"junit:TestRetry", "sarif:rule-7", "verification:run"} {
		block := Decide(Input{
			ChangeID:  "SP-030",
			Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
			Matrix:    provenMatrix(),
			HasMatrix: true,
			Checks: []schema.Check{
				{ID: id, Status: "fail", Source: "junit", Provenance: schema.ProvenanceObserved},
			},
			Unexplained: zero(),
		})
		if block.Verdict != Failed {
			t.Fatalf("a failed %s check produced %q, want %q", id, block.Verdict, Failed)
		}
	}
}

// TestAFailedInformationalCheckDoesNotFail holds requirement R4 of SP-031.
// Section 5 reserves FAILED for a failed proof and a failed gate. A check that
// reports the state of the pack is neither.
func TestAFailedInformationalCheckDoesNotFail(t *testing.T) {
	for _, id := range []string{"intent:staleness", "coverage:R2", "agent:review:claude"} {
		block := Decide(Input{
			ChangeID:  "SP-030",
			Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
			Matrix:    provenMatrix(),
			HasMatrix: true,
			Checks: []schema.Check{
				{ID: id, Status: "fail", Source: "shipproof", Provenance: schema.ProvenanceObserved},
			},
			Unexplained: zero(),
		})
		if block.Verdict == Failed {
			t.Fatalf("a failed %s check produced %q, and Section 5 reserves that word", id, Failed)
		}
	}
}

// TestAStaleIntentIsNotProven holds requirement R5 of SP-031.
func TestAStaleIntentIsNotProven(t *testing.T) {
	block := Decide(Input{
		ChangeID:  "SP-030",
		Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.IntentStale, NextCommand: "shipproof start SP-030"},
		Matrix:    provenMatrix(),
		HasMatrix: true,
		Checks: []schema.Check{
			{ID: "intent:staleness", Status: "fail", Source: "shipproof", Provenance: schema.ProvenanceObserved},
		},
		Unexplained: zero(),
	})
	if block.Verdict != NotProven {
		t.Fatalf("a stale intent produced %q, want %q", block.Verdict, NotProven)
	}
}

// TestAFailedClaimedCheckDoesNotFail holds the other half of the rule. An
// agent claim is not evidence in either direction.
func TestAFailedClaimedCheckDoesNotFail(t *testing.T) {
	block := Decide(Input{
		ChangeID:  "SP-030",
		Phase:     phase.Result{ChangeID: "SP-030", Phase: phase.ReadyForHuman},
		Matrix:    provenMatrix(),
		HasMatrix: true,
		Checks: []schema.Check{
			{ID: "review", Status: "fail", Source: "agent", Provenance: schema.ProvenanceInferred},
		},
		Unexplained: zero(),
	})
	if block.Verdict != Proven {
		t.Fatalf("verdict = %q, want %q; reason %q", block.Verdict, Proven, block.Reason)
	}
}
