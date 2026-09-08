package cli

import (
	"os"
	"path/filepath"
	"regexp"
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

// TestNoActionTargetsNodeTwenty keeps the workflows off a deprecated runtime.
// GitHub forces a Node 20 action onto Node 24 today and will stop doing so.
var nodeTwentyMajors = map[string]string{
	"actions/checkout":             "v7",
	"actions/setup-go":             "v7",
	"actions/upload-artifact":      "v7",
	"goreleaser/goreleaser-action": "v7",
}

func TestNoActionTargetsNodeTwenty(t *testing.T) {
	files := []string{
		filepath.Join("..", "..", ".github", "workflows", "ci.yml"),
		filepath.Join("..", "..", ".github", "workflows", "release.yml"),
		filepath.Join("..", "..", ".github", "workflows", "evidence.yml"),
		filepath.Join("..", "..", "action", "action.yml"),
	}
	pattern := regexp.MustCompile(`uses:\s+([a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+)@(v\d+)`)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
			want, tracked := nodeTwentyMajors[match[1]]
			if !tracked {
				continue
			}
			if match[2] != want {
				t.Errorf("%s pins %s@%s, and %s runs on a supported runtime",
					filepath.Base(file), match[1], match[2], want)
			}
		}
	}
}

// TestCosignInstallerStaysOnV3 records a deliberate exclusion. Its v4 installs
// cosign v3, where `sign-blob` needs a --bundle flag and writes a bundle
// instead of a separate signature and certificate. That upgrade changes the
// signing flow and needs its own proof.
func TestCosignInstallerStaysOnV3(t *testing.T) {
	body := readAction(t)
	if !strings.Contains(body, "sigstore/cosign-installer@v3") {
		t.Fatal("the cosign installer moved without a change document")
	}
	if !strings.Contains(body, "--bundle flag") {
		t.Fatal("the action does not record why the installer stays on v3")
	}
}
