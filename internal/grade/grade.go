// Package grade holds the three provenance grades of Section 6 of the design
// document. It maps the machine label onto the word that a human report
// prints.
//
// The machine record keeps the finer label. The human report keeps three
// words. A reader who never used ShipProof must tell a measured result from an
// asserted one, and three words do that. Four words do not.
package grade

import "github.com/alternayte/shipproof/internal/schema"

// Grade names how ShipProof learned a result. Only these three values exist.
type Grade string

const (
	// Observed holds when a tool ran and a machine recorded the result.
	Observed Grade = "observed"
	// Stated holds when a person accepted the result and signed for it.
	Stated Grade = "stated"
	// Claimed holds when an agent or a person asserted the result and no tool
	// confirmed it.
	Claimed Grade = "claimed"
)

// All lists every grade, in the order of decreasing strength.
var All = []Grade{Observed, Stated, Claimed}

// FromProvenance maps a machine label onto a grade. The old label `derived`
// and the old label `inferred` both collapse into Claimed. A label that this
// package does not know also maps to Claimed, because an unknown label is not
// evidence.
func FromProvenance(kind schema.ProvenanceKind) Grade {
	switch kind {
	case schema.ProvenanceObserved:
		return Observed
	case schema.ProvenanceHuman:
		return Stated
	default:
		return Claimed
	}
}

// Proves reports whether a check with this grade can make a requirement
// proven. A claimed check never can.
func (g Grade) Proves() bool { return g != Claimed }
