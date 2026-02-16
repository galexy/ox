package signature

// VerificationResult represents the outcome of signature verification
type VerificationResult struct {
	Verified      bool   // true if signature is valid
	HasMetadata   bool   // true if metadata block was found
	HasSignature  bool   // true if signature field is present in metadata
	KeyID         string // the key ID used for verification (if any)
	Metadata      *MetadataBlock
	Error         string // error message if verification failed
	SignatureInfo string // human-readable signature status
}
