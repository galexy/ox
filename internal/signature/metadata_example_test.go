package signature_test

import (
	"fmt"
	"time"

	"github.com/sageox/ox/internal/signature"
)

// Example demonstrates parsing metadata from SAGEOX.md
func ExampleParseMetadata() {
	content := []byte(`# SAGEOX.md

This is the content.

<!-- SAGEOX_METADATA
{
  "org": "ghostlayer",
  "team": "platform",
  "project": "sageox",
  "updated_at": "2025-12-09T12:00:00Z",
  "signature": "base64-encoded-signature"
}
-->
`)

	meta, err := signature.ParseMetadata(content)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if meta != nil {
		fmt.Printf("Org: %s\n", meta.Org)
		fmt.Printf("Project: %s\n", meta.Project)
	}
	// Output:
	// Org: ghostlayer
	// Project: sageox
}

// Example demonstrates writing metadata to SAGEOX.md
func ExampleWriteMetadata() {
	content := []byte("# SAGEOX.md\n\nThis is the content.\n")

	meta := &signature.MetadataBlock{
		Org:         "ghostlayer",
		Team:        "platform",
		Project:     "sageox",
		UpdatedAt:   time.Date(2025, 12, 9, 12, 0, 0, 0, time.UTC),
		PublicKeyID: "sageox-dev-2025-01",
		Signature:   "sig123",
	}

	result, err := signature.WriteMetadata(content, meta)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("%s", result)
	// Output:
	// # SAGEOX.md
	//
	// This is the content.
	//
	// <!-- SAGEOX_METADATA
	// {
	//     "org": "ghostlayer",
	//     "team": "platform",
	//     "project": "sageox",
	//     "updated_at": "2025-12-09T12:00:00Z",
	//     "public_key_id": "sageox-dev-2025-01",
	//     "signature": "sig123"
	//   }
	// -->
}

// Example demonstrates stripping metadata from SAGEOX.md for hashing
func ExampleStripMetadata() {
	content := []byte(`# SAGEOX.md

Content here.

<!-- SAGEOX_METADATA
{"org":"ghostlayer","signature":"sig123"}
-->
`)

	stripped := signature.StripMetadata(content)
	fmt.Printf("%s", stripped)
	// Output:
	// # SAGEOX.md
	//
	// Content here.
}
