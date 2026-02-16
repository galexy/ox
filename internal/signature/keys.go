package signature

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// mustDecodeBase64 decodes a base64 string or panics
// used for compile-time key embedding
func mustDecodeBase64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("failed to decode base64 key: %v", err))
	}
	return data
}

// SageoxPublicKeys contains the embedded Ed25519 public keys for signature verification.
// Keys are indexed by their key ID (e.g., "sageox-dev-2025-01").
//
// SECURITY: These keys are hardcoded rather than fetched dynamically to prevent
// key substitution attacks. An attacker who compromises the key distribution
// endpoint could not inject their own public key. The tradeoff is that key
// rotation requires a CLI update, but this is acceptable for signed guidance
// where the CLI version is already a trust boundary.
//
// Key rotation: Add new keys before deprecating old ones. The server can sign
// with multiple keys during transition periods.
var SageoxPublicKeys = map[string][]byte{
	"sageox-dev-2025-01":  mustDecodeBase64("DIpRfK3WCVDc4A27r3WbIf+RmWxnRalISUaT55i0hdE="),
	"sageox-test-2025-01": mustDecodeBase64("gJEcrysArAy0JmmdDoQEWAJxxL66N3QPRwZyQfdzX3Y="), // test key for unit tests
}

// GetPublicKeyByID returns the Ed25519 public key for the given key ID
func GetPublicKeyByID(keyID string) (ed25519.PublicKey, error) {
	keyBytes, ok := SageoxPublicKeys[keyID]
	if !ok {
		return nil, fmt.Errorf("unknown public key ID: %s", keyID)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size for %s: got %d, want %d", keyID, len(keyBytes), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(keyBytes), nil
}

// GetPublicKey returns the default Ed25519 public key for signature verification
// This returns the dev key for backwards compatibility
func GetPublicKey() (ed25519.PublicKey, error) {
	return GetPublicKeyByID("sageox-dev-2025-01")
}

// VerifySignatureWithKeyID verifies an Ed25519 signature using the specified key ID
func VerifySignatureWithKeyID(content, signature []byte, keyID string) bool {
	publicKey, err := GetPublicKeyByID(keyID)
	if err != nil {
		return false
	}

	return ed25519.Verify(publicKey, content, signature)
}

// VerifySignature verifies an Ed25519 signature against the provided content
// using the default public key (for backwards compatibility)
func VerifySignature(content, signature []byte) bool {
	publicKey, err := GetPublicKey()
	if err != nil {
		return false
	}

	return ed25519.Verify(publicKey, content, signature)
}
