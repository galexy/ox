package signature

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
)

// VerifyFile verifies the signature of a SAGEOX.md file
// Uses caching to avoid repeated verification
func VerifyFile(filePath string) (*VerificationResult, error) {
	// read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// compute SHA-256 hash
	hash := ComputeHash(content)

	// check cache first
	cachedEntry, err := GetCachedResult(filePath, hash)
	if err == nil && cachedEntry != nil {
		// cache hit - return cached result
		result := &VerificationResult{
			Verified:    cachedEntry.Verified,
			HasMetadata: true, // if in cache, metadata was present
		}
		return result, nil
	}

	// cache miss - perform full verification
	metadata, err := ParseMetadata(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// no metadata block found - file is unsigned
	if metadata == nil {
		result := &VerificationResult{
			Verified:      false,
			HasMetadata:   false,
			HasSignature:  false,
			SignatureInfo: "no signature (file has no metadata block)",
		}
		// cache this result
		_ = SetCachedResult(filePath, hash, false)
		return result, nil
	}

	// check if signature is present in metadata
	if metadata.Signature == "" {
		result := &VerificationResult{
			Verified:      false,
			HasMetadata:   true,
			HasSignature:  false,
			Metadata:      metadata,
			Error:         "metadata found but signature field is empty",
			SignatureInfo: "no signature (metadata present but signature field empty)",
		}
		_ = SetCachedResult(filePath, hash, false)
		return result, nil
	}

	// decode signature from base64
	sigBytes, err := base64.StdEncoding.DecodeString(metadata.Signature)
	if err != nil {
		result := &VerificationResult{
			Verified:      false,
			HasMetadata:   true,
			HasSignature:  true,
			Metadata:      metadata,
			Error:         fmt.Sprintf("failed to decode signature: %v", err),
			SignatureInfo: "invalid signature (base64 decode failed)",
		}
		_ = SetCachedResult(filePath, hash, false)
		return result, nil
	}

	// check for public key ID
	keyID := metadata.PublicKeyID
	if keyID == "" {
		// fallback to default key for backwards compatibility
		keyID = "sageox-dev-2025-01"
	}

	// strip metadata to get content for verification
	contentWithoutMetadata := StripMetadata(content)

	// verify signature using the key ID from metadata
	verified := VerifySignatureWithKeyID(contentWithoutMetadata, sigBytes, keyID)

	var signatureInfo string
	if verified {
		signatureInfo = fmt.Sprintf("valid (key: %s)", keyID)
	} else {
		signatureInfo = fmt.Sprintf("INVALID (key: %s) - content may have been tampered with", keyID)
	}

	result := &VerificationResult{
		Verified:      verified,
		HasMetadata:   true,
		HasSignature:  true,
		KeyID:         keyID,
		Metadata:      metadata,
		SignatureInfo: signatureInfo,
	}

	// cache the result
	_ = SetCachedResult(filePath, hash, verified)

	return result, nil
}

// QuickCheck performs fast verification check
// Returns true if file is verified (from cache or fresh check)
// Returns false if unsigned or invalid - caller should show warning
func QuickCheck(filePath string) bool {
	result, err := VerifyFile(filePath)
	if err != nil {
		return false
	}
	return result.Verified
}

// ComputeHash computes SHA-256 hash of file content
func ComputeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return base64.StdEncoding.EncodeToString(hash[:])
}
