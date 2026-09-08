package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alternayte/shipproof/internal/attest"
	"github.com/alternayte/shipproof/internal/schema"
)

// loadPackFile reads one written evidence pack. Every option that acts on a
// file uses it, so one malformed file produces one message.
func loadPackFile(path string, stderr io.Writer) (schema.EvidencePack, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read the pack: %v\n", err)
		return schema.EvidencePack{}, false
	}
	var loaded schema.EvidencePack
	if err := json.Unmarshal(data, &loaded); err != nil {
		fmt.Fprintf(stderr, "the file is not an evidence pack: %v\n", err)
		return schema.EvidencePack{}, false
	}
	return loaded, true
}

// runPackPayload prints the canonical payload. A signature covers these exact
// bytes and nothing else. The build system pipes this output into cosign.
func runPackPayload(path string, stdout, stderr io.Writer) int {
	loaded, ok := loadPackFile(path, stderr)
	if !ok {
		return 1
	}
	payload, err := attest.Canonical(loaded)
	if err != nil {
		fmt.Fprintf(stderr, "build the canonical payload: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(payload); err != nil {
		fmt.Fprintf(stderr, "write the payload: %v\n", err)
		return 1
	}
	return 0
}

// runPackAttach writes the signature block into a pack. It checks the
// signature before it writes. A block that `--verify` would reject must never
// reach the file.
func runPackAttach(path string, args []string, stdout, stderr io.Writer) int {
	signaturePath := optionValue(args, "--signature")
	certificatePath := optionValue(args, "--certificate")
	subject := optionValue(args, "--subject")

	if signaturePath == "" || certificatePath == "" {
		fmt.Fprintln(stderr, "--attach requires --signature and --certificate")
		return 2
	}

	loaded, ok := loadPackFile(path, stderr)
	if !ok {
		return 1
	}
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		fmt.Fprintf(stderr, "read the signature: %v\n", err)
		return 1
	}
	certificate, err := os.ReadFile(certificatePath)
	if err != nil {
		fmt.Fprintf(stderr, "read the certificate: %v\n", err)
		return 1
	}
	digest, err := attest.Digest(loaded)
	if err != nil {
		fmt.Fprintf(stderr, "build the digest: %v\n", err)
		return 1
	}

	loaded.Attestation = &schema.AttestationEvidence{
		Format:      attest.Format,
		PayloadType: attest.PayloadType,
		Signature:   strings.TrimSpace(string(signature)),
		Certificate: string(certificate),
		Subject:     subject,
		Digest:      digest,
	}

	// The section is no longer empty, so its reason must go. A signed pack
	// that still calls its attestation empty tells a reader the opposite of
	// the truth.
	delete(loaded.EmptySections, "attestation")

	// Check before write. A pack must never carry a block that fails.
	if _, err := attest.Verify(loaded); err != nil {
		fmt.Fprintf(stderr, "the signature does not match this pack, so nothing was written: %v\n", err)
		return 1
	}

	data, err := json.MarshalIndent(loaded, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "encode the pack: %v\n", err)
		return 1
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "write the pack: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Signed: %s\n", path)
	fmt.Fprintf(stdout, "subject   %s\n", subject)
	fmt.Fprintf(stdout, "digest    sha256:%s\n", digest)
	return 0
}

// optionValue reads the value that follows one option name. It reports an
// empty string when the option is absent.
func optionValue(args []string, name string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == name {
			return args[index+1]
		}
	}
	return ""
}
