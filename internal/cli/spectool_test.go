package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/requirements"
)

// openSpecDocument is shaped like an OpenSpec change proposal.
const openSpecDocument = `# Add rate limiting

## Why

The API has no protection against a burst.

## What Changes

- MUST reject a request over the configured limit.
- MUST return a Retry-After header on a rejection.
`

// specKitDocument is shaped like a Spec Kit specification.
const specKitDocument = `# Feature Specification: Rate limiting

**Feature Branch**: ` + "`003-rate-limiting`" + `

## Requirements

### Functional Requirements

- **FR-001**: The system MUST reject a request over the configured limit.
- **FR-002**: The system SHALL return a Retry-After header.
`

func writeIntent(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestStartSnapshotsAnySpecToolDocument is the proof for row C1.
func TestStartSnapshotsAnySpecToolDocument(t *testing.T) {
	cases := []struct{ name, body string }{
		{"openspec.md", openSpecDocument},
		{"speckit.md", specKitDocument},
	}
	for _, testCase := range cases {
		root := newCLITestRoot(t)
		source := writeIntent(t, root, testCase.name, testCase.body)

		var stdout, stderr bytes.Buffer
		if code := Run([]string{"start", "SP-500", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
			t.Fatalf("%s: exit code = %d, stderr = %s", testCase.name, code, stderr.String())
		}

		// The snapshot hash must match the file that the user named.
		record, err := change.Load(root, "SP-500")
		if err != nil {
			t.Fatalf("%s: load record: %v", testCase.name, err)
		}
		sum := sha256.Sum256([]byte(testCase.body))
		if record.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("%s: snapshot hash = %q, want the hash of the file", testCase.name, record.SHA256)
		}

		// The general pattern must find the obligations and propose them.
		if !strings.Contains(stdout.String(), "proposed") {
			t.Fatalf("%s: start proposed no requirement:\n%s", testCase.name, stdout.String())
		}
		if !strings.Contains(stdout.String(), "--confirm-requirements") {
			t.Fatalf("%s: start never names the command that confirms:\n%s", testCase.name, stdout.String())
		}
	}
}

// TestAProposalIsNotAdopted holds the honesty gate of decision D3. A pattern
// match is a proposal, never a fact.
func TestAProposalIsNotAdopted(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "openspec.md", openSpecDocument)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-501", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if requirements.Exists(root, "SP-501") {
		t.Fatal("start adopted a proposal that no person confirmed")
	}
}

// TestConfirmRequirementsAdoptsTheProposal holds requirement R4.
func TestConfirmRequirementsAdoptsTheProposal(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "openspec.md", openSpecDocument)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-502", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("start: exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"start", "SP-502", "--confirm-requirements"}, &stdout, &stderr); code != 0 {
		t.Fatalf("confirm: exit code = %d, stderr = %s", code, stderr.String())
	}
	if !requirements.Exists(root, "SP-502") {
		t.Fatal("the confirmation adopted nothing")
	}

	set, err := requirements.Load(root, "SP-502")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Requirements) != 2 {
		t.Fatalf("the set holds %d requirements, want 2", len(set.Requirements))
	}
	for _, requirement := range set.Requirements {
		if requirement.ConfirmedAt == "" {
			t.Fatalf("requirement %s carries no confirmation time", requirement.ID)
		}
	}
}

// TestStartAdoptsANativeDocumentWithoutConfirmation holds requirement R1.
func TestStartAdoptsANativeDocumentWithoutConfirmation(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "native.md", "# SP-503\n\n### SP-503-R1 — The tool works\n\nProse.\n")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-503", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !requirements.Exists(root, "SP-503") {
		t.Fatalf("start adopted no native requirement:\n%s", stdout.String())
	}
}

// TestStartReportsADocumentWithNoRequirement holds requirement R5.
func TestStartReportsADocumentWithNoRequirement(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "prose.md", "# A title\n\nOnly prose lives here.\n")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-504", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if requirements.Exists(root, "SP-504") {
		t.Fatal("start adopted a requirement from a document that holds none")
	}
	if !strings.Contains(stdout.String(), "none") {
		t.Fatalf("start does not report the absence:\n%s", stdout.String())
	}
	if _, err := os.Stat(requirements.ProposalPath(root, "SP-504")); !os.IsNotExist(err) {
		t.Fatal("start wrote a proposal for a document that holds no requirement")
	}
}

// TestTheProposalIsReadable keeps the proposal inspectable. A person confirms
// what they can read.
func TestTheProposalIsReadable(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "openspec.md", openSpecDocument)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-505", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(requirements.ProposalPath(root, "SP-505"))
	if err != nil {
		t.Fatalf("read the proposal: %v", err)
	}
	var set requirements.Set
	if err := json.Unmarshal(data, &set); err != nil {
		t.Fatalf("the proposal is not readable: %v", err)
	}
	if len(set.Requirements) != 2 {
		t.Fatalf("the proposal holds %d requirements, want 2", len(set.Requirements))
	}
}
