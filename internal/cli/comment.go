package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/alternayte/shipproof/internal/schema"
)

// runPackComment prints the pull-request comment of Section 9.2. The comment
// is the product for most readers. It holds the verdict block, the requirement
// table, the unexplained change count, the agent record, and one sentence that
// names where the pack lives. It holds nothing more.
func runPackComment(path string, stdout, stderr io.Writer) int {
	loaded, ok := loadPackFile(path, stderr)
	if !ok {
		return 1
	}
	fmt.Fprint(stdout, commentBody(loaded, path))
	return 0
}

func commentBody(pack schema.EvidencePack, path string) string {
	var body strings.Builder

	// The verdict block sits above everything. Section 5 fixes its shape.
	fmt.Fprintf(&body, "VERDICT: %s\n", pack.Verdict.Verdict)
	fmt.Fprintf(&body, "%s\n", pack.Verdict.Reason)
	fmt.Fprintf(&body, "NEXT: %s\n\n", pack.Verdict.Next)

	body.WriteString("| Requirement | Proof | Grade |\n")
	body.WriteString("| --- | --- | --- |\n")
	if len(pack.Requirements) == 0 {
		fmt.Fprintf(&body, "| none | %s | — |\n", reasonFor(pack, "requirements"))
	}
	for _, row := range pack.Requirements {
		proof := strings.Join(row.ProofRefs, "<br>")
		if proof == "" {
			proof = "no proof"
		}
		fmt.Fprintf(&body, "| %s | %s | %s |\n", cell(row.ID), cell(proof), cell(row.Grade))
	}
	body.WriteString("\n")

	body.WriteString(unexplainedSentence(pack))
	body.WriteString("\n\n")

	body.WriteString(agentSentence(pack))
	body.WriteString("\n\n")

	fmt.Fprintf(&body, "The full evidence pack is the build artifact `%s`.\n", path)
	return body.String()
}

// unexplainedSentence reports the count and the lines. An unmeasured count
// reads as not known. It never reads as zero.
func unexplainedSentence(pack schema.EvidencePack) string {
	if !pack.UnexplainedChange.Measured {
		return "The number of changed lines that match no requirement is not known. " +
			sentence(reasonFor(pack, "unexplained_change"))
	}

	total := 0
	var references []string
	for _, finding := range pack.UnexplainedChange.LineFindings {
		span := finding.EndLine - finding.StartLine + 1
		if span < 1 {
			span = 1
		}
		total += span
		references = append(references, fmt.Sprintf("`%s:%d-%d`", finding.File, finding.StartLine, finding.EndLine))
	}
	if total == 0 {
		return "No changed line is left over."
	}
	return fmt.Sprintf("%d changed lines match no requirement: %s.",
		total, strings.Join(references, ", "))
}

// agentSentence names the model and the session. An absent record says so.
func agentSentence(pack schema.EvidencePack) string {
	record := pack.Agent
	if record == nil {
		return "No agent record exists for this change."
	}
	var parts []string
	for _, field := range []struct{ name, value string }{
		{"provider", record.Provider},
		{"model", record.Model},
		{"session", record.SessionID},
	} {
		// A field that no tool reported stays missing. It never reads as blank.
		if strings.TrimSpace(field.value) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s `%s`", field.name, field.value))
	}
	if len(parts) == 0 {
		return "An agent record exists, and it names no provider, no model, and no session."
	}
	return "Agent record: " + strings.Join(parts, ", ") + "."
}

// reasonFor reads the stated reason for an empty section. Rule 2 of Section 7
// guarantees one, and a pack that states none says so plainly.
func reasonFor(pack schema.EvidencePack, name string) string {
	if reason := strings.TrimSpace(pack.EmptySections[name]); reason != "" {
		return reason
	}
	return "The pack states no reason."
}

// sentence starts a reason with a capital letter. A reason reads as a clause
// inside the pack, and it reads as a sentence in the comment.
func sentence(reason string) string {
	if reason == "" {
		return reason
	}
	return strings.ToUpper(reason[:1]) + reason[1:]
}

// cell keeps one table cell on one row. A pipe would break the table.
func cell(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|")
}
