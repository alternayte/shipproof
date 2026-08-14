package document

func AssessReadiness(findings []Finding, semanticReviewPending bool) Readiness {
	result := Readiness{
		State:                 ReadinessReady,
		Provisional:           semanticReviewPending,
		SemanticReviewPending: semanticReviewPending,
	}

	for _, finding := range findings {
		switch finding.Class {
		case FindingBlocker:
			result.Blockers++
		case FindingDecision:
			result.Decisions++
		case FindingAssumption:
			result.Assumptions++
		case FindingRisk:
			result.Risks++
		case FindingSuggestion:
			result.Suggestions++
		case FindingNit:
			result.Nits++
		}
	}

	switch {
	case result.Blockers > 0 || result.Decisions > 0:
		result.State = ReadinessBlocked
	case result.Assumptions > 0 || result.Risks > 0:
		result.State = ReadinessReadyWithAssumptions
	default:
		result.State = ReadinessReady
	}

	return result
}
