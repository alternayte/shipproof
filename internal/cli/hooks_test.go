package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// hookCommandPattern reads a documented hook command. Each example marks the
// command with a leading `#> ` marker inside its fenced block, so the test
// runs the exact string a hook author copies.
var hookCommandPattern = regexp.MustCompile(`(?m)^#> (shipproof .+)$`)

// hookTargets names the five harness targets of Section 8.1. The document must
// hold one example for each.
var hookTargets = []string{"Claude Code", "Cursor", "Codex", "OpenCode", "AGENTS.md"}

func readHooks(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "hooks.md"))
	if err != nil {
		t.Fatalf("read docs/hooks.md: %v", err)
	}
	return string(data)
}

// TestEveryHarnessHasAHookExample holds requirement R2.
func TestEveryHarnessHasAHookExample(t *testing.T) {
	document := readHooks(t)
	for _, target := range hookTargets {
		if !strings.Contains(document, target) {
			t.Errorf("docs/hooks.md holds no example for %s", target)
		}
	}
}

// TestEveryHookCommandRuns is the proof for row T3. The command is the
// contract, so the test runs every command the document names.
func TestEveryHookCommandRuns(t *testing.T) {
	matches := hookCommandPattern.FindAllStringSubmatch(readHooks(t), -1)
	if len(matches) < len(hookTargets) {
		t.Fatalf("docs/hooks.md marks %d commands, want at least %d", len(matches), len(hookTargets))
	}

	surface := map[string]bool{
		"init": true, "start": true, "prove": true, "pack": true, "status": true,
		"runner": true, "config": true, "run": true, "version": true,
	}

	for _, match := range matches {
		command := strings.TrimSpace(match[1])
		fields := strings.Fields(command)
		if len(fields) < 2 {
			t.Errorf("the command %q names no verb", command)
			continue
		}
		if !surface[fields[1]] {
			t.Errorf("the command %q names %q, which Section 4 does not hold", command, fields[1])
			continue
		}
		// Section 8.1 names the two commands a hook runs.
		if fields[1] != "prove" && fields[1] != "pack" {
			t.Errorf("the command %q is neither prove nor pack", command)
			continue
		}

		// Run the command in the state a hook finds: one open change with a
		// planned proof, in a real repository.
		root := newCLITestRoot(t)
		source := writeIntent(t, root, "intent.md",
			"# SP-700\n\n### SP-700-R1 — The tool answers\n\nThe command exits zero.\n")
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"start", "SP-700", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
			t.Fatalf("start: %s", stderr.String())
		}
		writePlanWithProof(t, root, "SP-700", "SP-700-R1", "true")
		initRepository(t, root)

		stdout.Reset()
		stderr.Reset()
		code := Run(fields[1:], &stdout, &stderr)

		// A documented hook command must run. Exit code 2 means the command
		// was used wrongly, and a document must never publish such a command.
		if code == 2 {
			t.Errorf("the command %q returned a usage error: %s%s", command, stdout.String(), stderr.String())
		}
		if code != 0 {
			t.Errorf("the command %q exited %d in a repository with one open change: %s%s",
				command, code, stdout.String(), stderr.String())
		}
	}
}

// TestTheDocumentStatesTheExitCodes holds requirement R5.
func TestTheDocumentStatesTheExitCodes(t *testing.T) {
	document := readHooks(t)
	for _, want := range []string{"exit code", "`0`", "`1`", "`2`"} {
		if !strings.Contains(document, want) {
			t.Errorf("docs/hooks.md does not state %s", want)
		}
	}
}

// TestNoExampleCallsALibrary holds requirement R7. Row T3 says the hook
// contract runs a command and never a library call.
func TestNoExampleCallsALibrary(t *testing.T) {
	document := readHooks(t)
	for _, forbidden := range []string{
		"github.com/alternayte/shipproof/internal",
		"import (",
		"pack.Assemble",
		"verdict.Decide",
	} {
		if strings.Contains(document, forbidden) {
			t.Errorf("docs/hooks.md holds the library reference %q", forbidden)
		}
	}
}

// TestTheDocumentStatesWhatItDoesNotVerify holds requirement R6. A document
// must never imply a guarantee that no proof supports.
func TestTheDocumentStatesWhatItDoesNotVerify(t *testing.T) {
	document := readHooks(t)
	if !strings.Contains(document, "ShipProof verifies the command, not the configuration format around it.") {
		t.Error("docs/hooks.md does not state the limit of its own guarantee")
	}
}

// TestThePreCommitScriptExists holds the promise the document makes. A
// document that names a file must name a file that exists.
func TestThePreCommitScriptExists(t *testing.T) {
	document := readHooks(t)
	if !strings.Contains(document, "scripts/pre-commit") {
		t.Skip("the document names no script")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "pre-commit"))
	if err != nil {
		t.Fatalf("docs/hooks.md names scripts/pre-commit and it does not exist: %v", err)
	}
	if !strings.Contains(string(data), "shipproof prove") {
		t.Error("scripts/pre-commit does not run shipproof prove")
	}
	info, err := os.Stat(filepath.Join("..", "..", "scripts", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("scripts/pre-commit is not executable")
	}
}

// TestThePreCommitScriptBlocksAFailedProof runs the documented script against
// a repository whose recorded run describes the working tree and holds a
// failed proof. The script must stop the commit.
func TestThePreCommitScriptBlocksAFailedProof(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	binary := buildBinary(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"),
		[]byte("version: 1\nverification:\n  command: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := writeIntent(t, root, "intent.md",
		"# SP-800\n\n### SP-800-R1 — The tool answers\n\nProse.\n")
	runBinary(t, binary, root, "start", "SP-800", "--intent", source, "--ceremony", "0")
	writePlanWithProof(t, root, "SP-800", "SP-800-R1", "false")
	initRepository(t, root)

	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", script)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+filepath.Dir(binary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("the script allowed a commit with a failed proof:\n%s", output)
	}
	if !strings.Contains(string(output), "FAILED") {
		t.Fatalf("the script did not report the verdict:\n%s", output)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "shipproof")
	command := exec.Command("go", "build", "-o", path, "./cmd/shipproof")
	command.Dir = filepath.Join("..", "..")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build shipproof: %v\n%s", err, out)
	}
	return path
}

func runBinary(t *testing.T, binary, root string, arguments ...string) {
	t.Helper()
	command := exec.Command(binary, arguments...)
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", arguments, err, out)
	}
}

// TestTheU1PacketScriptExists keeps the comprehension review runnable. The
// review document names the script, and a named file must exist.
func TestTheU1PacketScriptExists(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "u1-packet.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("scripts/u1-packet.sh does not exist: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("scripts/u1-packet.sh is not executable")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"report-a.html", "report-b.html", "report-c.html"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the script never writes %s", want)
		}
	}
}
