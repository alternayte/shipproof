// Package attest signs the evidence pack and verifies the signature.
//
// Section 10 of the design document sets four rules. The pack carries a
// detached signature. The signature covers a canonical serialization of the
// pack without the signature block. One command validates it. The subject is
// the head revision and the pack digest.
//
// Decision D1 sets the format: an in-toto statement with a SLSA provenance
// predicate. Decision D2 sets the key source: continuous integration signs
// with keyless Sigstore, and a local pack stays unsigned.
//
// This package verifies with the standard library alone. It checks the
// signature over the payload, and it reports the certificate subject. It does
// not check the Fulcio certificate chain, and it does not check the Rekor
// inclusion proof. Verify names both as not checked. A tool must never claim a
// check that it did not run.
package attest

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/alternayte/shipproof/internal/schema"
)

// StatementType and PredicateType name the in-toto format of decision D1.
const (
	StatementType = "https://in-toto.io/Statement/v1"
	PredicateType = "https://slsa.dev/provenance/v1"
	PayloadType   = "application/vnd.in-toto+json"
	Format        = "in-toto"
)

// ErrUnsigned reports a pack that carries no signature. A local run produces
// one, and that is the documented state, not a fault.
var ErrUnsigned = errors.New("the pack is unsigned. Only a build-system pack is audit-grade")

// UnsignedNotice is the one line that a local `pack` run prints.
const UnsignedNotice = "This pack is unsigned. Only a build-system pack is audit-grade."

// notChecked names every check that this package does not run. Verify reports
// it on every result, and the reader therefore learns the limit of the answer.
var notChecked = []string{
	"the Fulcio certificate chain",
	"the Rekor inclusion proof",
}

// Subject names what the attestation covers.
type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// InTotoStatement is the signed document of decision D1.
type InTotoStatement struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}

// Predicate holds the SLSA provenance fields that ShipProof can observe.
type Predicate struct {
	BuildDefinition BuildDefinition `json:"buildDefinition"`
	RunDetails      RunDetails      `json:"runDetails"`
}

type BuildDefinition struct {
	BuildType          string            `json:"buildType"`
	ExternalParameters map[string]string `json:"externalParameters"`
}

type RunDetails struct {
	Builder Builder `json:"builder"`
}

type Builder struct {
	ID string `json:"id"`
}

// Canonical serialises the pack without the attestation block. The signature
// covers these bytes. A signature over the whole pack cannot exist, because
// the pack holds the signature.
//
// The encoder sorts every map key, so the same pack always produces the same
// bytes. Rule 3 of Section 7 needs that.
func Canonical(pack schema.EvidencePack) ([]byte, error) {
	pack.Attestation = nil
	data, err := json.Marshal(pack)
	if err != nil {
		return nil, fmt.Errorf("encode the canonical payload: %w", err)
	}
	// Round-trip through a generic map. encoding/json sorts object keys on the
	// way out, and the round trip therefore removes any field-order difference
	// that a future struct change could introduce. Drop the attestation key
	// itself, so the payload never names the block that signs it.
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("decode the canonical payload: %w", err)
	}
	delete(value, "attestation")
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("re-encode the canonical payload: %w", err)
	}
	return canonical, nil
}

