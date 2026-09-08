package cli

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alternayte/shipproof/internal/attest"
	"github.com/alternayte/shipproof/internal/schema"
)

func unsignedPack() schema.EvidencePack {
	return schema.EvidencePack{
		SchemaVersion: schema.CurrentVersion,
		ChangeID:      "SP-028",
		Verdict:       schema.VerdictEvidence{Verdict: "NOT PROVEN", Reason: "no proof ran.", Next: "run `shipproof prove SP-028`."},
		Intent:        schema.IntentEvidence{SnapshotHash: "abc123"},
		Checks:        []schema.Check{},
		EmptySections: map[string]string{
			"requirements": "none", "checks": "none", "implementation": "none",
			"unexplained_change": "none", "agent": "none", "attestation": "none",
		},
		Provenance: schema.PackProvenance{GeneratedAt: "2026-09-08T10:00:00Z", ShipProofVersion: "0.3.0-dev"},
	}
}

// signPack stands in for the cosign run that continuous integration performs.
func signPack(t *testing.T, pack schema.EvidencePack) schema.EvidencePack {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "shipproof-action"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := attest.Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	digest, err := attest.Digest(pack)
	if err != nil {
		t.Fatal(err)
	}
	pack.Attestation = &schema.AttestationEvidence{
		Format:      attest.Format,
		PayloadType: attest.PayloadType,
		Signature:   base64.StdEncoding.EncodeToString(signature),
		Certificate: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		Subject:     "1cceb33",
		Digest:      digest,
	}
	return pack
}

func writePackFile(t *testing.T, pack schema.EvidencePack) string {
	t.Helper()
	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "evidence-pack.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestVerifyReturnsZeroOnASignedPack is the first half of the proof for row P1
// of the definition of done.
func TestVerifyReturnsZeroOnASignedPack(t *testing.T) {
	path := writePackFile(t, signPack(t, unsignedPack()))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"pack", "--verify", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "SIGNATURE: VALID") {
		t.Fatalf("stdout does not state the result:\n%s", stdout.String())
	}
	// A tool never claims a check it did not run.
	body := strings.ToLower(stdout.String())
	for _, want := range []string{"fulcio", "rekor"} {
		if !strings.Contains(body, want) {
			t.Fatalf("stdout never names %q as unchecked:\n%s", want, stdout.String())
		}
	}
}

// TestVerifyReturnsOneOnAnAlteredPack is the second half of the proof for row
// P1 of the definition of done.
func TestVerifyReturnsOneOnAnAlteredPack(t *testing.T) {
	signed := signPack(t, unsignedPack())
	signed.Verdict.Verdict = "PROVEN"
	path := writePackFile(t, signed)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"pack", "--verify", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1; stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "SIGNATURE: NOT VALID") {
		t.Fatalf("stdout does not state the result:\n%s", stdout.String())
	}
}

func TestVerifyReturnsOneOnAnUnsignedPack(t *testing.T) {
	path := writePackFile(t, unsignedPack())

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"pack", "--verify", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	body := strings.ToLower(stdout.String() + stderr.String())
	if !strings.Contains(body, "unsigned") {
		t.Fatalf("the output never says that the pack is unsigned:\n%s%s", stdout.String(), stderr.String())
	}
}

// TestShipProofHoldsNoDependency holds requirement R9. The repository carries
// no third-party module, and the attestation work must not add one.
func TestShipProofHoldsNoDependency(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "require") {
		t.Fatalf("go.mod holds a require block:\n%s", data)
	}
}
