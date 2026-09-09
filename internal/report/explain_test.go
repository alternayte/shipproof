package report

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/verdict"
)

// The seven findings of the reader review of 2026-09-09. Each test names the
// question a reader asked and the requirement of SP-039 that answers it.

// R1. What is unexplained change, and why is the count not known?
func TestTheUnexplainedSectionExplainsItself(t *testing.T) {
	html := flatten(renderFor(t, func(pack *schema.EvidencePack) {
		pack.UnexplainedChange = schema.UnexplainedEvidence{Measured: false}
		pack.EmptySections["unexplained_change"] = "no base revision is known."
	}))
	if !strings.Contains(html, "no requirement in this change explains") {
		t.Errorf("the section never says what unexplained change means:\n%s", section(t, html, "Unexplained change"))
	}
	// An unknown count must say what would make it known.
	if !strings.Contains(html, "--base") {
		t.Errorf("the section does not say what would measure the count:\n%s", section(t, html, "Unexplained change"))
	}
}

// R2. What does "add a proof" ask of me?
func TestThePageSaysHowToAddAProof(t *testing.T) {
	html := flatten(renderFor(t, nil))
	for _, want := range []string{
		"verification.json",
		"exits 0",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the page never names %q, so a reader cannot act on \"add a proof\"", want)
		}
	}
}

// R3. Which checks are these, how are they chosen, and what is source?
func TestTheCheckListExplainsAChecK(t *testing.T) {
	html := flatten(renderFor(t, func(pack *schema.EvidencePack) {
		pack.Checks = []schema.Check{
			{ID: "c1", Status: "pass", Source: "junit", Grade: "observed", Provenance: schema.ProvenanceObserved},
		}
		delete(pack.EmptySections, "checks")
	}))
	body := section(t, html, "Checks")
	if !strings.Contains(body, "one thing ShipProof looked at") {
		t.Errorf("the check list never says what a check is:\n%s", body)
	}
	if !strings.Contains(body, "which tool reported") {
		t.Errorf("the check list never explains the source column:\n%s", body)
	}
}

// R4. What would the agent record hold?
func TestTheAgentSectionSaysWhatItWouldHold(t *testing.T) {
	html := flatten(renderFor(t, nil))
	body := section(t, html, "Agent record")
	for _, want := range []string{"model", "session"} {
		if !strings.Contains(body, want) {
			t.Errorf("the absent agent record never names %q as a field it would hold:\n%s", want, body)
		}
	}
}

// R5. Where in the delivery flow does this run?
func TestThePageSaysWhereItWasProduced(t *testing.T) {
	local := flatten(renderFor(t, nil))
	if !strings.Contains(local, "on a developer machine") {
		t.Errorf("an unsigned report does not say a person produced it:\n%s", head(local))
	}

	signed := flatten(renderFor(t, func(pack *schema.EvidencePack) {
		pack.Attestation = &schema.AttestationEvidence{
			Format: "in-toto", Signature: "MEUCIQ", Subject: "1cceb33abc",
		}
		delete(pack.EmptySections, "attestation")
	}))
	if !strings.Contains(signed, "build system") {
		t.Errorf("a signed report does not say the build system produced it:\n%s", head(signed))
	}
	if !strings.Contains(signed, "1cceb33") {
		t.Errorf("the report does not name the revision it judged:\n%s", head(signed))
	}
}

// R6. Where did the requirements come from?
func TestTheRequirementsNameTheirSource(t *testing.T) {
	html := flatten(renderFor(t, func(pack *schema.EvidencePack) {
		pack.Intent.SourcePath = "docs/my-feature.md"
		pack.Requirements[0].SourceAnchor = "- MUST reject a request over the limit."
	}))
	if !strings.Contains(html, "docs/my-feature.md") {
		t.Errorf("the page never names the document the requirements came from")
	}
	if !strings.Contains(html, "MUST reject a request over the limit.") {
		t.Errorf("the requirement row does not show the line it came from")
	}
}

// R7. What is the hash for?
func TestTheHashExplainsItself(t *testing.T) {
	html := flatten(renderFor(t, nil))
	if !strings.Contains(html, "changed since") {
		t.Errorf("the hash carries no explanation:\n%s", section(t, html, "Intent"))
	}
}

