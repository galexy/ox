package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

// test artifact for signing tests
func init() {
	RegisterArtifact("test_artifact", func() ([]byte, error) {
		return []byte(`{"test":"data","version":"1"}`), nil
	})
}

func TestRegisterAndListArtifacts(t *testing.T) {
	artifacts := ListArtifacts()
	if len(artifacts) == 0 {
		t.Error("expected at least one artifact registered")
	}

	// check test artifact is present
	found := false
	for _, name := range artifacts {
		if name == "test_artifact" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test_artifact not found in registered artifacts")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("public key size: got %d, want %d", len(pub), ed25519.PublicKeySize)
	}

	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("private key size: got %d, want %d", len(priv), ed25519.PrivateKeySize)
	}
}

func TestSignAndVerify(t *testing.T) {
	// generate key pair
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// get manifest bytes
	manifest, err := GetManifestBytes("test_artifact")
	if err != nil {
		t.Fatalf("failed to get manifest: %v", err)
	}

	// sign the manifest
	sig := Sign(priv, manifest)
	if len(sig) != ed25519.SignatureSize {
		t.Errorf("signature size: got %d, want %d", len(sig), ed25519.SignatureSize)
	}

	// set up embedded values for verification
	oldPubKey := EmbeddedPublicKey
	oldSigs := EmbeddedSignatures
	defer func() {
		EmbeddedPublicKey = oldPubKey
		EmbeddedSignatures = oldSigs
	}()

	EmbeddedPublicKey = base64.StdEncoding.EncodeToString(pub)
	EmbeddedSignatures = "test_artifact:" + base64.StdEncoding.EncodeToString(sig)

	// verify should succeed
	result := Verify("test_artifact")
	if result.Status != StatusValid {
		t.Errorf("verification status: got %v, want %v", result.Status, StatusValid)
		if result.Error != nil {
			t.Errorf("error: %v", result.Error)
		}
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	// generate key pair
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// set up embedded values with invalid signature
	oldPubKey := EmbeddedPublicKey
	oldSigs := EmbeddedSignatures
	defer func() {
		EmbeddedPublicKey = oldPubKey
		EmbeddedSignatures = oldSigs
	}()

	EmbeddedPublicKey = base64.StdEncoding.EncodeToString(pub)
	// create a fake signature (wrong length will cause StatusError, valid length but wrong bytes causes StatusInvalid)
	fakeSig := make([]byte, ed25519.SignatureSize)
	EmbeddedSignatures = "test_artifact:" + base64.StdEncoding.EncodeToString(fakeSig)

	// verify should fail
	result := Verify("test_artifact")
	if result.Status != StatusInvalid {
		t.Errorf("verification status: got %v, want %v", result.Status, StatusInvalid)
	}
}

func TestVerifyNotConfigured(t *testing.T) {
	// ensure embedded values are empty
	oldPubKey := EmbeddedPublicKey
	oldSigs := EmbeddedSignatures
	defer func() {
		EmbeddedPublicKey = oldPubKey
		EmbeddedSignatures = oldSigs
	}()

	EmbeddedPublicKey = ""
	EmbeddedSignatures = ""

	result := Verify("test_artifact")
	if result.Status != StatusNotConfigured {
		t.Errorf("verification status: got %v, want %v", result.Status, StatusNotConfigured)
	}

	// hash should still be computed
	if result.Hash == "" {
		t.Error("hash should be computed even when not configured")
	}
}

func TestVerifyUnknownArtifact(t *testing.T) {
	result := Verify("nonexistent_artifact")
	if result.Status != StatusError {
		t.Errorf("verification status: got %v, want %v", result.Status, StatusError)
	}
}

func TestVerifyMissingSignature(t *testing.T) {
	// generate key pair
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// set up embedded values without our artifact's signature
	oldPubKey := EmbeddedPublicKey
	oldSigs := EmbeddedSignatures
	defer func() {
		EmbeddedPublicKey = oldPubKey
		EmbeddedSignatures = oldSigs
	}()

	EmbeddedPublicKey = base64.StdEncoding.EncodeToString(pub)
	EmbeddedSignatures = "other_artifact:AAAA" // different artifact

	result := Verify("test_artifact")
	if result.Status != StatusMissing {
		t.Errorf("verification status: got %v, want %v", result.Status, StatusMissing)
	}
}

func TestGetManifestBytes(t *testing.T) {
	bytes, err := GetManifestBytes("test_artifact")
	if err != nil {
		t.Fatalf("failed to get manifest bytes: %v", err)
	}

	expected := `{"test":"data","version":"1"}`
	if string(bytes) != expected {
		t.Errorf("manifest bytes: got %q, want %q", string(bytes), expected)
	}
}

func TestGetManifestBytesUnknown(t *testing.T) {
	_, err := GetManifestBytes("nonexistent")
	if err == nil {
		t.Error("expected error for unknown artifact")
	}
}

func TestHash(t *testing.T) {
	data := []byte("test data")
	hash := Hash(data)

	// SHA-256 produces 64 hex characters
	if len(hash) != 64 {
		t.Errorf("hash length: got %d, want 64", len(hash))
	}

	// hash should be deterministic
	hash2 := Hash(data)
	if hash != hash2 {
		t.Error("hash is not deterministic")
	}
}

func TestFindSignature(t *testing.T) {
	tests := []struct {
		name       string
		signatures string
		artifact   string
		want       string
	}{
		{
			name:       "single artifact",
			signatures: "artifact1:sig1",
			artifact:   "artifact1",
			want:       "sig1",
		},
		{
			name:       "multiple artifacts first",
			signatures: "artifact1:sig1;artifact2:sig2;artifact3:sig3",
			artifact:   "artifact1",
			want:       "sig1",
		},
		{
			name:       "multiple artifacts middle",
			signatures: "artifact1:sig1;artifact2:sig2;artifact3:sig3",
			artifact:   "artifact2",
			want:       "sig2",
		},
		{
			name:       "multiple artifacts last",
			signatures: "artifact1:sig1;artifact2:sig2;artifact3:sig3",
			artifact:   "artifact3",
			want:       "sig3",
		},
		{
			name:       "not found",
			signatures: "artifact1:sig1;artifact2:sig2",
			artifact:   "artifact3",
			want:       "",
		},
		{
			name:       "empty signatures",
			signatures: "",
			artifact:   "artifact1",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSigs := EmbeddedSignatures
			defer func() { EmbeddedSignatures = oldSigs }()

			EmbeddedSignatures = tt.signatures
			got := findSignature(tt.artifact)
			if got != tt.want {
				t.Errorf("findSignature(%q) = %q, want %q", tt.artifact, got, tt.want)
			}
		})
	}
}

