package report

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/verdict"
)

func renderFor(t *testing.T, mutate func(*schema.EvidencePack)) string {
	t.Helper()
	root, ev := setupReportTest(t, "SP-T32")
	defer os.RemoveAll(root)
	if mutate != nil {
		mutate(&ev)
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T32"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return sb.String()
}

// TestTheVerdictBlockIsTheFirstContent holds requirement R1. Section 12 says
// the verdict block sits above the fold.
func TestTheVerdictBlockIsTheFirstContent(t *testing.T) {
	html := renderFor(t, nil)

	body := html[strings.Index(html, "<body"):]
	verdictAt := strings.Index(body, "VERDICT:")
	if verdictAt < 0 {
		t.Fatalf("the page holds no verdict block:\n%s", html)
	}
	for _, later := range []string{"Requirements", "Checks", "Unexplained", "Agent"} {
		at := strings.Index(body, later)
		if at >= 0 && at < verdictAt {
			t.Fatalf("the section %q appears before the verdict block", later)
		}
	}
}

// TestTheVerdictBlockHoldsThreeLines holds requirement R1.
func TestTheVerdictBlockHoldsThreeLines(t *testing.T) {
	html := renderFor(t, nil)
	for _, want := range []string{"VERDICT:", "NEXT:"} {
		if !strings.Contains(html, want) {
			t.Fatalf("the page holds no %q line:\n%s", want, html)
		}
	}
	if !strings.Contains(html, "2 of 2 requirements have no proof") {
		t.Fatalf("the page holds no reason line:\n%s", html)
	}
}

// TestEachVerdictCarriesItsOwnColour holds requirement R2.
func TestEachVerdictCarriesItsOwnColour(t *testing.T) {
	cases := []struct{ word, class string }{
		{"PROVEN", "verdict-proven"},
		{"NOT PROVEN", "verdict-not-proven"},
		{"FAILED", "verdict-failed"},
	}
	for _, testCase := range cases {
		html := renderFor(t, func(pack *schema.EvidencePack) {
			pack.Verdict.Verdict = testCase.word
		})
		if !strings.Contains(html, testCase.class) {
			t.Fatalf("the verdict %q carries no %q class", testCase.word, testCase.class)
		}
	}
}

// TestNoOtherColourCarriesMeaning holds requirement R3. Section 12 assigns
// three colours. A pill that means a fourth thing must not exist.
func TestNoOtherColourCarriesMeaning(t *testing.T) {
	html := renderFor(t, nil)
	for _, class := range []string{"pill-skip", "pill-unknown", "prov-derived", "prov-inferred", "prov-human"} {
		if strings.Contains(html, class) {
			t.Fatalf("the page holds the colour class %q, which carries a fourth meaning", class)
		}
	}
}

// TestTheRequirementTableHoldsFourColumns holds requirement R4.
func TestTheRequirementTableHoldsFourColumns(t *testing.T) {
	html := renderFor(t, nil)
	for _, column := range []string{"<th>Requirement</th>", "<th>Proof</th>", "<th>State</th>", "<th>Grade</th>"} {
		if !strings.Contains(html, column) {
			t.Fatalf("the requirement table holds no %s:\n%s", column, html)
		}
	}
	for _, want := range []string{"R1", "R2", "go test -run TestR1"} {
		if !strings.Contains(html, want) {
			t.Fatalf("the requirement table never names %q", want)
		}
	}
}

// TestTheChecksAreGroupedByGrade holds requirements R5 and R6.
func TestTheChecksAreGroupedByGrade(t *testing.T) {
	html := renderFor(t, func(pack *schema.EvidencePack) {
		pack.Checks = []schema.Check{
			{ID: "c-claimed", Status: "unknown", Source: "agent", Grade: "claimed", Provenance: schema.ProvenanceInferred},
			{ID: "c-observed", Status: "pass", Source: "junit", Grade: "observed", Provenance: schema.ProvenanceObserved},
			{ID: "c-stated", Status: "pass", Source: "person", Grade: "stated", Provenance: schema.ProvenanceHuman},
		}
		delete(pack.EmptySections, "checks")
	})

	observedAt := strings.Index(html, "c-observed")
	statedAt := strings.Index(html, "c-stated")
	claimedAt := strings.Index(html, "c-claimed")
	if observedAt < 0 || statedAt < 0 || claimedAt < 0 {
		t.Fatalf("the page does not hold every check:\n%s", html)
	}
	if !(observedAt < statedAt && statedAt < claimedAt) {
		t.Fatal("the checks do not appear in the order observed, stated, claimed")
	}
	if !strings.Contains(html, "A claimed check never proves a requirement") {
		t.Fatal("the claimed group does not state that a claimed check proves nothing")
	}
}

// TestAnUnmeasuredCountReadsAsNotKnown holds requirement R7.
func TestAnUnmeasuredCountReadsAsNotKnown(t *testing.T) {
	html := renderFor(t, func(pack *schema.EvidencePack) {
		pack.UnexplainedChange = schema.UnexplainedEvidence{Measured: false}
		pack.EmptySections["unexplained_change"] = "no base revision is known."
	})
	if !strings.Contains(html, "not known") {
		t.Fatalf("the page does not report the count as unknown:\n%s", html)
	}
	if !strings.Contains(html, "no base revision is known.") {
		t.Fatal("the page does not name the reason")
	}
}

// TestThePageNamesTheSignatureState holds requirement R10.
func TestThePageNamesTheSignatureState(t *testing.T) {
	html := renderFor(t, nil)
	if !strings.Contains(strings.ToLower(html), "unsigned") {
		t.Fatalf("the page never states that the pack is unsigned:\n%s", html)
	}
}

// TestTheVerdictBlockHoldsNoJargon keeps rule U2 true on the page.
func TestTheVerdictBlockHoldsNoJargon(t *testing.T) {
	html := renderFor(t, nil)
	block := regexp.MustCompile(`(?s)<section class="verdict.*?</section>`).FindString(html)
	if block == "" {
		t.Fatalf("the page holds no verdict section:\n%s", html)
	}
	// The class names are markup, not prose. Strip every tag first.
	prose := strings.ToLower(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(block, " "))
	for _, word := range verdict.JargonWords {
		if strings.Contains(prose, word) {
			t.Fatalf("the verdict block holds the jargon word %q:\n%s", word, prose)
		}
	}
}

// TestTheFourSectionsRunInOrder holds the shape of Section 12. After the
// verdict block the report shows the requirement table, the unexplained change
// list, the check list, and the agent record, in that order.
func TestTheFourSectionsRunInOrder(t *testing.T) {
	html := renderFor(t, nil)

	previous := strings.Index(html, "VERDICT:")
	if previous < 0 {
		t.Fatal("the page holds no verdict block")
	}
	for _, heading := range []string{
		"<h2>Requirements</h2>",
		"<h2>Unexplained change</h2>",
		"<h2>Checks</h2>",
		"<h2>Agent record</h2>",
	} {
		at := strings.Index(html, heading)
		if at < 0 {
			t.Fatalf("the page holds no %s", heading)
		}
		if at < previous {
			t.Fatalf("the section %s appears out of order", heading)
		}
		previous = at
	}
}

// TestThePageStatesAnAbsentAgentRecord holds requirement R8.
func TestThePageStatesAnAbsentAgentRecord(t *testing.T) {
	html := renderFor(t, nil)
	if !strings.Contains(html, "No agent record exists for this change.") {
		t.Fatalf("the page hides the absent agent record:\n%s", html)
	}
}