// Digest reports the SHA-256 of the canonical payload, in hexadecimal.
func Digest(pack schema.EvidencePack) (string, error) {
	payload, err := Canonical(pack)
	if err != nil {
		return "", err
	}
	return sha256Hex(payload), nil
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Statement builds the in-toto statement for one pack. The subject names the
// head revision and the digest of the canonical payload. Section 10 rule 4
// requires both.
func Statement(pack schema.EvidencePack, headRevision string) (InTotoStatement, error) {
	digest, err := Digest(pack)
	if err != nil {
		return InTotoStatement{}, err
	}
	return InTotoStatement{
		Type:          StatementType,
		Subject:       []Subject{{Name: headRevision, Digest: map[string]string{"sha256": digest}}},
		PredicateType: PredicateType,
		Predicate: Predicate{
			BuildDefinition: BuildDefinition{
				BuildType: "https://shipproof.dev/evidence-pack/v1",
				ExternalParameters: map[string]string{
					"change_id":     pack.ChangeID,
					"head_revision": headRevision,
				},
			},
			RunDetails: RunDetails{
				Builder: Builder{ID: "https://shipproof.dev/builder/" + pack.Provenance.ShipProofVersion},
			},
		},
	}, nil
}

// Result states what Verify checked and what it did not check.
type Result struct {
	SignatureValid     bool     `json:"signature_valid"`
	Subject            string   `json:"subject,omitempty"`
	Digest             string   `json:"digest,omitempty"`
	CertificateSubject string   `json:"certificate_subject,omitempty"`
	Checked            []string `json:"checked"`
	NotChecked         []string `json:"not_checked"`
}

// Verify checks the signature over the canonical payload. It returns an error
// for an unsigned pack and for an altered pack. The result names the limits of
// the answer in both cases.
func Verify(pack schema.EvidencePack) (Result, error) {
	result := Result{NotChecked: append([]string{}, notChecked...), Checked: []string{}}

	block := pack.Attestation
	if block == nil || block.Signature == "" {
		return result, ErrUnsigned
	}

	payload, err := Canonical(pack)
	if err != nil {
		return result, err
	}
	result.Digest = sha256Hex(payload)
	result.Subject = block.Subject

	if block.Certificate == "" {
		return result, errors.New("the pack carries a signature and no certificate, so nothing can check it")
	}
	certificate, err := parseCertificate(block.Certificate)
	if err != nil {
		return result, err
	}
	result.CertificateSubject = certificateSubject(certificate)

	// A signer wraps base64 output in newlines. Remove every space first, the
	// same way the certificate is read.
	signature, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(block.Signature), ""))
	if err != nil {
		return result, fmt.Errorf("decode the signature: %w", err)
	}

	sum := sha256.Sum256(payload)
	if err := checkSignature(certificate.PublicKey, sum[:], payload, signature); err != nil {
		return result, err
	}

	result.SignatureValid = true
	result.Checked = []string{
		"the signature over the canonical payload",
		"the digest of the canonical payload",
	}

	// A recorded digest that disagrees with the payload is a defect, even when
	// the signature holds. Report it rather than hide it.
	if block.Digest != "" && block.Digest != result.Digest {
		result.SignatureValid = false
		return result, fmt.Errorf("the recorded digest %q does not match the canonical payload digest %q",
			block.Digest, result.Digest)
	}
	return result, nil
}

// parseCertificate reads the certificate in every encoding that a signer
// produces. `cosign sign-blob` base64-encodes its output by default, so the
// certificate arrives as base64 around the PEM rather than as the PEM itself.
// A verifier that reads only raw PEM rejects a true signature.
//
// The order is: raw PEM, then base64 of a PEM, then base64 of the raw DER.
func parseCertificate(text string) (*x509.Certificate, error) {
	if block, _ := pem.Decode([]byte(text)); block != nil {
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse the certificate: %w", err)
		}
		return certificate, nil
	}

	// A signer wraps base64 output in newlines. Remove every space first.
	compact := strings.Join(strings.Fields(text), "")
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, errors.New("the certificate is neither PEM nor base64")
	}
	if block, _ := pem.Decode(decoded); block != nil {
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse the certificate: %w", err)
		}
		return certificate, nil
	}
	certificate, err := x509.ParseCertificate(decoded)
	if err != nil {
		return nil, fmt.Errorf("parse the certificate: %w", err)
	}
	return certificate, nil
}

// certificateSubject names the workload that signed. A keyless Sigstore
// certificate carries the workflow identity in a subject alternative name. A
// certificate with none falls back to the common name.
func certificateSubject(certificate *x509.Certificate) string {
	for _, uri := range certificate.URIs {
		return uri.String()
	}
	for _, name := range certificate.DNSNames {
		return name
	}
	for _, address := range certificate.EmailAddresses {
		return address
	}
	return certificate.Subject.CommonName
}

func checkSignature(key any, digest, payload, signature []byte) error {
	switch typed := key.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(typed, digest, signature) {
			return errors.New("the signature does not match the pack. The pack changed after the signature")
		}
		return nil
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(typed, crypto.SHA256, digest, signature); err != nil {
			return errors.New("the signature does not match the pack. The pack changed after the signature")
		}
		return nil
	case ed25519.PublicKey:
		if !ed25519.Verify(typed, payload, signature) {
			return errors.New("the signature does not match the pack. The pack changed after the signature")
		}
		return nil
	default:
		return fmt.Errorf("the certificate holds a key of type %T, which this build cannot check", key)
	}
}
