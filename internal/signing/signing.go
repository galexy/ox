// Package signing provides Ed25519 cryptographic signing and verification
// for embedded artifacts in the ox CLI.
//
// This is used to ensure that compiled-in data (redaction policies, guidance,
// configuration schemas, etc.) hasn't been tampered with after release.
//
// # Architecture
//
// During release:
//  1. Generate manifests for each artifact type
//  2. Sign each manifest with the release private key
//  3. Embed public key + signatures via ldflags
//
// At runtime:
//  1. Regenerate manifest from embedded data
//  2. Verify signature against embedded public key
//  3. Report status to user
//
// # Artifact Types
//
// Each artifact type (redaction, guidance, etc.) registers itself with a
// manifest generator function. The signing tool iterates all registered
// artifacts and generates signatures for each.
package signing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

// These variables are set at build time via ldflags.
// During development, they remain empty (signature verification disabled).
// During release, they are populated by the signing process.
//
// The format is: artifact_name -> base64-encoded value
// Set via: -X 'github.com/sageox/ox/internal/signing.EmbeddedPublicKey=...'
var (
	// EmbeddedPublicKey is the base64-encoded Ed25519 public key used for all artifacts.
	// All artifacts share the same key pair for simplicity.
	EmbeddedPublicKey string

	// EmbeddedSignatures is a semicolon-separated list of artifact:signature pairs.
	// Format: "redaction:sig1;guidance:sig2;..."
	// Each signature is base64-encoded.
	EmbeddedSignatures string
)

// VerificationStatus represents the result of signature verification.
type VerificationStatus int

const (
	// StatusNotConfigured means no signature was embedded (dev build).
	StatusNotConfigured VerificationStatus = iota
	// StatusValid means the signature verified successfully.
	StatusValid
	// StatusInvalid means the signature did not match (possible tampering).
	StatusInvalid
	// StatusError means an error occurred during verification.
	StatusError
	// StatusMissing means the artifact has no embedded signature.
	StatusMissing
)

func (s VerificationStatus) String() string {
	switch s {
	case StatusNotConfigured:
		return "not_configured"
	case StatusValid:
		return "valid"
	case StatusInvalid:
		return "invalid"
	case StatusError:
		return "error"
	case StatusMissing:
		return "missing"
	default:
		return "unknown"
	}
}

// VerificationResult contains the full result of signature verification.
type VerificationResult struct {
	Artifact  string
	Status    VerificationStatus
	Hash      string // hex-encoded hash of the artifact
	PublicKey string // base64-encoded public key (if configured)
	Signature string // base64-encoded signature (if found)
	Error     error
}

// ManifestGenerator generates the canonical bytes of an artifact for signing.
// The returned bytes must be deterministic for the same artifact content.
type ManifestGenerator func() ([]byte, error)

// ArtifactRegistry holds registered artifacts and their manifest generators.
type ArtifactRegistry struct {
	mu        sync.RWMutex
	artifacts map[string]ManifestGenerator
}

// global registry for artifacts
var registry = &ArtifactRegistry{
	artifacts: make(map[string]ManifestGenerator),
}

// RegisterArtifact registers a manifest generator for an artifact.
// Call this from init() in each artifact package.
func RegisterArtifact(name string, generator ManifestGenerator) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.artifacts[name] = generator
}

// ListArtifacts returns the names of all registered artifacts.
func ListArtifacts() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.artifacts))
	for name := range registry.artifacts {
		names = append(names, name)
	}
	return names
}

// IsSigned returns true if the binary was built with signing enabled.
func IsSigned() bool {
	return EmbeddedPublicKey != "" && EmbeddedSignatures != ""
}

// Verify checks the signature of a specific artifact.
func Verify(artifactName string) *VerificationResult {
	result := &VerificationResult{
		Artifact:  artifactName,
		PublicKey: EmbeddedPublicKey,
	}

	// get the manifest generator
	registry.mu.RLock()
	generator, exists := registry.artifacts[artifactName]
	registry.mu.RUnlock()

	if !exists {
		result.Status = StatusError
		result.Error = fmt.Errorf("unknown artifact: %s", artifactName)
		return result
	}

	// generate manifest
	manifest, err := generator()
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Errorf("failed to generate manifest: %w", err)
		return result
	}

	// compute hash (always compute, even if not signed)
	hash := sha256.Sum256(manifest)
	result.Hash = hex.EncodeToString(hash[:])

	// check if signing is configured
	if !IsSigned() {
		result.Status = StatusNotConfigured
		return result
	}

	// find signature for this artifact
	signature := findSignature(artifactName)
	if signature == "" {
		result.Status = StatusMissing
		result.Error = fmt.Errorf("no signature found for artifact: %s", artifactName)
		return result
	}
	result.Signature = signature

	// decode public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(EmbeddedPublicKey)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Errorf("failed to decode public key: %w", err)
		return result
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		result.Status = StatusError
		result.Error = fmt.Errorf("invalid public key size: got %d, want %d",
			len(pubKeyBytes), ed25519.PublicKeySize)
		return result
	}

	// decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Errorf("failed to decode signature: %w", err)
		return result
	}

	if len(sigBytes) != ed25519.SignatureSize {
		result.Status = StatusError
		result.Error = fmt.Errorf("invalid signature size: got %d, want %d",
			len(sigBytes), ed25519.SignatureSize)
		return result
	}

	// verify signature (sign the hash, not the full manifest)
	if ed25519.Verify(pubKeyBytes, hash[:], sigBytes) {
		result.Status = StatusValid
	} else {
		result.Status = StatusInvalid
		result.Error = errors.New("signature verification failed: artifact may have been tampered with")
	}

	return result
}

// VerifyAll checks signatures of all registered artifacts.
func VerifyAll() map[string]*VerificationResult {
	results := make(map[string]*VerificationResult)
	for _, name := range ListArtifacts() {
		results[name] = Verify(name)
	}
	return results
}

// findSignature extracts the signature for a specific artifact from EmbeddedSignatures.
func findSignature(artifactName string) string {
	if EmbeddedSignatures == "" {
		return ""
	}

	// parse "artifact1:sig1;artifact2:sig2;..."
	// simple parsing - no semicolons or colons allowed in artifact names
	start := 0
	for start < len(EmbeddedSignatures) {
		// find the next semicolon or end
		end := start
		for end < len(EmbeddedSignatures) && EmbeddedSignatures[end] != ';' {
			end++
		}

		// parse "name:signature"
		pair := EmbeddedSignatures[start:end]
		colonIdx := -1
		for i := 0; i < len(pair); i++ {
			if pair[i] == ':' {
				colonIdx = i
				break
			}
		}

		if colonIdx > 0 && colonIdx < len(pair)-1 {
			name := pair[:colonIdx]
			sig := pair[colonIdx+1:]
			if name == artifactName {
				return sig
			}
		}

		start = end + 1
	}

	return ""
}

// Sign signs the given data with the private key and returns the signature.
// This is used by the signing tool during release.
func Sign(privateKey ed25519.PrivateKey, data []byte) []byte {
	hash := sha256.Sum256(data)
	return ed25519.Sign(privateKey, hash[:])
}

// GenerateKeyPair creates a new Ed25519 key pair.
// This is used by the key generation tool.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

// Hash computes the SHA-256 hash of data and returns it as hex.
func Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GetManifestBytes returns the canonical manifest bytes for an artifact.
// This is used by the signing tool to get the bytes to sign.
func GetManifestBytes(artifactName string) ([]byte, error) {
	registry.mu.RLock()
	generator, exists := registry.artifacts[artifactName]
	registry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown artifact: %s", artifactName)
	}

	return generator()
}
