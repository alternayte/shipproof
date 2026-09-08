// Package verdict maps the state that ShipProof holds onto one of three
// verdicts. It renders the three-line block of Section 5 of the design
// document.
//
// The block is the whole answer for a reader who never used ShipProof. It
// therefore carries no machine token and no trade word. A count that no tool
// measured reads as not known. It never reads as zero.
package verdict

import (
	"fmt"
	"strings"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/phase"
	"github.com/alternayte/shipproof/internal/schema"
)

// Verdict names the outcome. Only these three values exist. No score exists.
type Verdict string

const (
	// Proven holds when every requirement carries a passing observed proof or
	// an accepted human proof, and no unexplained change remains.
	Proven Verdict = "PROVEN"
	// NotProven holds when the work is incomplete or unproven.
	NotProven Verdict = "NOT PROVEN"
	// Failed holds when a proof ran and failed, or the gate failed.
	Failed Verdict = "FAILED"
)

// JargonWords names every word that the block must exclude. A reader with no
// ShipProof knowledge does not know these words. Row U2 of the definition of
// done tests this list. Every entry is lower case, and the test folds case.
var JargonWords = []string{
	"provenance",
	"derived",
	"inferred",
	"attestation",
	"attest",
	"artifact",
	"matrix",
	"sidecar",
	"invariant",
	"telemetry",
	"snapshot",
	"harness",
	"schema",
	"gate",
	"phase",
	"blocker",
	"stale",
}

// Input holds every fact that the verdict rule reads.
type Input struct {
	ChangeID string
	// Phase names the state of the change and the next command.
	Phase phase.Result
	// Matrix holds one row per requirement. HasMatrix reports whether the
	// change holds enough state to build it.
	Matrix    coverage.Matrix
	HasMatrix bool
	// Unexplained holds the count of changed lines that match no requirement.
	// A nil value means that no tool measured the count. A nil value is not a
	// zero, and it never produces PROVEN.
	Unexplained *int
	// Checks holds the recorded checks. Only an observed check and a stated
	// check carry weight. A claimed check never proves a requirement, and it
	// never fails one either. An agent claim is not evidence in either
	// direction.
	Checks []schema.Check
}

// Block is the three-line answer.
type Block struct {
	Verdict Verdict `json:"verdict"`
	// Reason states the counts that drive the verdict.
	Reason string `json:"reason"`
	// Next states one action with a runnable command.
	Next string `json:"next"`
}

// String renders the block. It always holds three lines and a final newline.
func (b Block) String() string {
	return fmt.Sprintf("VERDICT: %s\n%s\nNEXT: %s\n", b.Verdict, b.Reason, b.Next)
}

// Decide applies the rule. The first condition that holds is the answer.
func Decide(input Input) Block {
	counts := count(input)
	next := nextAction(input, counts)

	switch {
	case input.Phase.Phase == phase.RunFailed || counts.failed > 0 || failedProofChecks(input) > 0:
		return Block{Verdict: Failed, Reason: failReason(input, counts), Next: next}
	case staleIntent(input):
		return Block{Verdict: NotProven, Reason: staleReason(input, counts), Next: next}
	case input.HasMatrix && counts.total > 0 && counts.settled == counts.total &&
		input.Unexplained != nil && *input.Unexplained == 0:
		return Block{Verdict: Proven, Reason: provenReason(counts), Next: next}
	default:
		return Block{Verdict: NotProven, Reason: openReason(input, counts), Next: next}
	}
}

// tally holds the requirement counts that the reason lines report.
type tally struct {
	total   int
	settled int
	failed  int
	open    []string
}

func count(input Input) tally {
	result := tally{}
	if !input.HasMatrix {
		return result
	}
	for _, row := range input.Matrix.Rows {
		result.total++
		switch row.State {
		case coverage.Proven, coverage.Accepted:
			result.settled++
		case coverage.Failed:
			result.failed++
		default:
			result.open = append(result.open, row.RequirementID)
		}
	}
	return result
}

// staleIntent reports whether the requirement document changed after ShipProof
// read it. A proof that ran against an older document proves nothing about the
// document that now stands, so the work is not proven.
func staleIntent(input Input) bool {
	if input.Phase.Phase == phase.IntentStale {
		return true
	}
	for _, check := range input.Checks {
		if check.ID == "intent:staleness" && check.Status == "fail" {
			return true
		}
	}
	return false
}

func staleReason(input Input, counts tally) string {
	return fmt.Sprintf("The requirement document changed after ShipProof read it, so no proof describes the document that now stands. %s",
		lineSentence(input))
}

func provenReason(counts tally) string {
	return fmt.Sprintf("%s. No changed line is left over.", carriesAProof(counts.total))
}

