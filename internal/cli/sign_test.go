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
	"testing"
	"time"

	"github.com/alternayte/shipproof/internal/attest"
	"github.com/alternayte/shipproof/internal/schema"
)

// signBlob stands in for `cosign sign-blob`. It signs the bytes it receives
// and returns the signature file and the certificate file.
func signBlob(t *testing.T, payload []byte) (string, string) {
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
	sum := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	signaturePath := filepath.Join(dir, "signature.b64")
	certificatePath := filepath.Join(dir, "certificate.pem")
	if err := os.WriteFile(signaturePath, []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		t.Fatal(err)
	}
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certificatePath, certificate, 0o644); err != nil {
		t.Fatal(err)
	}
	return signaturePath, certificatePath
}

// TestThePayloadRoundTripsThroughAttachAndVerify holds requirements R1 and R2.
// It runs the exact sequence that the action runs.
func TestThePayloadRoundTripsThroughAttachAndVerify(t *testing.T) {
	path := writePackFile(t, unsignedPack())

	// Step one. The action asks for the bytes that a signature must cover.
	var payload, stderr bytes.Buffer
	if code := Run([]string{"pack", "--payload", path}, &payload, &stderr); code != 0 {
		t.Fatalf("--payload exit code = %d, stderr = %s", code, stderr.String())
	}
	want, err := attest.Canonical(unsignedPack())
	if err != nil {
		t.Fatal(err)
	}
	if payload.String() != string(want) {
		t.Fatalf("--payload printed\n%s\nwant\n%s", payload.String(), want)
	}

	// Step two. Cosign signs those bytes.
	signaturePath, certificatePath := signBlob(t, payload.Bytes())

	// Step three. The action writes the block back into the pack.
	var stdout bytes.Buffer
	stderr.Reset()
	code := Run([]string{"pack", "--attach", path,
		"--signature", signaturePath, "--certificate", certificatePath, "--subject", "1cceb33"},
		&stdout, &stderr)
	if code != 0 {
		t.Fatalf("--attach exit code = %d, stderr = %s", code, stderr.String())
	}

	// Step four. The written pack verifies.
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pack", "--verify", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("--verify exit code = %d after attach; stdout = %s", code, stdout.String())
	}
}

// TestAttachRefusesAForeignSignature holds requirement R3. A block that
// `--verify` would reject must never reach the file.
func TestAttachRefusesAForeignSignature(t *testing.T) {
	path := writePackFile(t, unsignedPack())

	other := unsignedPack()
	other.ChangeID = "SP-999"
	foreign, err := attest.Canonical(other)
	if err != nil {
		t.Fatal(err)
	}
	signaturePath, certificatePath := signBlob(t, foreign)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pack", "--attach", path,
		"--signature", signaturePath, "--certificate", certificatePath, "--subject", "1cceb33"},
		&stdout, &stderr)
	if code == 0 {
		t.Fatal("--attach accepted a signature for a different pack")
	}

	// The file must still hold no attestation block.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var loaded schema.EvidencePack
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Attestation != nil {
		t.Fatal("--attach wrote a block that --verify would reject")
	}
}
