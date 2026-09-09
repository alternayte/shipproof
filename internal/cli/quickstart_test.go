package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readREADME(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestTheREADMENamesNoRemovedConcept holds requirement R3. A newcomer must not
// learn a vocabulary that the product dropped.
func TestTheREADMENamesNoRemovedConcept(t *testing.T) {
	document := readREADME(t)
	for _, removed := range []string{
		// The readiness states and the finding classes of SP-023.
		"SHAPING", "READY_WITH_ASSUMPTIONS",
		"Finding classes", "readiness state",
		// The grade words that SP-026 replaced. `BLOCKED` is not here: it is
		// a live `shipproof run` outcome, not a removed readiness state.
		"`derived`", "`inferred`", "agent-inferred",
		"derived, inferred",
		// The skill catalog of SP-033. The instruction files replaced it, so
		// no sentence may still send a reader to a skill.
		"Agent Skills",
		"the skill that",
		"skill that handles",
	} {
		if strings.Contains(document, removed) {
			t.Errorf("README.md still names the removed concept %q", removed)
		}
	}
	// It must name what replaced them.
	for _, want := range []string{"`observed`", "`stated`", "`claimed`", "PROVEN"} {
		if !strings.Contains(document, want) {
			t.Errorf("README.md never names %q", want)
		}
	}
}

// TestTheREADMEOffersAnInstallWithoutGo holds requirements R1 and R2.
func TestTheREADMEOffersAnInstallWithoutGo(t *testing.T) {
	document := readREADME(t)
	for _, want := range []string{
		"releases/latest/download",
		"shipproof_",
		"go install",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("README.md never names %q", want)
		}
	}
}

// TestInitWritesARunnableGate holds requirements R4 and R5. The first verdict a
// newcomer sees must never read FAILED because of a build tool they do not use.
func TestInitWritesARunnableGate(t *testing.T) {
	root := t.TempDir()
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("init: exit code = %d, stderr = %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	// A bare directory holds no justfile, so the gate must not name `just`.
	if strings.Contains(body, "just verify") {
		t.Fatalf("init wrote a gate the repository cannot run:\n%s", body)
	}
	if !strings.Contains(stdout.String(), "Gate:") {
		t.Fatalf("init did not report the gate it chose:\n%s", stdout.String())
	}
}

// TestInitKeepsAGateTheRepositoryCanRun asserts that a repository which does
// hold a justfile still gets the command it already uses.
func TestInitKeepsAGateTheRepositoryCanRun(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "justfile"), []byte("verify:\n\techo ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("init: exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "just verify") {
		t.Fatalf("init ignored the justfile the repository holds:\n%s", data)
	}
}

// TestTheDownloadURLMatchesTheReleaseAssets binds the README to the release
// configuration. A documented install command that returns 404 is worse than
// no command, and the two files drift silently without this test.
func TestTheDownloadURLMatchesTheReleaseAssets(t *testing.T) {
	config, err := os.ReadFile(filepath.Join("..", "..", ".goreleaser.yml"))
	if err != nil {
		t.Fatal(err)
	}
	// `releases/latest/download/<name>` resolves only when the asset name is
	// the same in every release. A version in the template breaks it.
	template := string(config)
	start := strings.Index(template, "name_template:")
	if start < 0 {
		t.Fatal(".goreleaser.yml holds no archive name template")
	}
	end := strings.Index(template[start:], "checksum:")
	if end < 0 {
		end = len(template) - start
	}
	archive := template[start : start+end]
	if strings.Contains(archive, ".Version") {
		t.Fatal("the archive name holds the version, so the README download URL cannot resolve")
	}

	readme := readREADME(t)
	for _, want := range []string{
		"releases/latest/download/shipproof_darwin_arm64.tar.gz",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md does not name the asset %q", want)
		}
	}
	// Every platform the README names must be one the release builds.
	for _, platform := range []string{"darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64"} {
		if !strings.Contains(readme, platform) {
			t.Errorf("README.md never names the platform %q", platform)
		}
	}
}

// readmeCommandPattern reads every ShipProof command the README shows.
var readmeCommandPattern = regexp.MustCompile(`shipproof ([a-z-]+)`)

// TestTheREADMENamesOnlyRealCommands binds the README to the surface. A
// quickstart that names a command the tool does not have is worse than no
// quickstart.
func TestTheREADMENamesOnlyRealCommands(t *testing.T) {
	surface := map[string]bool{
		"init": true, "start": true, "prove": true, "pack": true, "status": true,
		"runner": true, "config": true, "run": true, "version": true,
	}
	for _, match := range readmeCommandPattern.FindAllStringSubmatch(readREADME(t), -1) {
		if !surface[match[1]] {
			t.Errorf("README.md names the command %q, which Section 4 does not hold", match[1])
		}
	}
}

// TestTheREADMEDocumentsTheOngoingLoop holds what the second reader review
// asked for. A quickstart that stops after the first run leaves the reader
// with no answer to "how do I add and track a requirement".
func TestTheREADMEDocumentsTheOngoingLoop(t *testing.T) {
	readme := readREADME(t)
	for _, want := range []struct{ name, probe string }{
		{"editing a proposal before confirming", "requirements-proposal.json"},
		{"merging when the document changes", "--force"},
		{"that a merge never deletes", "never deletes"},
		{"the list of requirements needing a proof", "needs a proof"},
		{"a proof whose program is missing", "broken proof"},
	} {
		if !strings.Contains(readme, want.probe) {
			t.Errorf("README.md does not document %s (looked for %q)", want.name, want.probe)
		}
	}
}
