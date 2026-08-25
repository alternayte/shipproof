package proofs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alternayte/shipproof/internal/verification"
)

func planWithOnePassAndOneFail() verification.Plan {
	return verification.Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []verification.Item{
			{
				ID: "SP-021-R1",
				Proof: []verification.Proof{
					{Type: "command", Target: "a", Command: "true"},
					{Type: "command", Target: "b", Command: "exit 3"},
				},
			},
		},
		Invariants: []verification.Item{
			{
				ID: "SP-021-I1",
				Proof: []verification.Proof{
					{Type: "human", Target: "c", Human: true, Rationale: "A person reads it."},
				},
			},
		},
	}
}

func TestRunRecordsOneResultPerProof(t *testing.T) {
	t.Parallel()

	set, err := Run(t.TempDir(), planWithOnePassAndOneFail(), "", nil, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(set.Results) != 3 {
		t.Fatalf("Results = %d, want 3", len(set.Results))
	}
	if err := set.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRunRecordsThePassingProof(t *testing.T) {
	t.Parallel()

	set, err := Run(t.TempDir(), planWithOnePassAndOneFail(), "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	first := set.Results[0]
	if first.RequirementID != "SP-021-R1" || first.ProofIndex != 0 {
		t.Fatalf("result 1 = %+v", first)
	}
	if first.Status != Pass || first.ExitCode != 0 {
		t.Fatalf("result 1 status = %q, exit = %d; want pass, 0", first.Status, first.ExitCode)
	}
	if first.Command != "true" {
		t.Fatalf("result 1 command = %q", first.Command)
	}
}

func TestRunRecordsTheFailingProofWithItsExitCode(t *testing.T) {
	t.Parallel()

	set, err := Run(t.TempDir(), planWithOnePassAndOneFail(), "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	second := set.Results[1]
	if second.Status != Fail {
		t.Fatalf("result 2 status = %q, want %q", second.Status, Fail)
	}
	if second.ExitCode != 3 {
		t.Fatalf("result 2 exit = %d, want 3", second.ExitCode)
	}
	if second.ProofIndex != 1 {
		t.Fatalf("result 2 proof_index = %d, want 1", second.ProofIndex)
	}
}

func TestRunRunsNoCommandForAHumanProof(t *testing.T) {
	t.Parallel()

	set, err := Run(t.TempDir(), planWithOnePassAndOneFail(), "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	third := set.Results[2]
	if third.Status != Human {
		t.Fatalf("result 3 status = %q, want %q", third.Status, Human)
	}
	if third.Command != "" {
		t.Fatalf("result 3 command = %q, want an empty command", third.Command)
	}
	if third.RequirementID != "SP-021-I1" {
		t.Fatalf("result 3 requirement = %q", third.RequirementID)
	}
}

func TestRunCarriesTheRecordedRevisionAndTreeState(t *testing.T) {
	t.Parallel()

	clean := true
	set, err := Run(t.TempDir(), planWithOnePassAndOneFail(), "1cceb33", &clean, nil)
	if err != nil {
		t.Fatal(err)
	}
	if set.HeadRev != "1cceb33" {
		t.Fatalf("HeadRev = %q", set.HeadRev)
	}
	if set.TreeClean == nil || !*set.TreeClean {
		t.Fatalf("TreeClean = %v, want true", set.TreeClean)
	}
	if set.Timestamp == "" {
		t.Fatal("Timestamp is empty")
	}
	if set.ChangeID != "SP-021" {
		t.Fatalf("ChangeID = %q", set.ChangeID)
	}
}

func TestRunOnAnEmptyPlanRecordsNoResult(t *testing.T) {
	t.Parallel()

	set, err := Run(t.TempDir(), verification.Plan{SchemaVersion: "0.1", ChangeID: "SP-021"}, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Results) != 0 {
		t.Fatalf("Results = %d, want 0", len(set.Results))
	}
	if set.Results == nil {
		t.Fatal("Results is nil; it must encode as an empty array")
	}
}

func TestRunUsesTheRepositoryRootAsTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := verification.Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []verification.Item{{
			ID:    "SP-021-R1",
			Proof: []verification.Proof{{Type: "command", Target: "marker", Command: "test -f marker"}},
		}},
	}

	set, err := Run(root, plan, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if set.Results[0].Status != Fail {
		t.Fatalf("status = %q, want %q before the marker exists", set.Results[0].Status, Fail)
	}

	if err := writeMarker(root); err != nil {
		t.Fatal(err)
	}
	set, err = Run(root, plan, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if set.Results[0].Status != Pass {
		t.Fatalf("status = %q, want %q after the marker exists", set.Results[0].Status, Pass)
	}
}

func writeMarker(root string) error {
	return os.WriteFile(filepath.Join(root, "marker"), []byte("x\n"), 0o644)
}

func TestRunWritesOneProfilePerProof(t *testing.T) {
	root := t.TempDir()
	plan := verification.Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-100",
		Requirements: []verification.Item{
			{ID: "SP-100-R1", Proof: []verification.Proof{
				{Type: "test", Target: "internal/a", Command: "true"},
				{Type: "human", Target: "docs/a.md", Human: true, Rationale: "a person reads it"},
			}},
		},
	}

	dir := filepath.Join(root, ".shipproof", "runs", "SP-100", "coverage")
	set, err := Run(root, plan, "abc123", nil, &Coverage{
		Command: "printf 'mode: set\\nm/internal/a/a.go:1.1,2.2 1 1\\n' > {{profile}} && test -n '{{target}}'",
		Dir:     dir,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(set.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(set.Results))
	}
	if set.Results[0].Status != Pass {
		t.Errorf("status = %q, want pass", set.Results[0].Status)
	}

	written := filepath.Join(dir, "SP-100-R1-0.out")
	if _, err := os.Stat(written); err != nil {
		t.Errorf("profile not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SP-100-R1-1.out")); err == nil {
		t.Error("a human proof produced a profile")
	}
}

func TestRunKeepsTheResultWhenCoverageFails(t *testing.T) {
	root := t.TempDir()
	plan := verification.Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-101",
		Requirements: []verification.Item{
			{ID: "SP-101-R1", Proof: []verification.Proof{
				{Type: "test", Target: "internal/a", Command: "true"},
			}},
		},
	}

	set, err := Run(root, plan, "abc123", nil, &Coverage{
		Command: "exit 3",
		Dir:     filepath.Join(root, "coverage"),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if set.Results[0].Status != Pass || set.Results[0].ExitCode != 0 {
		t.Errorf("a failed coverage command changed the result: %#v", set.Results[0])
	}
}

func TestRunWithoutCoverageWritesNoProfile(t *testing.T) {
	root := t.TempDir()
	plan := verification.Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-102",
		Requirements: []verification.Item{
			{ID: "SP-102-R1", Proof: []verification.Proof{{Type: "test", Target: "internal/a", Command: "true"}}},
		},
	}
	set, err := Run(root, plan, "abc123", nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(set.Results) != 1 || set.Results[0].Status != Pass {
		t.Errorf("results = %#v", set.Results)
	}
}
