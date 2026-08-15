package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func makeEvidencePack(changeID string, checks []schema.Check) schema.EvidencePack {
	return schema.EvidencePack{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		Intent: schema.IntentEvidence{
			SnapshotHash: "abc123",
			Requirements: []schema.Requirement{
				{ID: "R1", VerificationRefs: []string{"fail:check", "inferred:check"}},
				{ID: "R2", VerificationRefs: []string{"pass:check"}},
			},
		},
		Verification: schema.VerificationEvidence{
			Checks: checks,
		},
		Implementation: schema.ImplementationEvidence{
			Commits: []schema.ImplementationCommit{
				{Hash: "abc", Author: "Alice", Timestamp: "2024-01-01T00:00:00Z", Subject: "fix stuff"},
			},
			ChangedFiles: []string{"file.go", "file_test.go"},
			Additions:    50,
			Deletions:    10,
			DiffStat:     "2 files changed",
		},
		Provenance: schema.PackProvenance{
			GeneratedAt:      "2024-01-01T00:00:00Z",
			ShipProofVersion: "0.1",
		},
	}
}

func writeEvidencePack(t *testing.T, root, changeID string, pack schema.EvidencePack) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		t.Fatalf("marshal pack: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "evidence-pack.json"), data, 0o644); err != nil {
		t.Fatalf("write pack: %v", err)
	}
}

func TestLoadEvidencePack(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if packet.ChangeID != "SP-006" {
		t.Errorf("expected change_id SP-006, got %s", packet.ChangeID)
	}
}

func TestPrepareMissingEvidencePack(t *testing.T) {
	root := t.TempDir()

	_, err := Prepare(root, "SP-006")
	if err == nil {
		t.Fatal("expected error for missing evidence pack, got nil")
	}
}

func TestBuildChangeIntentSection(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if packet.Intent.SnapshotHash != "abc123" {
		t.Errorf("expected snapshot_hash abc123, got %s", packet.Intent.SnapshotHash)
	}
	if packet.Intent.RequirementCount != 2 {
		t.Errorf("expected requirement_count 2, got %d", packet.Intent.RequirementCount)
	}
}

func TestBuildAlreadyProvenSection(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "fail:check", Status: "fail", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(packet.AlreadyProven) != 1 {
		t.Errorf("expected 1 proven check, got %d", len(packet.AlreadyProven))
	}
	if packet.AlreadyProven[0].ID != "pass:check" {
		t.Errorf("expected pass:check in proven, got %s", packet.AlreadyProven[0].ID)
	}
}

func TestBuildHumanAttentionSection(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "fail:check", Status: "fail", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "inferred:check", Status: "pass", Source: "analyzer", Provenance: schema.ProvenanceInferred},
		{ID: "derived:check", Status: "pass", Source: "compute", Provenance: schema.ProvenanceDerived},
		{ID: "human:check", Status: "pass", Source: "reviewer", Provenance: schema.ProvenanceHuman},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(packet.HumanAttention) != 4 {
		t.Errorf("expected 4 human attention checks, got %d", len(packet.HumanAttention))
	}
	for _, item := range packet.HumanAttention {
		if item.Reason == "" {
			t.Errorf("check %q has empty reason", item.CheckID)
		}
	}
}

func TestBuildUnknownSection(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "unknown:check", Status: "unknown", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "skip:check", Status: "skip", Source: "runner", Provenance: schema.ProvenanceHuman},
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(packet.Unknown) != 2 {
		t.Errorf("expected 2 unknown checks, got %d", len(packet.Unknown))
	}
	for _, item := range packet.Unknown {
		if item.WhatIsUncertain == "" {
			t.Errorf("check %q has empty what_is_uncertain", item.CheckID)
		}
	}
}

func TestExcludesObservedPassFromHumanAttention(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "fail:check", Status: "fail", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	for _, item := range packet.HumanAttention {
		if item.CheckID == "pass:check" {
			t.Errorf("observed-pass check should not be in human attention")
		}
	}
	for _, item := range packet.Unknown {
		if item.CheckID == "pass:check" {
			t.Errorf("observed-pass check should not be in unknown")
		}
	}
}

func TestAllObservedPassLeavesEmptyAttention(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "other:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(packet.HumanAttention) != 0 {
		t.Errorf("expected 0 human attention checks, got %d", len(packet.HumanAttention))
	}
	if len(packet.Unknown) != 0 {
		t.Errorf("expected 0 unknown checks, got %d", len(packet.Unknown))
	}
	if len(packet.AlreadyProven) != 2 {
		t.Errorf("expected 2 proven checks, got %d", len(packet.AlreadyProven))
	}
}

func TestIncludesGitSummary(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(packet.GitSummary.Commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(packet.GitSummary.Commits))
	}
	if packet.GitSummary.ChangedFilesCount != 2 {
		t.Errorf("expected 2 changed files, got %d", packet.GitSummary.ChangedFilesCount)
	}
	if packet.GitSummary.Additions != 50 {
		t.Errorf("expected 50 additions, got %d", packet.GitSummary.Additions)
	}
	if packet.GitSummary.Deletions != 10 {
		t.Errorf("expected 10 deletions, got %d", packet.GitSummary.Deletions)
	}
}

func TestWriteReviewPacket(t *testing.T) {
	root := t.TempDir()
	pack := makeEvidencePack("SP-006", []schema.Check{
		{ID: "pass:check", Status: "pass", Source: "runner", Provenance: schema.ProvenanceObserved},
		{ID: "fail:check", Status: "fail", Source: "runner", Provenance: schema.ProvenanceObserved},
	})
	writeEvidencePack(t, root, "SP-006", pack)

	packet, err := Prepare(root, "SP-006")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if err := WritePacket(root, packet); err != nil {
		t.Fatalf("WritePacket returned error: %v", err)
	}

	packetPath := filepath.Join(root, ".shipproof", "changes", "SP-006", "review-packet.json")
	data, err := os.ReadFile(packetPath)
	if err != nil {
		t.Fatalf("review-packet.json not written: %v", err)
	}
	var readBack ReviewPacket
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("parse review packet: %v", err)
	}
	if readBack.ChangeID != "SP-006" {
		t.Errorf("expected change_id SP-006, got %s", readBack.ChangeID)
	}
}

func TestRelevantRequirementIDs(t *testing.T) {
	pack := makeEvidencePack("SP-006", []schema.Check{})
	check := schema.Check{ID: "fail:check", Status: "fail", Provenance: schema.ProvenanceObserved}
	ids := relevantRequirementIDs(pack, check)
	if len(ids) != 1 {
		t.Fatalf("expected 1 requirement ID, got %d", len(ids))
	}
	if ids[0] != "R1" {
		t.Errorf("expected R1, got %s", ids[0])
	}
}
