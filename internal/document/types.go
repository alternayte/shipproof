package document

import "fmt"

type Kind string

const (
	KindPRD Kind = "prd"
	KindSDD Kind = "sdd"
)

func ParseKind(value string) (Kind, error) {
	switch Kind(value) {
	case KindPRD:
		return KindPRD, nil
	case KindSDD:
		return KindSDD, nil
	default:
		return "", fmt.Errorf("unsupported document kind %q", value)
	}
}

type ReadinessState string

const (
	ReadinessShaping              ReadinessState = "shaping"
	ReadinessBlocked              ReadinessState = "blocked"
	ReadinessReadyWithAssumptions ReadinessState = "ready_with_assumptions"
	ReadinessReady                ReadinessState = "ready"
)

type FindingClass string

const (
	FindingBlocker    FindingClass = "blocker"
	FindingDecision   FindingClass = "decision"
	FindingAssumption FindingClass = "assumption"
	FindingRisk       FindingClass = "risk"
	FindingSuggestion FindingClass = "suggestion"
	FindingNit        FindingClass = "nit"
)

type Finding struct {
	ID              string       `json:"id"`
	Class           FindingClass `json:"class"`
	Category        string       `json:"category"`
	Line            int          `json:"line,omitempty"`
	Claim           string       `json:"claim"`
	Reason          string       `json:"reason"`
	SuggestedAction string       `json:"suggested_action,omitempty"`
	Source          string       `json:"source"`
}

type Readiness struct {
	State                 ReadinessState `json:"state"`
	Provisional           bool           `json:"provisional"`
	SemanticReviewPending bool           `json:"semantic_review_pending"`
	Blockers              int            `json:"blockers"`
	Decisions             int            `json:"decisions"`
	Assumptions           int            `json:"assumptions"`
	Risks                 int            `json:"risks"`
	Suggestions           int            `json:"suggestions"`
	Nits                  int            `json:"nits"`
}

type Review struct {
	Kind      Kind      `json:"kind"`
	Path      string    `json:"path"`
	Readiness Readiness `json:"readiness"`
	Findings  []Finding `json:"findings"`
}
