package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func commentPack() schema.EvidencePack {
	pack := unsignedPack()
	pack.Verdict = schema.VerdictEvidence{
		Verdict: "NOT PROVEN",
		Reason:  "1 of 2 requirements have no proof. 14 changed lines match no requirement.",
		Next:    "run `shipproof prove SP-028` after you add a proof for R2.",
	}
	pack.Requirements = []schema.RequirementRow{
		{ID: "R1", Statement: "The tool records the intent.", ProofRefs: []string{"go test -run TestIntent"},
			State: "proven", Grade: "observed"},
		{ID: "R2", Statement: "The tool signs the pack.", State: "unproven", Grade: "claimed"},
	}
	pack.UnexplainedChange = schema.UnexplainedEvidence{
		Measured:          true,
		CoverageAvailable: true,
		LineFindings: []schema.UnexplainedLine{
			{File: "internal/a.go", Symbol: "Run", StartLine: 10, EndLine: 23},
		},
	}
	return pack
}

func renderComment(t *testing.T, pack schema.EvidencePack) string {
	t.Helper()
	path := writePackFile(t, pack)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"pack", "--comment", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	return stdout.String()
}

// TestTheCommentOpensWithTheVerdictBlock is the proof for row P3 of the
// definition of done.
func TestTheCommentOpensWithTheVerdictBlock(t *testing.T) {
	body := renderComment(t, commentPack())

	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) < 3 {
		t.Fatalf("the comment holds fewer than three lines:\n%s", body)
	}
	if !strings.HasPrefix(lines[0], "VERDICT: ") {
		t.Fatalf("line 1 = %q, want the verdict line first", lines[0])
	}
	if !strings.Contains(lines[1], "1 of 2 requirements have no proof") {
		t.Fatalf("line 2 = %q, want the count line", lines[1])
	}
	if !strings.HasPrefix(lines[2], "NEXT: ") {
		t.Fatalf("line 3 = %q, want the next line", lines[2])
	}
}

// TestTheCommentHoldsTheRequirementTable holds requirement R5.
func TestTheCommentHoldsTheRequirementTable(t *testing.T) {
	body := renderComment(t, commentPack())

	if !strings.Contains(body, "| Requirement | Proof | Grade |") {
		t.Fatalf("the comment holds no requirement table:\n%s", body)
	}
	for _, want := range []string{"R1", "R2", "go test -run TestIntent", "observed", "claimed"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the table never names %q:\n%s", want, body)
		}
	}
}

// TestTheCommentStatesTheUnexplainedCount holds requirement R6.
func TestTheCommentStatesTheUnexplainedCount(t *testing.T) {
	body := renderComment(t, commentPack())

	if !strings.Contains(body, "14 changed lines match no requirement") {
		t.Fatalf("the comment states no unexplained count:\n%s", body)
	}
	if !strings.Contains(body, "internal/a.go:10-23") {
		t.Fatalf("the comment references no line:\n%s", body)
	}
}

// TestTheCommentStatesAnUnmeasuredCountAsUnknown holds the honesty rule. An
// absent measurement never reads as zero.
func TestTheCommentStatesAnUnmeasuredCountAsUnknown(t *testing.T) {
	pack := commentPack()
	pack.UnexplainedChange = schema.UnexplainedEvidence{Measured: false}
	pack.EmptySections["unexplained_change"] = "no base revision is known."

	body := renderComment(t, pack)
	if !strings.Contains(body, "not known") {
		t.Fatalf("the comment does not report the count as unknown:\n%s", body)
	}
	if strings.Contains(body, "0 changed lines") {
		t.Fatalf("the comment reports an unmeasured count as zero:\n%s", body)
	}
}

// TestTheCommentStatesTheAgentRecord holds requirement R7.
func TestTheCommentStatesTheAgentRecord(t *testing.T) {
	withoutAgent := renderComment(t, commentPack())
	if !strings.Contains(withoutAgent, "No agent record") {
		t.Fatalf("the comment never states the absent agent record:\n%s", withoutAgent)
	}

	pack := commentPack()
	pack.Agent = &schema.AgentEvidence{Provider: "anthropic", Model: "opus", SessionID: "s1"}
	delete(pack.EmptySections, "agent")
	withAgent := renderComment(t, pack)
	for _, want := range []string{"anthropic", "opus", "s1"} {
		if !strings.Contains(withAgent, want) {
			t.Fatalf("the comment never names %q:\n%s", want, withAgent)
		}
	}
}

// TestTheCommentNamesWhereThePackLives holds requirement R8.
func TestTheCommentNamesWhereThePackLives(t *testing.T) {
	body := renderComment(t, commentPack())
	if !strings.Contains(body, "evidence-pack.json") {
		t.Fatalf("the comment never names the pack:\n%s", body)
	}
}

// TestTheCommentHoldsNothingElse holds requirement R9. Section 9.2 lists what
// the comment carries and it ends with "and nothing more".
func TestTheCommentHoldsNothingElse(t *testing.T) {
	body := strings.ToLower(renderComment(t, commentPack()))
	for _, word := range []string{
		"## checks", "## implementation", "## intent", "diff stat",
		"empty_sections", "schema_version", "provenance",
	} {
		if strings.Contains(body, word) {
			t.Fatalf("the comment holds the section %q, which Section 9.2 omits:\n%s", word, body)
		}
	}
}
