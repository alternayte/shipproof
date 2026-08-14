package document

import "testing"

func TestSuggestionsDoNotBlockReadiness(t *testing.T) {
	t.Parallel()

	readiness := AssessReadiness([]Finding{{Class: FindingSuggestion}, {Class: FindingNit}}, false)
	if readiness.State != ReadinessReady {
		t.Fatalf("state = %q, want %q", readiness.State, ReadinessReady)
	}
}

func TestAssumptionProducesReadyWithAssumptions(t *testing.T) {
	t.Parallel()

	readiness := AssessReadiness([]Finding{{Class: FindingAssumption}}, false)
	if readiness.State != ReadinessReadyWithAssumptions {
		t.Fatalf("state = %q, want %q", readiness.State, ReadinessReadyWithAssumptions)
	}
}

func TestDecisionBlocksReadiness(t *testing.T) {
	t.Parallel()

	readiness := AssessReadiness([]Finding{{Class: FindingDecision}}, false)
	if readiness.State != ReadinessBlocked {
		t.Fatalf("state = %q, want %q", readiness.State, ReadinessBlocked)
	}
}
