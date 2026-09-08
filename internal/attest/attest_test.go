package attest

import (
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
	"strings"
	"testing"
	"time"

	"github.com/alternayte/shipproof/internal/schema"
)

func samplePack() schema.EvidencePack {
	return schema.EvidencePack{
		SchemaVersion: schema.CurrentVersion,
		ChangeID:      "SP-028",
		Verdict:       schema.VerdictEvidence{Verdict: "NOT PROVEN", Reason: "no proof ran.", Next: "run `shipproof prove SP-028`."},
		Intent:        schema.IntentEvidence{SnapshotHash: "abc123"},
		Checks:        []schema.Check{},
		EmptySections: map[string]string{"x": "y"},
		Provenance:    schema.PackProvenance{GeneratedAt: "2026-09-08T10:00:00Z", ShipProofVersion: "0.3.0-dev"},
	}
}

// signFor returns a certificate and a signature over the canonical payload of
// the pack. It stands in for the cosign run that continuous integration
// performs.
func signFor(t *testing.T, pack schema.EvidencePack, subject string) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	payload, err := Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return string(certPEM), base64.StdEncoding.EncodeToString(signature)
}

// TestCanonicalExcludesTheSignatureBlock holds requirement R2.
func TestCanonicalExcludesTheSignatureBlock(t *testing.T) {
	pack := samplePack()
	before, err := Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}

	pack.Attestation = &schema.AttestationEvidence{Format: "in-toto", Signature: "MEUCIQ"}
	after, err := Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("the attestation block changed the canonical payload")
	}
	if strings.Contains(string(after), "attestation") {
		t.Fatalf("the canonical payload names the attestation block: %s", after)
	}
}

// TestCanonicalIsStable holds requirement R1.
func TestCanonicalIsStable(t *testing.T) {
	pack := samplePack()
	first, err := Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		again, err := Canonical(pack)
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != string(again) {
			t.Fatal("the canonical payload is not stable")
		}
	}
}

// TestTheSubjectNamesTheRevisionAndTheDigest holds requirement R3.
func TestTheSubjectNamesTheRevisionAndTheDigest(t *testing.T) {
	pack := samplePack()
	statement, err := Statement(pack, "1cceb33")
	if err != nil {
		t.Fatal(err)
	}
	if statement.Type == "" || statement.PredicateType == "" {
		t.Fatalf("the statement holds no type: %+v", statement)
	}
	if len(statement.Subject) != 1 {
		t.Fatalf("the statement holds %d subjects, want 1", len(statement.Subject))
	}
	if statement.Subject[0].Name != "1cceb33" {
		t.Fatalf("subject name = %q, want the head revision", statement.Subject[0].Name)
	}
	payload, err := Canonical(pack)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256Hex(payload)
	if statement.Subject[0].Digest["sha256"] != want {
		t.Fatalf("subject digest = %q, want %q", statement.Subject[0].Digest["sha256"], want)
	}
	if _, err := json.Marshal(statement); err != nil {
		t.Fatalf("the statement does not encode: %v", err)
	}
}

// TestVerifyAcceptsATrueSignature holds requirement R4.
func TestVerifyAcceptsATrueSignature(t *testing.T) {
	pack := samplePack()
	certificate, signature := signFor(t, pack, "https://github.com/alternayte/shipproof/.github/workflows/pack.yml@refs/heads/main")
	pack.Attestation = &schema.AttestationEvidence{
		Format:      "in-toto",
		PayloadType: "application/vnd.in-toto+json",
		Signature:   signature,
		Certificate: certificate,
		Subject:     "1cceb33",
	}

	result, err := Verify(pack)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.SignatureValid {
		t.Fatalf("the signature did not verify: %+v", result)
	}
	if result.CertificateSubject == "" {
		t.Fatal("Verify reported no certificate subject")
	}
	// R7. A tool never claims a check it did not run.
	if len(result.NotChecked) == 0 {
		t.Fatal("Verify named no unchecked item")
	}
	joined := strings.ToLower(strings.Join(result.NotChecked, " "))
	for _, want := range []string{"fulcio", "rekor"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("Verify never names %q as unchecked: %v", want, result.NotChecked)
		}
	}
}

// TestVerifyRejectsAnAlteredPack holds requirement R5.
func TestVerifyRejectsAnAlteredPack(t *testing.T) {
	pack := samplePack()
	certificate, signature := signFor(t, pack, "workflow")
	pack.Attestation = &schema.AttestationEvidence{
		Format: "in-toto", Signature: signature, Certificate: certificate, Subject: "1cceb33",
	}

	// One field changes after the signature.
	pack.Verdict.Verdict = "PROVEN"

	result, err := Verify(pack)
	if err == nil {
		t.Fatal("Verify accepted an altered pack")
	}
	if result.SignatureValid {
		t.Fatal("Verify reported a valid signature for an altered pack")
	}
}

// TestVerifyRejectsAnUnsignedPack holds requirement R6.
func TestVerifyRejectsAnUnsignedPack(t *testing.T) {
	result, err := Verify(samplePack())
	if err == nil {
		t.Fatal("Verify accepted an unsigned pack")
	}
	if result.SignatureValid {
		t.Fatal("Verify reported a valid signature for an unsigned pack")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsigned") {
		t.Fatalf("the error never says that the pack is unsigned: %v", err)
	}
}
