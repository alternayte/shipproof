package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readAction(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "action", "action.yml"))
	if err != nil {
		t.Fatalf("read the action: %v", err)
	}
	return string(data)
}

// TestTheActionPerformsTheFourSteps holds requirement R10. Section 9.1 names
// the four steps in order.
func TestTheActionPerformsTheFourSteps(t *testing.T) {
	body := readAction(t)

	steps := []struct{ name, marker string }{
		{"install the binary", "go build -o \"$RUNNER_TEMP/shipproof\""},
		{"run pack over the range", "shipproof pack $CHANGE --base"},
		{"upload the pack", "actions/upload-artifact"},
		{"post one comment", "gh pr comment"},
	}
	position := 0
	for _, step := range steps {
		index := strings.Index(body[position:], step.marker)
		if index < 0 {
			t.Fatalf("the action never performs the step %q", step.name)
		}
		position += index
	}
}

// TestTheActionSignsBeforeItUploads holds the rule that a pack which does not
// verify must never reach the artifact store.
func TestTheActionSignsBeforeItUploads(t *testing.T) {
	body := readAction(t)

	for _, marker := range []string{
		"shipproof pack --payload",
		"cosign sign-blob",
		"shipproof pack --attach",
		"shipproof pack --verify",
		"actions/upload-artifact",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("the action never runs %q", marker)
		}
	}
	if strings.Index(body, "shipproof pack --verify") > strings.Index(body, "actions/upload-artifact") {
		t.Fatal("the action uploads the pack before it verifies the signature")
	}
}

// TestTheActionNamesNoCutCommand keeps the reduction of SP-023 true in the
// workflow files too.
func TestTheActionNamesNoCutCommand(t *testing.T) {
	body := readAction(t)
	for _, word := range cutReferences {
		if strings.Contains(body, word) {
			t.Fatalf("the action names the removed reference %q", word)
		}
	}
}
