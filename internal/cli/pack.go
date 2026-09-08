package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alternayte/shipproof/internal/attest"
	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/evidence/pack"
	"github.com/alternayte/shipproof/internal/report"
	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/telemetry"
)

// runPack collects the agent telemetry, assembles the evidence pack, and
// writes the HTML change report. It replaces the old `evidence pack`,
// `telemetry collect`, and `report change` verbs.
func runPack(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof pack [change-id] [--base <rev>] [--head <rev>] [--adapter <claude|opencode>] [--output <path>]\n" +
		"       shipproof pack --verify <file>\n" +
		"       shipproof pack --payload <file>\n" +
		"       shipproof pack --comment <file>\n" +
		"       shipproof pack --attach <file> --signature <path> --certificate <path> --subject <rev>"

	// These options read one written pack. They touch no repository state, so
	// they run before every other option.
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--verify", "--payload", "--comment", "--attach":
			if index+1 >= len(args) {
				fmt.Fprintf(stderr, "%s requires a path\n", args[index])
				return 2
			}
			switch args[index] {
			case "--verify":
				return runPackVerify(args[index+1], stdout, stderr)
			case "--payload":
				return runPackPayload(args[index+1], stdout, stderr)
			case "--comment":
				return runPackComment(args[index+1], stdout, stderr)
			default:
				return runPackAttach(args[index+1], args, stdout, stderr)
			}
		}
	}

	changeID := ""
	adapter := ""
	output := ""
	opts := pack.Options{Warn: stderr}

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--base":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--base requires a revision")
				return 2
			}
			index++
			opts.BaseRev = args[index]
		case "--head":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--head requires a revision")
				return 2
			}
			index++
			opts.HeadRev = args[index]
		case "--adapter":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--adapter requires a name")
				return 2
			}
			index++
			adapter = args[index]
		case "--output":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--output requires a path")
				return 2
			}
			index++
			output = args[index]
		case "--evidence":
			for index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
				index++
				opts.EvidenceFiles = append(opts.EvidenceFiles, args[index])
			}
		default:
			if strings.HasPrefix(args[index], "--") || changeID != "" {
				fmt.Fprintln(stderr, usage)
				return 2
			}
			changeID = args[index]
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if changeID == "" {
		resolved, open, err := soleOpenChange(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			for _, candidate := range open {
				fmt.Fprintf(stderr, "  %s\n", candidate)
			}
			return 2
		}
		changeID = resolved
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	// Telemetry is optional. A missing adapter leaves the agent section
	// absent, and an absent section states that no record exists.
	if adapter != "" {
		if err := telemetry.Collect(root, changeID, adapter, ""); err != nil {
			fmt.Fprintf(stderr, "collect agent telemetry: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Agent record: .shipproof/runs/%s/agent-run.json\n", changeID)
	}

	assembled, err := pack.Assemble(root, changeID, opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := pack.WritePack(root, assembled); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	packPath := filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json")
	rel, _ := filepath.Rel(root, packPath)
	fmt.Fprintf(stdout, "Evidence pack: %s\n", filepath.ToSlash(rel))

	// Decision D2. A local pack stays unsigned, and the reader must learn that
	// from the run, not from the documentation.
	if assembled.Attestation == nil {
		fmt.Fprintln(stdout, attest.UnsignedNotice)
	}

	if output == "" {
		output = filepath.Join(root, ".shipproof", "changes", changeID, "report.html")
	}
	if code := writeChangeReport(root, changeID, output, stdout, stderr); code != 0 {
		return code
	}
	return 0
}

// writeChangeReport renders the HTML change report to the named path.
func writeChangeReport(root, changeID, output string, stdout, stderr io.Writer) int {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	file, err := os.Create(output)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer file.Close()

	if err := report.GenerateChangeReport(file, root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if rel, relErr := filepath.Rel(root, output); relErr == nil {
		fmt.Fprintf(stdout, "Change report: %s\n", filepath.ToSlash(rel))
	} else {
		fmt.Fprintf(stdout, "Change report: %s\n", output)
	}
	return 0
}

// runPackVerify checks the signature on one written pack. It prints the checks
// it ran and the checks it did not run. Section 10 rule 3 needs a plain
// result, and a plain result never hides its own limit.
func runPackVerify(path string, stdout, stderr io.Writer) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read the pack: %v\n", err)
		return 1
	}
	var loaded schema.EvidencePack
	if err := json.Unmarshal(data, &loaded); err != nil {
		fmt.Fprintf(stderr, "the file is not an evidence pack: %v\n", err)
		return 1
	}

	result, err := attest.Verify(loaded)
	if result.SignatureValid {
		fmt.Fprintln(stdout, "SIGNATURE: VALID")
	} else {
		fmt.Fprintln(stdout, "SIGNATURE: NOT VALID")
	}
	if err != nil {
		fmt.Fprintf(stdout, "reason    %v\n", err)
	}
	if result.Subject != "" {
		fmt.Fprintf(stdout, "subject   %s\n", result.Subject)
	}
	if result.Digest != "" {
		fmt.Fprintf(stdout, "digest    sha256:%s\n", result.Digest)
	}
	if result.CertificateSubject != "" {
		fmt.Fprintf(stdout, "signer    %s\n", result.CertificateSubject)
	}
	for _, item := range result.Checked {
		fmt.Fprintf(stdout, "checked   %s\n", item)
	}
	for _, item := range result.NotChecked {
		fmt.Fprintf(stdout, "unchecked %s\n", item)
	}
	if err != nil || !result.SignatureValid {
		return 1
	}
	return 0
}
