package cli

import (
	"fmt"
	"io"
)

// removedMessages holds one line for each verb that SP-023 cut. The line
// names the replacement, or it states that the feature is gone.
var removedMessages = map[string]string{
	"doc":    "shipproof doc is gone. Document quality review is not a ShipProof feature.",
	"shape":  "shipproof shape is gone. Use your spec tool, then run `shipproof change start <change-id> --source <file>`.",
	"plan":   "shipproof plan is gone. Plan decomposition is not a ShipProof feature.",
	"linear": "shipproof linear is gone. ShipProof needs no issue tracker credential.",

	// SP-024 folded these verbs into the five commands of Section 4.
	"verification": "shipproof verification is gone. Run `shipproof prove <change-id>`.",
	"verify":       "shipproof verify is gone. Run `shipproof prove <change-id>`.",
	"harness":      "shipproof harness is gone. `shipproof init` installs the instructions.",
	"change":       "shipproof change is gone. Run `shipproof start <change-id> --intent <file>` or `shipproof status <change-id>`.",
	"next":         "shipproof next is gone. Run `shipproof status <change-id>`.",
	"coverage":     "shipproof coverage is gone. Run `shipproof status <change-id>`.",
	"skill":        "shipproof skill is gone. Skill validation is not a ShipProof feature.",
	"evidence":     "shipproof evidence is gone. Run `shipproof pack <change-id>`.",
	"review":       "shipproof review is gone. Run `shipproof pack <change-id>` and read the change report.",
	"telemetry":    "shipproof telemetry is gone. Run `shipproof pack <change-id> --adapter <claude|opencode>`.",
	"report":       "shipproof report is gone. `shipproof pack` writes the change report.",
}

// runRemoved prints the one line for a cut verb and returns exit code 2.
func runRemoved(verb string, stderr io.Writer) int {
	fmt.Fprintln(stderr, removedMessages[verb])
	return 2
}
