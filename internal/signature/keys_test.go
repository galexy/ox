package signature

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPublicKey(t *testing.T) {
	publicKey, err := GetPublicKey()
	require.NoError(t, err, "GetPublicKey() error")

	assert.Len(t, publicKey, ed25519.PublicKeySize, "GetPublicKey() returned key of wrong size")
}

func TestGetPublicKeyByID(t *testing.T) {
	tests := []struct {
		name    string
		keyID   string
		wantErr bool
	}{
		{
			name:    "valid dev key",
			keyID:   "sageox-dev-2025-01",
			wantErr: false,
		},
		{
			name:    "unknown key ID",
			keyID:   "unknown-key",
			wantErr: true,
		},
		{
			name:    "empty key ID",
			keyID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GetPublicKeyByID(tt.keyID)
			if tt.wantErr {
				assert.Error(t, err, "GetPublicKeyByID() expected error")
				return
			}
			require.NoError(t, err, "GetPublicKeyByID() unexpected error")
			assert.Len(t, key, ed25519.PublicKeySize, "GetPublicKeyByID() returned key of wrong size")
		})
	}
}

func TestVerifySignature(t *testing.T) {
	// generate a test keypair for signing
	// this is separate from the embedded keys - just for testing verification logic
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err, "failed to generate test keypair")

	tests := []struct {
		name      string
		content   []byte
		signature []byte
		publicKey ed25519.PublicKey
		want      bool
	}{
		{
			name:      "valid signature",
			content:   []byte("test message"),
			signature: ed25519.Sign(priv, []byte("test message")),
			publicKey: pub,
			want:      true,
		},
		{
			name:      "invalid signature",
			content:   []byte("test message"),
			signature: make([]byte, ed25519.SignatureSize),
			publicKey: pub,
			want:      false,
		},
		{
			name:      "wrong content",
			content:   []byte("different message"),
			signature: ed25519.Sign(priv, []byte("test message")),
			publicKey: pub,
			want:      false,
		},
		{
			name:      "empty signature",
			content:   []byte("test message"),
			signature: []byte{},
			publicKey: pub,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ed25519.Verify(tt.publicKey, tt.content, tt.signature)
			assert.Equal(t, tt.want, got, "ed25519.Verify()")
		})
	}
}

func TestVerifySignatureWithKeyID(t *testing.T) {
	// test that VerifySignatureWithKeyID returns false for unknown key
	got := VerifySignatureWithKeyID([]byte("test"), []byte("fake sig"), "unknown-key")
	assert.False(t, got, "VerifySignatureWithKeyID() with unknown key should be false")
}

func TestEmbeddedKeyFormat(t *testing.T) {
	// verify that the embedded dev key is properly formatted
	expectedKeyID := "sageox-dev-2025-01"
	expectedBase64 := "DIpRfK3WCVDc4A27r3WbIf+RmWxnRalISUaT55i0hdE="

	keyBytes, ok := SageoxPublicKeys[expectedKeyID]
	require.True(t, ok, "expected key %s not found in SageoxPublicKeys", expectedKeyID)

	// verify it's 32 bytes
	assert.Len(t, keyBytes, ed25519.PublicKeySize, "key %s has wrong size", expectedKeyID)

	// verify base64 encoding matches
	actualBase64 := base64.StdEncoding.EncodeToString(keyBytes)
	assert.Equal(t, expectedBase64, actualBase64, "key base64 mismatch")
}

func TestMustDecodeBase64(t *testing.T) {
	// test valid base64
	result := mustDecodeBase64("dGVzdA==")
	assert.Equal(t, "test", string(result))

	// test that invalid base64 panics
	assert.Panics(t, func() {
		mustDecodeBase64("invalid!!!")
	}, "mustDecodeBase64() with invalid base64 did not panic")
}