// carriesAProof states the count and agrees with it. "All 1 requirement carry"
// reads wrong, and a reader who meets it stops trusting the page.
func carriesAProof(total int) string {
	if total == 1 {
		return "The 1 requirement carries a proof"
	}
	return fmt.Sprintf("All %d requirements carry a proof", total)
}

// informationalPrefixes names the check identifiers that ShipProof produces to
// report the state of the pack. ShipProof owns this namespace, so the list
// cannot drift with a third-party tool name.
//
// A check under one of these prefixes is not a proof and it is not the gate.
// Section 5 reserves FAILED for a failed proof and a failed gate, so a failed
// check under one of these prefixes never produces that word. A stale intent
// makes the work NOT PROVEN, and the reason line says why.
var informationalPrefixes = []string{
	"intent:",
	"coverage:",
	"agent:",
}

// failedProofChecks counts the proof results and the gate results that failed.
// A claimed check is excluded, because nothing confirmed it. An informational
// check is excluded, because it reports a state rather than a proof.
func failedProofChecks(input Input) int {
	total := 0
	for _, check := range input.Checks {
		if check.Status != "fail" {
			continue
		}
		if !grade.FromProvenance(check.Provenance).Proves() {
			continue
		}
		if isInformational(check.ID) {
			continue
		}
		total++
	}
	return total
}

func isInformational(id string) bool {
	for _, prefix := range informationalPrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

func failReason(input Input, counts tally) string {
	if failed := failedProofChecks(input); counts.failed == 0 && failed > 0 {
		return fmt.Sprintf("%s failed. %s",
			capitalize(plural(failed, "check")), lineSentence(input))
	}
	if counts.failed > 0 {
		return fmt.Sprintf("%s failed a proof. %s",
			capitalize(plural(counts.failed, "requirement")), lineSentence(input))
	}
	return fmt.Sprintf("The repository command failed. %s", lineSentence(input))
}

func openReason(input Input, counts tally) string {
	if !input.HasMatrix || counts.total == 0 {
		return fmt.Sprintf("The change lists no requirement with a proof yet. %s",
			lineSentence(input))
	}
	if len(counts.open) > 0 {
		return fmt.Sprintf("%d of %d requirements have no proof. %s",
			len(counts.open), counts.total, lineSentence(input))
	}
	return fmt.Sprintf("%s. %s", carriesAProof(counts.total), lineSentence(input))
}

// lineSentence reports the unexplained change count. An unmeasured count says
// so. It never reads as zero.
func lineSentence(input Input) string {
	if input.Unexplained == nil {
		return "The number of changed lines that match no requirement is not known yet."
	}
	if *input.Unexplained == 0 {
		return "No changed line is left over."
	}
	return fmt.Sprintf("%d changed lines match no requirement.", *input.Unexplained)
}

// nextAction states one runnable command. It prefers the command that the
// phase names, and it adds the requirements that need a proof.
func nextAction(input Input, counts tally) string {
	command := strings.TrimSpace(input.Phase.NextCommand)
	if command == "" {
		command = defaultCommand(input)
	}
	if len(counts.open) > 0 {
		return fmt.Sprintf("run `%s` after you add a proof for %s.",
			command, list(counts.open))
	}
	// An unmeasured count is the only thing left between here and PROVEN, so
	// name the command that measures it. Repeating the command the reader just
	// ran moves nobody forward.
	if input.Unexplained == nil && counts.total > 0 && counts.settled == counts.total {
		identifier := strings.TrimSpace(input.ChangeID)
		return fmt.Sprintf("run `shipproof pack %s --base <the revision you started from>` to count the lines no requirement explains.", identifier)
	}
	return fmt.Sprintf("run `%s`.", command)
}

// defaultCommand names the command for a phase that carries none. The five
// commands of the design document are the only answers.
func defaultCommand(input Input) string {
	identifier := strings.TrimSpace(input.ChangeID)
	if identifier == "" {
		identifier = strings.TrimSpace(input.Phase.ChangeID)
	}
	suffix := ""
	if identifier != "" {
		suffix = " " + identifier
	}
	switch input.Phase.Phase {
	case phase.NoChange, phase.IntentStale:
		return "shipproof start" + suffix
	case phase.NeedsEvidence, phase.ReadyForHuman:
		return "shipproof pack" + suffix
	default:
		return "shipproof prove" + suffix
	}
}

// list joins requirement identifiers into plain words.
func list(items []string) string {
	switch len(items) {
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func plural(number int, noun string) string {
	if number == 1 {
		return fmt.Sprintf("%d %s", number, noun)
	}
	return fmt.Sprintf("%d %ss", number, noun)
}

func capitalize(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + text[1:]
}
