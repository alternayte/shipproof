package report

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/schema"
)

// badgePattern captures the word inside every grade badge on the page.
var badgePattern = regexp.MustCompile(`<span class="prov-badge[^"]*">([^<]*)</span>`)

func renderReportWithEveryLabel(t *testing.T) string {
	t.Helper()
	root, ev := setupReportTest(t, "SP-T26")
	defer os.RemoveAll(root)

	ev.Checks = []schema.Check{
		{ID: "c1", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		{ID: "c2", Status: "pass", Source: "calc", Provenance: schema.ProvenanceDerived},
		{ID: "c3", Status: "fail", Source: "agent", Provenance: schema.ProvenanceInferred},
		{ID: "c4", Status: "skip", Source: "person", Provenance: schema.ProvenanceHuman},
	}
	ev.UnexplainedChange = schema.UnexplainedEvidence{
		Measured:          true,
		CoverageAvailable: true,
		LineFindings:      []schema.UnexplainedLine{{File: "a.go", Symbol: "Run", StartLine: 1, EndLine: 4}},
		FileFindings:      []schema.UnexplainedFile{{Path: "b.go", IgnorePattern: "*_test.go"}},
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T26"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return sb.String()
}

// TestOnlyThreeGradesAppearInAReport is the proof for row H1 of the definition
// of done.
func TestOnlyThreeGradesAppearInAReport(t *testing.T) {
	html := renderReportWithEveryLabel(t)

	allowed := map[string]bool{}
	for _, value := range grade.All {
		allowed[string(value)] = true
	}

	matches := badgePattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		t.Fatal("the page holds no grade badge")
	}
	seen := map[string]bool{}
	for _, match := range matches {
		word := strings.TrimSpace(match[1])
		if !allowed[word] {
			t.Fatalf("the page holds the grade word %q, which is not one of three", word)
		}
		seen[word] = true
	}
	for _, want := range []string{"observed", "stated", "claimed"} {
		if !seen[want] {
			t.Fatalf("the page never shows the grade %q", want)
		}
	}
}

// TestTheOldGradeWordsAreGone asserts that no reader meets `derived` or
// `inferred` on the page, in a badge, in a style, or in prose.
func TestTheOldGradeWordsAreGone(t *testing.T) {
	html := strings.ToLower(renderReportWithEveryLabel(t))
	for _, word := range []string{"derived", "inferred"} {
		if strings.Contains(html, word) {
			t.Fatalf("the page holds the old grade word %q", word)
		}
	}
}

// TestTheReportSaysAClaimedCheckProvesNothing is the words-on-the-page rule of
// Section 6.
func TestTheReportSaysAClaimedCheckProvesNothing(t *testing.T) {
	html := strings.ToLower(renderReportWithEveryLabel(t))
	if !strings.Contains(html, "a claimed check never proves a requirement") {
		t.Fatal("the page never states that a claimed check proves nothing")
	}
}
