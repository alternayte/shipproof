package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/verdict"
)

// TestStatusPrintsTheVerdictBlockFirst is the proof for row C4 of the
// definition of done.
func TestStatusPrintsTheVerdictBlockFirst(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"status", "SP-999"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) < 3 {
		t.Fatalf("stdout holds fewer than three lines:\n%s", stdout.String())
	}
	if !strings.HasPrefix(lines[0], "VERDICT: ") {
		t.Fatalf("line 1 = %q, want the verdict line first", lines[0])
	}
	word := strings.TrimPrefix(lines[0], "VERDICT: ")
	switch verdict.Verdict(word) {
	case verdict.Proven, verdict.NotProven, verdict.Failed:
	default:
		t.Fatalf("verdict word = %q, want one of three", word)
	}
	if strings.TrimSpace(lines[1]) == "" {
		t.Fatalf("line 2 is empty:\n%s", stdout.String())
	}
	if !strings.HasPrefix(lines[2], "NEXT: ") {
		t.Fatalf("line 3 = %q, want the next line", lines[2])
	}
	if !strings.Contains(lines[2], "shipproof ") {
		t.Fatalf("line 3 = %q, want a runnable command", lines[2])
	}
}

// TestStatusBlockHoldsNoJargon is the proof for row U2 of the definition of
// done, at the command surface.
func TestStatusBlockHoldsNoJargon(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"status", "SP-999"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	lines := strings.Split(stdout.String(), "\n")
	block := strings.ToLower(strings.Join(lines[:3], "\n"))
	for _, word := range verdict.JargonWords {
		if strings.Contains(block, word) {
			t.Fatalf("the block holds the jargon word %q:\n%s", word, block)
		}
	}
}

func TestStatusJSONCarriesTheVerdict(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"status", "SP-999", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var payload struct {
		Verdict *verdict.Block `json:"verdict"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Verdict == nil {
		t.Fatalf("the JSON output holds no verdict:\n%s", stdout.String())
	}
	if payload.Verdict.Verdict != verdict.NotProven {
		t.Fatalf("verdict = %q, want %q", payload.Verdict.Verdict, verdict.NotProven)
	}
	if payload.Verdict.Next == "" || payload.Verdict.Reason == "" {
		t.Fatalf("the verdict is incomplete: %+v", payload.Verdict)
	}
}