func TestVerificationStatusString(t *testing.T) {
	tests := []struct {
		status VerificationStatus
		want   string
	}{
		{StatusNotConfigured, "not_configured"},
		{StatusValid, "valid"},
		{StatusInvalid, "invalid"},
		{StatusError, "error"},
		{StatusMissing, "missing"},
		{VerificationStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("VerificationStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestIsSigned(t *testing.T) {
	oldPubKey := EmbeddedPublicKey
	oldSigs := EmbeddedSignatures
	defer func() {
		EmbeddedPublicKey = oldPubKey
		EmbeddedSignatures = oldSigs
	}()

	// both empty
	EmbeddedPublicKey = ""
	EmbeddedSignatures = ""
	if IsSigned() {
		t.Error("IsSigned() should return false when both are empty")
	}

	// only public key
	EmbeddedPublicKey = "test"
	EmbeddedSignatures = ""
	if IsSigned() {
		t.Error("IsSigned() should return false when signatures are empty")
	}

	// only signatures
	EmbeddedPublicKey = ""
	EmbeddedSignatures = "test"
	if IsSigned() {
		t.Error("IsSigned() should return false when public key is empty")
	}

	// both set
	EmbeddedPublicKey = "test"
	EmbeddedSignatures = "test"
	if !IsSigned() {
		t.Error("IsSigned() should return true when both are set")
	}
}

func TestVerifyAll(t *testing.T) {
	results := VerifyAll()

	// should have at least test_artifact
	if len(results) == 0 {
		t.Error("VerifyAll() returned no results")
	}

	// check test_artifact is present
	result, ok := results["test_artifact"]
	if !ok {
		t.Error("test_artifact not in VerifyAll results")
	}

	// without signing configured, should be StatusNotConfigured
	if result.Status != StatusNotConfigured {
		t.Errorf("test_artifact status: got %v, want %v", result.Status, StatusNotConfigured)
	}
}
