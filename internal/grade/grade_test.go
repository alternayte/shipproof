package grade

import (
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func TestOnlyThreeGradesExist(t *testing.T) {
	if len(All) != 3 {
		t.Fatalf("All holds %d grades, want 3", len(All))
	}
	want := map[Grade]bool{Observed: true, Stated: true, Claimed: true}
	for _, value := range All {
		if !want[value] {
			t.Fatalf("All holds the unexpected grade %q", value)
		}
	}
}

func TestTheMapCollapsesTheOldLabels(t *testing.T) {
	cases := []struct {
		label schema.ProvenanceKind
		want  Grade
	}{
		{schema.ProvenanceObserved, Observed},
		{schema.ProvenanceHuman, Stated},
		{schema.ProvenanceDerived, Claimed},
		{schema.ProvenanceInferred, Claimed},
	}
	for _, testCase := range cases {
		if got := FromProvenance(testCase.label); got != testCase.want {
			t.Fatalf("FromProvenance(%q) = %q, want %q", testCase.label, got, testCase.want)
		}
	}
}

// TestAnUnknownLabelIsClaimed holds the honesty rule. A label that ShipProof
// does not know is never evidence.
func TestAnUnknownLabelIsClaimed(t *testing.T) {
	for _, label := range []schema.ProvenanceKind{"", "guessed", "OBSERVED"} {
		if got := FromProvenance(label); got != Claimed {
			t.Fatalf("FromProvenance(%q) = %q, want %q", label, got, Claimed)
		}
	}
}

func TestOnlyAClaimedGradeProvesNothing(t *testing.T) {
	if !Observed.Proves() {
		t.Fatal("an observed grade must prove")
	}
	if !Stated.Proves() {
		t.Fatal("a stated grade must prove")
	}
	if Claimed.Proves() {
		t.Fatal("a claimed grade must never prove")
	}
}
