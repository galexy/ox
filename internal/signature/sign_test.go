package signature

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	pubKey, privKey, err := GenerateKeyPair()
	require.NoError(t, err, "GenerateKeyPair() error")

	// verify public key decodes correctly
	pubBytes, err := base64.StdEncoding.DecodeString(pubKey)
	require.NoError(t, err, "failed to decode public key")
	assert.Len(t, pubBytes, ed25519.PublicKeySize, "public key wrong size")

	// verify private key decodes correctly
	privBytes, err := base64.StdEncoding.DecodeString(privKey)
	require.NoError(t, err, "failed to decode private key")
	assert.Len(t, privBytes, ed25519.PrivateKeySize, "private key wrong size")
}

func TestNewSigner_MissingEnvVar(t *testing.T) {
	// ensure env var is not set
	os.Unsetenv(PrivateKeyEnvVar)

	_, err := NewSigner()
	require.Error(t, err, "NewSigner() expected error when env var not set")
	assert.Contains(t, err.Error(), "not set", "error message should mention 'not set'")
}

func TestNewSigner_InvalidBase64(t *testing.T) {
	t.Setenv(PrivateKeyEnvVar, "not-valid-base64!!!")

	_, err := NewSigner()
	assert.Error(t, err, "NewSigner() expected error for invalid base64")
}

func TestNewSigner_InvalidKeySize(t *testing.T) {
	// set a valid base64 but wrong size key
	t.Setenv(PrivateKeyEnvVar, base64.StdEncoding.EncodeToString([]byte("too-short")))

	_, err := NewSigner()
	require.Error(t, err, "NewSigner() expected error for wrong key size")
	assert.Contains(t, err.Error(), "invalid private key size", "error message should mention key size")
}

func TestNewSigner_Valid(t *testing.T) {
	// generate a valid key pair
	_, privKey, err := GenerateKeyPair()
	require.NoError(t, err, "failed to generate key pair")

	t.Setenv(PrivateKeyEnvVar, privKey)

	signer, err := NewSigner()
	require.NoError(t, err, "NewSigner() error")

	assert.Equal(t, DefaultKeyID, signer.KeyID())
}

func TestSigner_Sign(t *testing.T) {
	// generate key pair
	pubKeyBase64, privKeyBase64, err := GenerateKeyPair()
	require.NoError(t, err, "failed to generate key pair")

	privKeyBytes, _ := base64.StdEncoding.DecodeString(privKeyBase64)
	pubKeyBytes, _ := base64.StdEncoding.DecodeString(pubKeyBase64)

	signer := NewSignerFromKey(ed25519.PrivateKey(privKeyBytes), "test-key")

	content := []byte("test content to sign")
	signature := signer.Sign(content)

	// verify signature is valid base64
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	require.NoError(t, err, "signature is not valid base64")

	// verify signature with public key
	assert.True(t, ed25519.Verify(ed25519.PublicKey(pubKeyBytes), content, sigBytes), "signature verification failed")
}

func TestSigner_SignContent(t *testing.T) {
	// generate key pair
	pubKeyBase64, privKeyBase64, err := GenerateKeyPair()
	require.NoError(t, err, "failed to generate key pair")

	privKeyBytes, _ := base64.StdEncoding.DecodeString(privKeyBase64)
	signer := NewSignerFromKey(ed25519.PrivateKey(privKeyBytes), "test-key")

	content := []byte("# Guidance\n\nSome guidance content here.\n")
	signedContent, err := signer.SignContent(content, "acme", "infra", "main")
	require.NoError(t, err, "SignContent() error")

	// verify metadata was added
	assert.True(t, HasMetadata(signedContent), "signed content should have metadata")

	// parse metadata
	meta, err := ParseMetadata(signedContent)
	require.NoError(t, err, "failed to parse metadata")

	assert.Equal(t, "acme", meta.Org)
	assert.Equal(t, "infra", meta.Team)
	assert.Equal(t, "main", meta.Project)
	assert.Equal(t, "test-key", meta.PublicKeyID)
	assert.NotEmpty(t, meta.Signature)

	// verify signature is valid
	sigBytes, _ := base64.StdEncoding.DecodeString(meta.Signature)
	contentWithoutMeta := StripMetadata(signedContent)
	pubKeyBytes, _ := base64.StdEncoding.DecodeString(pubKeyBase64)

	assert.True(t, ed25519.Verify(ed25519.PublicKey(pubKeyBytes), contentWithoutMeta, sigBytes), "signature verification failed on signed content")
}

func TestSigner_SignContent_ReplacesExistingMetadata(t *testing.T) {
	privKeyBytes := make([]byte, ed25519.PrivateKeySize)
	for i := range privKeyBytes {
		privKeyBytes[i] = byte(i)
	}
	signer := NewSignerFromKey(ed25519.PrivateKey(privKeyBytes), "new-key")

	// content with existing metadata
	content := []byte(`# Guidance

Some content.

<!-- SAGEOX_METADATA
{
  "org": "old-org",
  "signature": "old-signature"
}
-->
`)

	signedContent, err := signer.SignContent(content, "new-org", "new-team", "new-project")
	require.NoError(t, err, "SignContent() error")

	meta, _ := ParseMetadata(signedContent)
	assert.Equal(t, "new-org", meta.Org)
	assert.Equal(t, "new-key", meta.PublicKeyID)

	// should only have one metadata block
	count := strings.Count(string(signedContent), "SAGEOX_METADATA")
	assert.Equal(t, 1, count, "should have exactly 1 metadata block")
}

func TestSignerFromKey(t *testing.T) {
	// create a deterministic key for testing
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)

	signer := NewSignerFromKey(privateKey, "custom-key-id")

	assert.Equal(t, "custom-key-id", signer.KeyID())

	// signing should work
	sig := signer.Sign([]byte("test"))
	assert.NotEmpty(t, sig, "Sign() returned empty signature")
}

func TestRoundTrip_SignAndVerify(t *testing.T) {
	// this tests the full round-trip: sign with Signer, verify with existing verification
	pubKeyBase64, privKeyBase64, err := GenerateKeyPair()
	require.NoError(t, err, "failed to generate key pair")

	// temporarily add public key to registry for verification
	pubKeyBytes, _ := base64.StdEncoding.DecodeString(pubKeyBase64)
	SageoxPublicKeys["test-roundtrip-key"] = pubKeyBytes
	defer delete(SageoxPublicKeys, "test-roundtrip-key")

	// create signer
	privKeyBytes, _ := base64.StdEncoding.DecodeString(privKeyBase64)
	signer := NewSignerFromKey(ed25519.PrivateKey(privKeyBytes), "test-roundtrip-key")

	// sign content
	content := []byte("# Important Guidance\n\nDo the thing correctly.\n")
	signedContent, err := signer.SignContent(content, "testorg", "testteam", "testproject")
	require.NoError(t, err, "SignContent() error")

	// write to temp file for VerifyFile
	tmpFile, err := os.CreateTemp("", "sageox-*.md")
	require.NoError(t, err, "failed to create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(signedContent)
	require.NoError(t, err, "failed to write temp file")
	tmpFile.Close()

	// verify using existing verification infrastructure
	result, err := VerifyFile(tmpFile.Name())
	require.NoError(t, err, "VerifyFile() error")

	assert.True(t, result.Verified, "signature should be verified")
	assert.True(t, result.HasMetadata, "should have metadata")
	assert.True(t, result.HasSignature, "should have signature")
	assert.Equal(t, "test-roundtrip-key", result.KeyID)
}
