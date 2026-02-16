package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// ManifestSchemaVersion is incremented when the manifest structure changes.
// Signature verification requires matching schema versions.
const ManifestSchemaVersion = "1"

// PatternManifestEntry represents a single pattern in the manifest.
// This is the canonical, signable representation of a SecretPattern.
type PatternManifestEntry struct {
	Name   string `json:"name"`
	Regex  string `json:"regex"`
	Redact string `json:"redact"`
}

// RedactionManifest is the complete, signable manifest of all redaction patterns.
// The structure is designed for deterministic serialization.
type RedactionManifest struct {
	SchemaVersion string                 `json:"schema_version"`
	Patterns      []PatternManifestEntry `json:"patterns"`
}

// GenerateManifest creates a deterministic manifest from the default patterns.
// Patterns are sorted by name for consistent ordering.
func GenerateManifest() *RedactionManifest {
	patterns := DefaultPatterns()
	entries := make([]PatternManifestEntry, len(patterns))

	for i, p := range patterns {
		entries[i] = PatternManifestEntry{
			Name:   p.Name,
			Regex:  p.Pattern.String(),
			Redact: p.Redact,
		}
	}

	// sort by name for deterministic output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return &RedactionManifest{
		SchemaVersion: ManifestSchemaVersion,
		Patterns:      entries,
	}
}

// Hash returns the SHA-256 hash of the manifest's canonical JSON representation.
// This hash is what gets signed during release.
func (m *RedactionManifest) Hash() ([]byte, error) {
	canonical, err := m.CanonicalJSON()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(canonical)
	return hash[:], nil
}

// HashHex returns the hex-encoded SHA-256 hash of the manifest.
func (m *RedactionManifest) HashHex() (string, error) {
	hash, err := m.Hash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// CanonicalJSON returns the deterministic JSON encoding of the manifest.
// Uses sorted keys and no extra whitespace for consistent hashing.
func (m *RedactionManifest) CanonicalJSON() ([]byte, error) {
	// json.Marshal produces deterministic output for structs with defined field order
	// and we've already sorted the patterns slice
	return json.Marshal(m)
}

// PrettyJSON returns human-readable JSON for display purposes.
func (m *RedactionManifest) PrettyJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// PatternCount returns the number of patterns in the manifest.
func (m *RedactionManifest) PatternCount() int {
	return len(m.Patterns)
}

// FindPattern looks up a pattern by name.
func (m *RedactionManifest) FindPattern(name string) *PatternManifestEntry {
	for i := range m.Patterns {
		if m.Patterns[i].Name == name {
			return &m.Patterns[i]
		}
	}
	return nil
}
