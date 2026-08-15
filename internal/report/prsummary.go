package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/alternayte/shipproof/internal/review"
	"github.com/alternayte/shipproof/internal/schema"
)

func writePRSummary(w io.Writer, pack schema.EvidencePack, packet review.ReviewPacket) error {
	var sb strings.Builder

	sb.WriteString("# PR Evidence Summary — ")
	sb.WriteString(pack.ChangeID)
	sb.WriteString("\n\n")

	sb.WriteString("## What changed\n\n")
	writeChangedSection(&sb, pack)

	sb.WriteString("## Deterministic evidence\n\n")
	writeDeterministicSection(&sb, packet)

	sb.WriteString("## What remains uncertain\n\n")
	writeUncertainSection(&sb, packet)

	sb.WriteString("## What to inspect\n\n")
	writeInspectSection(&sb, packet)

	sb.WriteString("## Why each inspection matters\n\n")
	writeInspectReasonsSection(&sb, packet)

	_, err := io.WriteString(w, sb.String())
	return err
}

func writeChangedSection(sb *strings.Builder, pack schema.EvidencePack) {
	impl := pack.Implementation
	sb.WriteString(fmt.Sprintf("- **Commits:** %d\n", len(impl.Commits)))
	sb.WriteString(fmt.Sprintf("- **Files changed:** %d\n", len(impl.ChangedFiles)))
	sb.WriteString(fmt.Sprintf("- **Additions:** %d\n", impl.Additions))
	sb.WriteString(fmt.Sprintf("- **Deletions:** %d\n", impl.Deletions))

	if len(impl.Commits) > 0 {
		sb.WriteString("\n### Commits\n\n")
		for _, c := range impl.Commits {
			shortHash := c.Hash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			sb.WriteString(fmt.Sprintf("- `%s` %s\n", shortHash, c.Subject))
		}
	}

	if len(impl.ChangedFiles) > 0 {
		sb.WriteString("\n### Changed files\n\n")
		for _, f := range impl.ChangedFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	sb.WriteString("\n")
}

func writeDeterministicSection(sb *strings.Builder, packet review.ReviewPacket) {
	if len(packet.AlreadyProven) == 0 {
		sb.WriteString("No deterministically proven checks.\n\n")
		return
	}

	sb.WriteString("| Check | Status | Source | Provenance |\n")
	sb.WriteString("|-------|--------|--------|------------|\n")
	for _, c := range packet.AlreadyProven {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			c.ID, c.Status, c.Source, provenanceLabel(schema.ProvenanceKind(c.Provenance))))
	}
	sb.WriteString("\n")
}

func writeUncertainSection(sb *strings.Builder, packet review.ReviewPacket) {
	if len(packet.Unknown) == 0 {
		sb.WriteString("No uncertain checks.\n\n")
		return
	}

	for _, u := range packet.Unknown {
		sb.WriteString(fmt.Sprintf("- **`%s`** (%s) %s: %s\n",
			u.CheckID, u.Status, provenanceLabel(schema.ProvenanceKind(u.Provenance)), u.WhatIsUncertain))
	}
	sb.WriteString("\n")
}

func writeInspectSection(sb *strings.Builder, packet review.ReviewPacket) {
	if len(packet.HumanAttention) == 0 {
		sb.WriteString("No items require human inspection.\n\n")
		return
	}

	for _, a := range packet.HumanAttention {
		reqRefs := ""
		if len(a.RelevantRequirements) > 0 {
			reqRefs = " (requirements: " + strings.Join(a.RelevantRequirements, ", ") + ")"
		}
		sb.WriteString(fmt.Sprintf("- **`%s`** — %s %s%s\n",
			a.CheckID, a.Status, provenanceLabel(schema.ProvenanceKind(a.Provenance)), reqRefs))
	}
	sb.WriteString("\n")
}

func writeInspectReasonsSection(sb *strings.Builder, packet review.ReviewPacket) {
	if len(packet.HumanAttention) == 0 {
		sb.WriteString("No items require human inspection.\n\n")
		return
	}

	for _, a := range packet.HumanAttention {
		sb.WriteString(fmt.Sprintf("- **`%s`:** %s\n", a.CheckID, a.Reason))
	}
	sb.WriteString("\n")
}
