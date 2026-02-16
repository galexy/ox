package session

import (
	"github.com/sageox/ox/internal/signing"
)

// ArtifactName is the signing artifact name for the redaction manifest.
const ArtifactName = "redaction"

func init() {
	// register the redaction manifest with the signing system
	signing.RegisterArtifact(ArtifactName, generateRedactionManifest)
}

// generateRedactionManifest produces the canonical bytes for signing.
func generateRedactionManifest() ([]byte, error) {
	manifest := GenerateManifest()
	return manifest.CanonicalJSON()
}

// VerifyRedactionSignature checks that the redaction manifest hasn't been tampered with.
func VerifyRedactionSignature() *signing.VerificationResult {
	return signing.Verify(ArtifactName)
}

// IsRedactionSigned returns true if redaction signing is configured.
func IsRedactionSigned() bool {
	return signing.IsSigned()
}
