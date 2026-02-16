package signature

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// MetadataBlock represents the metadata in SAGEOX.md footer
type MetadataBlock struct {
	Org         string    `json:"org"`
	Team        string    `json:"team"`
	Project     string    `json:"project"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublicKeyID string    `json:"public_key_id"`
	Signature   string    `json:"signature"`
}

var (
	// regex to match the metadata block
	// matches: <!-- SAGEOX_METADATA\n{...}\n-->
	metadataRegex = regexp.MustCompile(`(?s)<!-- SAGEOX_METADATA\s*\n(.*?)\n-->`)
)

// ParseMetadata extracts metadata from SAGEOX.md content
// Returns nil if no metadata block found (not an error)
func ParseMetadata(content []byte) (*MetadataBlock, error) {
	matches := metadataRegex.FindSubmatch(content)
	if matches == nil {
		return nil, nil // no metadata block found
	}

	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid metadata block format")
	}

	jsonData := matches[1]
	var meta MetadataBlock
	if err := json.Unmarshal(jsonData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return &meta, nil
}

// WriteMetadata appends or replaces the metadata block in SAGEOX.md content
// If metadata block already exists, it replaces it; otherwise appends to end
func WriteMetadata(content []byte, meta *MetadataBlock) ([]byte, error) {
	if meta == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	// marshal metadata to JSON with indentation
	jsonData, err := json.MarshalIndent(meta, "  ", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// construct metadata block
	var metadataBlock bytes.Buffer
	metadataBlock.WriteString("<!-- SAGEOX_METADATA\n")
	metadataBlock.Write(jsonData)
	metadataBlock.WriteString("\n-->")

	// strip existing metadata if present
	contentWithoutMetadata := StripMetadata(content)

	// ensure content ends with newline before appending metadata
	result := bytes.TrimRight(contentWithoutMetadata, "\n")
	result = append(result, '\n', '\n')
	result = append(result, metadataBlock.Bytes()...)
	result = append(result, '\n')

	return result, nil
}

// StripMetadata removes the metadata block from content, returning only the content
// This is useful for hashing the content without the signature
func StripMetadata(content []byte) []byte {
	return metadataRegex.ReplaceAll(content, nil)
}

// HasMetadata checks if content contains a metadata block
func HasMetadata(content []byte) bool {
	return metadataRegex.Match(content)
}
