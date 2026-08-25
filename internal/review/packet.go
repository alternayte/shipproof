package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alternayte/shipproof/internal/schema"
)

type ReviewPacket struct {
	SchemaVersion string `json:"schema_version"`
	ChangeID      string `json:"change_id"`
	// UnexplainedChange leads the packet. A reviewer reads the code no proof
	// ran before the reviewer reads what the proofs did cover.
	UnexplainedChange *schema.UnexplainedEvidence `json:"unexplained_change,omitempty"`
	Intent            IntentSection               `json:"intent"`
	GitSummary        GitSummarySection           `json:"git_summary"`
	AlreadyProven     []ProvenCheck               `json:"already_proven"`
	HumanAttention    []AttentionCheck            `json:"human_attention"`
	Unknown           []UnknownCheck              `json:"unknown"`
}

type IntentSection struct {
	SnapshotHash     string `json:"snapshot_hash"`
	RequirementCount int    `json:"requirement_count"`
}

type GitSummarySection struct {
	Commits           []schema.ImplementationCommit `json:"commits"`
	ChangedFilesCount int                           `json:"changed_files_count"`
	Additions         int                           `json:"additions"`
	Deletions         int                           `json:"deletions"`
}

type ProvenCheck struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	Provenance string `json:"provenance"`
}

type AttentionCheck struct {
	CheckID              string   `json:"check_id"`
	Status               string   `json:"status"`
	Provenance           string   `json:"provenance"`
	Source               string   `json:"source"`
	Reason               string   `json:"reason"`
	RelevantRequirements []string `json:"relevant_requirements"`
}

type UnknownCheck struct {
	CheckID         string `json:"check_id"`
	Status          string `json:"status"`
	Provenance      string `json:"provenance"`
	Source          string `json:"source"`
	WhatIsUncertain string `json:"what_is_uncertain"`
}

func Prepare(root, changeID string) (ReviewPacket, error) {
	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")

	data, err := os.ReadFile(packPath)
	if err != nil {
		return ReviewPacket{}, fmt.Errorf("read evidence pack: %w", err)
	}

	var pack schema.EvidencePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return ReviewPacket{}, fmt.Errorf("parse evidence pack: %w", err)
	}

	packet := ReviewPacket{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
	}

	packet.UnexplainedChange = pack.UnexplainedChange
	packet.Intent = buildIntentSection(pack)
	packet.GitSummary = buildGitSummary(pack)
	packet.AlreadyProven = buildAlreadyProven(pack)
	packet.HumanAttention = buildHumanAttention(pack)
	packet.Unknown = buildUnknown(pack)

	return packet, nil
}

func WritePacket(root string, packet ReviewPacket) error {
	dir := filepath.Join(root, ".shipproof", "changes", packet.ChangeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create change directory: %w", err)
	}

	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return fmt.Errorf("encode review packet: %w", err)
	}
	data = append(data, '\n')

	packetPath := filepath.Join(dir, "review-packet.json")
	if err := os.WriteFile(packetPath, data, 0o644); err != nil {
		return fmt.Errorf("write review packet: %w", err)
	}

	return nil
}

func buildIntentSection(pack schema.EvidencePack) IntentSection {
	return IntentSection{
		SnapshotHash:     pack.Intent.SnapshotHash,
		RequirementCount: len(pack.Intent.Requirements),
	}
}

func buildGitSummary(pack schema.EvidencePack) GitSummarySection {
	return GitSummarySection{
		Commits:           pack.Implementation.Commits,
		ChangedFilesCount: len(pack.Implementation.ChangedFiles),
		Additions:         pack.Implementation.Additions,
		Deletions:         pack.Implementation.Deletions,
	}
}

func buildAlreadyProven(pack schema.EvidencePack) []ProvenCheck {
	var proven []ProvenCheck
	for _, check := range pack.Verification.Checks {
		if check.Provenance == schema.ProvenanceObserved && check.Status == "pass" {
			proven = append(proven, ProvenCheck{
				ID:         check.ID,
				Status:     check.Status,
				Source:     check.Source,
				Provenance: string(check.Provenance),
			})
		}
	}
	return proven
}

func buildHumanAttention(pack schema.EvidencePack) []AttentionCheck {
	var attention []AttentionCheck
	for _, check := range pack.Verification.Checks {
		if isHumanAttentionRequired(check) {
			attention = append(attention, AttentionCheck{
				CheckID:              check.ID,
				Status:               check.Status,
				Provenance:           string(check.Provenance),
				Source:               check.Source,
				Reason:               reasonForAttention(check),
				RelevantRequirements: relevantRequirementIDs(pack, check),
			})
		}
	}
	return attention
}

func buildUnknown(pack schema.EvidencePack) []UnknownCheck {
	var unknown []UnknownCheck
	for _, check := range pack.Verification.Checks {
		if isUnknown(check) {
			unknown = append(unknown, UnknownCheck{
				CheckID:         check.ID,
				Status:          check.Status,
				Provenance:      string(check.Provenance),
				Source:          check.Source,
				WhatIsUncertain: uncertaintyForCheck(check),
			})
		}
	}
	return unknown
}

func isHumanAttentionRequired(check schema.Check) bool {
	if isObservedPass(check) {
		return false
	}
	if check.Status == "unknown" {
		return false
	}
	return true
}

func isObservedPass(check schema.Check) bool {
	return check.Provenance == schema.ProvenanceObserved && check.Status == "pass"
}

func isUnknown(check schema.Check) bool {
	return check.Status == "unknown" || check.Status == "skip"
}

func reasonForAttention(check schema.Check) string {
	if check.Status == "fail" {
		return fmt.Sprintf("check %q returned status %q", check.ID, check.Status)
	}
	if check.Provenance == schema.ProvenanceInferred {
		return fmt.Sprintf("check %q has %q provenance; the status is inferred and requires human interpretation", check.ID, check.Provenance)
	}
	if check.Provenance == schema.ProvenanceDerived {
		return fmt.Sprintf("check %q has %q provenance; the status was computed and requires verification", check.ID, check.Provenance)
	}
	if check.Provenance == schema.ProvenanceHuman {
		return fmt.Sprintf("check %q was provided by a human; the status requires review", check.ID)
	}
	return fmt.Sprintf("check %q requires attention (status: %q, provenance: %q)", check.ID, check.Status, check.Provenance)
}

func uncertaintyForCheck(check schema.Check) string {
	if check.Status == "unknown" {
		return fmt.Sprintf("check %q has unknown status; the outcome was not determined", check.ID)
	}
	if check.Status == "skip" {
		return fmt.Sprintf("check %q was skipped; the behavior was not verified", check.ID)
	}
	return fmt.Sprintf("check %q status is %q; the verification outcome is uncertain", check.ID, check.Status)
}

func relevantRequirementIDs(pack schema.EvidencePack, check schema.Check) []string {
	var reqIDs []string
	for _, req := range pack.Intent.Requirements {
		for _, ref := range req.VerificationRefs {
			if containsCheckID(ref, check.ID) || containsCheckID(ref, check.Source) {
				reqIDs = append(reqIDs, req.ID)
				break
			}
		}
	}
	return reqIDs
}

func containsCheckID(ref, checkID string) bool {
	return len(checkID) > 0 && len(ref) > 0 && (ref == checkID || stringContains(ref, checkID))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
