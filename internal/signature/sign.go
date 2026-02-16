package signature

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

const (
	// PrivateKeyEnvVar is the environment variable containing the base64-encoded private key
	// SECURITY: This should only be set in secure build/CI environments
	PrivateKeyEnvVar = "SAGEOX_SIGNING_KEY"

	// DefaultKeyID is the default key ID for new signatures
	DefaultKeyID = "sageox-dev-2025-01"
)

// Signer handles content signing for guidance files
type Signer struct {
	privateKey ed25519.PrivateKey
	keyID      string
}

// NewSigner creates a new Signer from environment variable
// Returns error if private key is not set or invalid
func NewSigner() (*Signer, error) {
	return NewSignerWithKeyID(DefaultKeyID)
}

// NewSignerWithKeyID creates a new Signer with a specific key ID
func NewSignerWithKeyID(keyID string) (*Signer, error) {
	keyBase64 := os.Getenv(PrivateKeyEnvVar)
	if keyBase64 == "" {
		return nil, fmt.Errorf("%s environment variable not set", PrivateKeyEnvVar)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d, want %d", len(keyBytes), ed25519.PrivateKeySize)
	}

	return &Signer{
		privateKey: ed25519.PrivateKey(keyBytes),
		keyID:      keyID,
	}, nil
}

// NewSignerFromKey creates a Signer from a raw private key (for testing)
func NewSignerFromKey(privateKey ed25519.PrivateKey, keyID string) *Signer {
	return &Signer{
		privateKey: privateKey,
		keyID:      keyID,
	}
}

// Sign signs the content and returns a base64-encoded signature
func (s *Signer) Sign(content []byte) string {
	sig := ed25519.Sign(s.privateKey, content)
	return base64.StdEncoding.EncodeToString(sig)
}

// KeyID returns the key ID used by this signer
func (s *Signer) KeyID() string {
	return s.keyID
}

// SignContent signs guidance content and returns content with embedded metadata
// The metadata includes org, team, project info along with the signature
func (s *Signer) SignContent(content []byte, org, team, project string) ([]byte, error) {
	// strip any existing metadata to get raw content
	contentWithoutMeta := StripMetadata(content)

	// normalize content to match what StripMetadata returns after WriteMetadata
	// WriteMetadata does: TrimRight + "\n\n" + metadata + "\n"
	// StripMetadata removes metadata but leaves the surrounding newlines
	// Result: TrimRight(content) + "\n\n" + "\n" = content with 3 trailing newlines
	normalizedContent := bytes.TrimRight(contentWithoutMeta, "\n")
	normalizedContent = append(normalizedContent, '\n', '\n', '\n')

	// sign the normalized content (this is what verification will check)
	signature := s.Sign(normalizedContent)

	// create metadata block
	meta := &MetadataBlock{
		Org:         org,
		Team:        team,
		Project:     project,
		UpdatedAt:   time.Now().UTC(),
		PublicKeyID: s.keyID,
		Signature:   signature,
	}

	// write metadata back to content
	return WriteMetadata(contentWithoutMeta, meta)
}

// SignFile reads a file, signs it, and optionally writes the signed version back
// If outputPath is empty, returns the signed content without writing
func (s *Signer) SignFile(inputPath string, org, team, project string) ([]byte, error) {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return s.SignContent(content, org, team, project)
}

// GenerateKeyPair generates a new Ed25519 key pair for signing
// Returns (publicKey, privateKey) as base64-encoded strings
// SECURITY: Private key should be stored securely, never committed to source
func GenerateKeyPair() (publicKeyBase64, privateKeyBase64 string, err error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	publicKeyBase64 = base64.StdEncoding.EncodeToString(publicKey)
	privateKeyBase64 = base64.StdEncoding.EncodeToString(privateKey)

	return publicKeyBase64, privateKeyBase64, nil
}
