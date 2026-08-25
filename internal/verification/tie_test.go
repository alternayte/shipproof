package verification

import (
	"testing"

	"github.com/alternayte/shipproof/internal/requirements"
)

func tieSet(ids ...string) requirements.Set {
	set := requirements.Set{
		SchemaVersion: requirements.SchemaVersion,
		ChangeID:      "SP-028",
		Adopter:       requirements.AdopterNative,
	}
	for _, id := range ids {
		set.Requirements = append(set.Requirements, requirements.Requirement{
			ID: id, Statement: "A statement.", Provenance: requirements.Observed,
		})
	}
	return set
}

func tiePlan(ids ...string) Plan {
	plan := New("SP-028")
	for _, id := range ids {
		plan.Requirements = append(plan.Requirements, Item{
			ID: id, Proof: []Proof{{Type: "unit", Target: "x_test.go"}},
		})
	}
	return plan
}

func TestTieCheckPassesWhenTheSetsMatch(t *testing.T) {
	t.Parallel()

	blockers := TieCheck(tieSet("SP-028-R1", "SP-028-R2"), tiePlan("SP-028-R1", "SP-028-R2"))
	if len(blockers) != 0 {
		t.Fatalf("blockers = %+v, want none", blockers)
	}
}

func TestTieCheckBlocksARequirementWithNoPlanEntry(t *testing.T) {
	t.Parallel()

	blockers := TieCheck(tieSet("SP-028-R1", "SP-028-R2"), tiePlan("SP-028-R1"))
	if len(blockers) != 1 {
		t.Fatalf("blockers = %+v, want 1", blockers)
	}
	if blockers[0].Kind != BlockerUnplanned {
		t.Fatalf("Kind = %q, want %q", blockers[0].Kind, BlockerUnplanned)
	}
	if blockers[0].RequirementID != "SP-028-R2" {
		t.Fatalf("RequirementID = %q, want SP-028-R2", blockers[0].RequirementID)
	}
}

func TestTieCheckBlocksAPlanEntryWithNoRequirement(t *testing.T) {
	t.Parallel()

	blockers := TieCheck(tieSet("SP-028-R1"), tiePlan("SP-028-R1", "SP-028-R9"))
	if len(blockers) != 1 {
		t.Fatalf("blockers = %+v, want 1", blockers)
	}
	if blockers[0].Kind != BlockerUntied {
		t.Fatalf("Kind = %q, want %q", blockers[0].Kind, BlockerUntied)
	}
	if blockers[0].RequirementID != "SP-028-R9" {
		t.Fatalf("RequirementID = %q, want SP-028-R9", blockers[0].RequirementID)
	}
}

func TestTieCheckReportsBothDirectionsAtOnce(t *testing.T) {
	t.Parallel()

	blockers := TieCheck(tieSet("SP-028-R1", "SP-028-R2"), tiePlan("SP-028-R1", "SP-028-R9"))
	if len(blockers) != 2 {
		t.Fatalf("blockers = %+v, want 2", blockers)
	}
	if blockers[0].Kind != BlockerUnplanned || blockers[0].RequirementID != "SP-028-R2" {
		t.Fatalf("blocker 1 = %+v", blockers[0])
	}
	if blockers[1].Kind != BlockerUntied || blockers[1].RequirementID != "SP-028-R9" {
		t.Fatalf("blocker 2 = %+v", blockers[1])
	}
}

func TestTieCheckIgnoresAnInvariant(t *testing.T) {
	t.Parallel()

	plan := tiePlan("SP-028-R1")
	plan.Invariants = append(plan.Invariants, Item{
		ID: "SP-028-I1", Proof: []Proof{{Type: "unit", Target: "x_test.go"}},
	})
	blockers := TieCheck(tieSet("SP-028-R1"), plan)
	if len(blockers) != 0 {
		t.Fatalf("blockers = %+v, want none", blockers)
	}
}

func TestTieCheckIsStableAcrossCalls(t *testing.T) {
	t.Parallel()

	set := tieSet("SP-028-R3", "SP-028-R1", "SP-028-R2")
	plan := tiePlan("SP-028-R9", "SP-028-R8")
	first := TieCheck(set, plan)
	second := TieCheck(set, plan)
	if len(first) != len(second) {
		t.Fatalf("length changed: %d then %d", len(first), len(second))
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("order changed at %d: %+v then %+v", index, first[index], second[index])
		}
	}
	wantOrder := []string{"SP-028-R1", "SP-028-R2", "SP-028-R3", "SP-028-R8", "SP-028-R9"}
	for index, id := range wantOrder {
		if first[index].RequirementID != id {
			t.Fatalf("blocker %d = %q, want %q", index, first[index].RequirementID, id)
		}
	}
}
