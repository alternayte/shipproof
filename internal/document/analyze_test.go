package document

import (
	"testing"

	"github.com/alternayte/shipproof/internal/language"
)

func TestPRDReadyEnoughDoesNotRequirePerfectTemplate(t *testing.T) {
	t.Parallel()

	content := `# Retry transient webhook deliveries

## Problem
Customers lose webhook deliveries when an endpoint has a short outage.

## Users and outcome
A customer must receive a delivery after a transient endpoint failure.
Success means the delivery completes without manual replay.

## Scope
Retry transient HTTP failures. Do not add customer-configurable policies in this change.

## Acceptance
Verify a 503 response is retried and eventually succeeds.
`
	review := Analyze("prd.md", content, AnalyzeOptions{Kind: KindPRD, Language: language.DefaultOptions()})
	if review.Readiness.State == ReadinessBlocked {
		t.Fatalf("unexpected blocked readiness: %+v", review.Findings)
	}
}

func TestContextlessNFRIsSuggestionNotBlocker(t *testing.T) {
	t.Parallel()

	content := `# Search

## Problem
Users wait too long for search.

## Users and outcome
Users get useful results.

## Requirements
The API must be scalable.

## Acceptance
Measure the response time.
`
	review := Analyze("prd.md", content, AnalyzeOptions{Kind: KindPRD, Language: language.DefaultOptions()})
	found := false
	for _, finding := range review.Findings {
		if finding.ID == "DOC-CONTEXTLESS-QUALITY" {
			found = true
			if finding.Class != FindingSuggestion {
				t.Fatalf("class = %q, want suggestion", finding.Class)
			}
		}
	}
	if !found {
		t.Fatal("expected contextless quality finding")
	}
}

func TestPlaceholderBlocksAsDecision(t *testing.T) {
	t.Parallel()

	content := `# Retry

## Problem
Users lose deliveries.

## Users and outcome
Customers get their deliveries.

## Acceptance
TBD: decide how long retries continue.
`
	review := Analyze("prd.md", content, AnalyzeOptions{Kind: KindPRD, Language: language.DefaultOptions()})
	if review.Readiness.State != ReadinessBlocked {
		t.Fatalf("state = %q, want blocked", review.Readiness.State)
	}
}
