package session

import (
	"encoding/json"
	"testing"
)

func TestGenerateManifest(t *testing.T) {
	manifest := GenerateManifest()

	// verify schema version
	if manifest.SchemaVersion != ManifestSchemaVersion {
		t.Errorf("schema version: got %s, want %s", manifest.SchemaVersion, ManifestSchemaVersion)
	}

	// verify we have patterns
	if manifest.PatternCount() == 0 {
		t.Error("manifest has no patterns")
	}

	// verify patterns are sorted by name
	for i := 1; i < len(manifest.Patterns); i++ {
		if manifest.Patterns[i].Name < manifest.Patterns[i-1].Name {
			t.Errorf("patterns not sorted: %s comes after %s",
				manifest.Patterns[i].Name, manifest.Patterns[i-1].Name)
		}
	}
}

func TestManifestDeterministic(t *testing.T) {
	// generate manifest twice and verify identical output
	m1 := GenerateManifest()
	m2 := GenerateManifest()

	j1, err := m1.CanonicalJSON()
	if err != nil {
		t.Fatalf("failed to generate JSON for m1: %v", err)
	}

	j2, err := m2.CanonicalJSON()
	if err != nil {
		t.Fatalf("failed to generate JSON for m2: %v", err)
	}

	if string(j1) != string(j2) {
		t.Error("manifest generation is not deterministic")
	}

	// verify hash is also deterministic
	h1, _ := m1.HashHex()
	h2, _ := m2.HashHex()

	if h1 != h2 {
		t.Errorf("hash not deterministic: %s != %s", h1, h2)
	}
}

func TestManifestHash(t *testing.T) {
	manifest := GenerateManifest()

	hash, err := manifest.Hash()
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	// SHA-256 produces 32 bytes
	if len(hash) != 32 {
		t.Errorf("hash length: got %d, want 32", len(hash))
	}

	// verify hex encoding
	hashHex, err := manifest.HashHex()
	if err != nil {
		t.Fatalf("failed to compute hex hash: %v", err)
	}

	// SHA-256 hex is 64 characters
	if len(hashHex) != 64 {
		t.Errorf("hex hash length: got %d, want 64", len(hashHex))
	}
}

func TestManifestJSON(t *testing.T) {
	manifest := GenerateManifest()

	// test canonical JSON
	canonical, err := manifest.CanonicalJSON()
	if err != nil {
		t.Fatalf("failed to generate canonical JSON: %v", err)
	}

	// verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(canonical, &parsed); err != nil {
		t.Errorf("canonical JSON is not valid: %v", err)
	}

	// test pretty JSON
	pretty, err := manifest.PrettyJSON()
	if err != nil {
		t.Fatalf("failed to generate pretty JSON: %v", err)
	}

	// verify it's valid JSON
	if err := json.Unmarshal(pretty, &parsed); err != nil {
		t.Errorf("pretty JSON is not valid: %v", err)
	}

	// pretty should be longer (has indentation)
	if len(pretty) <= len(canonical) {
		t.Error("pretty JSON should be longer than canonical")
	}
}

func TestManifestFindPattern(t *testing.T) {
	manifest := GenerateManifest()

	// find existing pattern
	pattern := manifest.FindPattern("aws_access_key")
	if pattern == nil {
		t.Fatal("failed to find aws_access_key pattern")
	}

	if pattern.Name != "aws_access_key" {
		t.Errorf("wrong pattern name: got %s", pattern.Name)
	}

	// non-existent pattern
	pattern = manifest.FindPattern("nonexistent_pattern")
	if pattern != nil {
		t.Error("found pattern that should not exist")
	}
}

func TestManifestPatternContent(t *testing.T) {
	manifest := GenerateManifest()

	// verify all patterns have required fields
	for _, p := range manifest.Patterns {
		if p.Name == "" {
			t.Error("pattern with empty name")
		}
		if p.Regex == "" {
			t.Errorf("pattern %s has empty regex", p.Name)
		}
		if p.Redact == "" {
			t.Errorf("pattern %s has empty redact token", p.Name)
		}
	}
}
