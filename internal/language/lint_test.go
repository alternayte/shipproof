package language

import "testing"

func TestLongSentenceIsSuggestion(t *testing.T) {
	t.Parallel()

	content := "The system records all delivery attempts and their outcomes so operators can investigate production failures without accessing application logs from several different services across multiple production regions and environments."
	findings := Lint(content, Glossary{}, DefaultOptions())
	if len(findings) == 0 {
		t.Fatal("expected sentence length finding")
	}
	if findings[0].Class != "suggestion" {
		t.Fatalf("class = %q, want suggestion", findings[0].Class)
	}
}

func TestCodeFenceIsIgnored(t *testing.T) {
	t.Parallel()

	content := "```text\nThis is a very long line that should not be linted because exact code and quoted technical material must not be rewritten by the language profile at all.\n```"
	if findings := Lint(content, Glossary{}, DefaultOptions()); len(findings) != 0 {
		t.Fatalf("findings = %+v, want none", findings)
	}
}