// R8. The verdict block does not change. It passed the review, and no
// suggestion reopens a criterion that a proof met.
func TestTheVerdictBlockIsUntouched(t *testing.T) {
	html := flatten(renderFor(t, nil))
	block := regexp.MustCompile(`(?s)<section class="verdict.*?</section>`).FindString(html)
	if block == "" {
		t.Fatal("the page holds no verdict block")
	}
	prose := strings.ToLower(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(block, " "))
	for _, word := range verdict.JargonWords {
		if strings.Contains(prose, word) {
			t.Fatalf("the verdict block gained the jargon word %q: %s", word, prose)
		}
	}
	// The block holds three lines and nothing more.
	for _, added := range []string{"verification.json", "which tool reported", "changed since"} {
		if strings.Contains(block, added) {
			t.Fatalf("an explanation leaked into the verdict block: %q", added)
		}
	}
}

// TestEverySectionCarriesAnExplanation is the general rule behind the seven.
func TestEverySectionCarriesAnExplanation(t *testing.T) {
	html := flatten(renderFor(t, nil))
	headings := regexp.MustCompile(`<h2>([^<]+)</h2>`).FindAllStringSubmatch(html, -1)
	if len(headings) == 0 {
		t.Fatal("the page holds no section")
	}
	for _, heading := range headings {
		body := section(t, html, heading[1])
		if !strings.Contains(body, `class="explain"`) {
			t.Errorf("the section %q carries no one-sentence explanation", heading[1])
		}
	}
}

// section returns the markup of one section, by its heading. It collapses
// every run of whitespace, so a line wrap in the template never breaks an
// assertion about the prose.
func section(t *testing.T, html, heading string) string {
	t.Helper()
	start := strings.Index(html, "<h2>"+heading+"</h2>")
	if start < 0 {
		t.Fatalf("the page holds no section %q", heading)
	}
	end := strings.Index(html[start:], "</section>")
	if end < 0 {
		end = len(html) - start
	}
	return flatten(html[start : start+end])
}

// flatten collapses every run of whitespace into one space. The template wraps
// its prose for a reader of the source, and a reader of the page sees one
// sentence.
func flatten(text string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
}

func head(html string) string {
	if len(html) > 1500 {
		return html[:1500]
	}
	return html
}

// TestThePageReadsOnAPhone holds the practical half of Section 2 principle 4.
// A reader often meets this page on a phone, and a page with no viewport tag
// renders at desktop width and forces them to pinch and zoom.
func TestThePageReadsOnAPhone(t *testing.T) {
	html := renderFor(t, nil)
	if !strings.Contains(html, `name="viewport"`) {
		t.Error("the page carries no viewport tag, so a phone renders it at desktop width")
	}
	if !strings.Contains(html, "max-width: 640px") {
		t.Error("the page carries no narrow-screen rules")
	}
	// A wide table must scroll on its own, never push the page sideways.
	if !strings.Contains(flatten(html), "table { display: block; overflow-x: auto; }") {
		t.Error("a wide table can push the whole page sideways on a phone")
	}
}

// TestTheSourceDocumentSaysWhatItIs holds a finding from the second reader
// review of 2026-09-09. A reader met "read from checkout.md" and asked what
// that file was. A bare filename names a thing without saying what kind of
// thing it is.
func TestTheSourceDocumentSaysWhatItIs(t *testing.T) {
	html := flatten(renderFor(t, func(pack *schema.EvidencePack) {
		pack.Intent.SourcePath = "docs/checkout-retry.md"
	}))
	if !strings.Contains(html, "requirements document <code>docs/checkout-retry.md</code>") {
		t.Errorf("the page names the file without saying what it is:\n%s", section(t, html, "Intent"))
	}
}

// TestThePageStaysShort holds what the second review praised. The readers
// liked that the page is not wordy, so an explanation must stay one sentence.
// A fix for one confusion must never bloat the page.
func TestThePageStaysShort(t *testing.T) {
	html := renderFor(t, nil)
	pattern := regexp.MustCompile(`(?s)<p class="explain">(.*?)</p>`)
	matches := pattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		t.Fatal("the page holds no explanation")
	}
	for _, match := range matches {
		prose := strings.TrimSpace(flatten(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(match[1], "")))
		if words := len(strings.Fields(prose)); words > 65 {
			t.Errorf("an explanation runs to %d words, which is no longer one short answer:\n%s", words, prose)
		}
	}
}
