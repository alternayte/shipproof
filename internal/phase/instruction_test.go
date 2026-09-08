package phase

import "testing"

// instructionFiles names the three files of Section 8.2. A phase may point at
// one of them and at nothing else.
var instructionFiles = map[string]bool{
	"capture-intent": true,
	"plan-proof":     true,
	"read-evidence":  true,
}

// TestEveryPhaseNamesAnInstructionFile holds requirement R6 of SP-033. A
// removed skill name must never reach a reader.
func TestEveryPhaseNamesAnInstructionFile(t *testing.T) {
	results := []Result{
		needsPlan("SP-100", "no plan", "shipproof prove SP-100"),
	}
	for _, name := range []Phase{NoChange, IntentStale, NeedsPlan, NeedsRun,
		RunStale, RunFailed, NeedsEvidence, ReadyForHuman} {
		results = append(results, Result{ChangeID: "SP-100", Phase: name,
			NextInstruction: instructionFor(name)})
	}

	for _, result := range results {
		if result.NextInstruction == "" {
			t.Errorf("the phase %s names no instruction file", result.Phase)
			continue
		}
		if !instructionFiles[result.NextInstruction] {
			t.Errorf("the phase %s names %q, which is not one of the three files",
				result.Phase, result.NextInstruction)
		}
	}
}
