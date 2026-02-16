package agentx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentHash(t *testing.T) {
	content := []byte("# Test\nSome content\n")
	hash := ContentHash(content)
	assert.Len(t, hash, 12)

	// same content = same hash
	assert.Equal(t, hash, ContentHash(content))

	// different content = different hash
	other := ContentHash([]byte("# Different\n"))
	assert.NotEqual(t, hash, other)
}

func TestStampedContent(t *testing.T) {
	content := []byte("# Test\nSome content\n")
	stamped := StampedContent(content, "0.13.0")

	hash := ContentHash(content)
	expected := "<!-- ox-hash: " + hash + " ver: 0.13.0 -->\n# Test\nSome content\n"
	assert.Equal(t, expected, string(stamped))
}

func TestExtractCommandHash(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"stamped", "<!-- ox-hash: abcdef012345 ver: 0.13.0 -->\n# Test\n", "abcdef012345"},
		{"no stamp", "# Test\nSome content\n", ""},
		{"empty", "", ""},
		{"wrong prefix", "<!-- ox-version: 0.10.0 -->\n# Test\n", ""},
		{"no newline", "<!-- ox-hash: abcdef012345 ver: 0.13.0 -->", "abcdef012345"},
		{"short hash", "<!-- ox-hash: abc -->", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCommandHash([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractStampVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"stamped", "<!-- ox-hash: abcdef012345 ver: 0.13.0 -->\n# Test\n", "0.13.0"},
		{"no stamp", "# Test\n", ""},
		{"no version in stamp", "<!-- ox-hash: abcdef012345 -->\n# Test\n", ""},
		{"wrong prefix", "<!-- ox-version: 0.10.0 -->\n# Test\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStampVersion([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"0.9.0", "0.10.0", true},   // 9 < 10
		{"0.10.0", "0.10.0", false}, // equal
		{"0.10.0", "0.9.0", false},  // 10 > 9
		{"0.10.0", "0.11.0", true},  // minor bump
		{"0.10.0", "1.0.0", true},   // major bump
		{"1.0.0", "0.10.0", false},  // major > minor
		{"0.10", "0.10.1", true},    // patch bump
		{"0.10.1", "0.10", false},   // has patch, other doesn't
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			result := CompareVersions(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "%s < %s should be %v", tt.a, tt.b, tt.expected)
		})
	}
}

func TestStampedContentRoundTrip(t *testing.T) {
	content := []byte("# Command\nDo something\n")
	stamped := StampedContent(content, "0.12.0")

	assert.Equal(t, ContentHash(content), ExtractCommandHash(stamped))
	assert.Equal(t, "0.12.0", ExtractStampVersion(stamped))
}

func TestShouldWriteCommand(t *testing.T) {
	cmd := CommandFile{
		Name:    "test.md",
		Content: []byte("# Test\n"),
		Version: "0.12.0",
	}
	stamped := StampedContent(cmd.Content, cmd.Version)

	tests := []struct {
		name      string
		existing  []byte
		cmd       CommandFile
		overwrite bool
		expected  bool
	}{
		{"new file", nil, cmd, false, true},
		{"new file overwrite", nil, cmd, true, true},
		{"exists no overwrite", stamped, cmd, false, false},
		{"same hash", stamped, cmd, true, false},
		{"user managed", []byte("# Custom\n"), cmd, true, false},
		{
			"different content newer version",
			stamped,
			CommandFile{Name: "test.md", Content: []byte("# Updated\n"), Version: "0.13.0"},
			true,
			true,
		},
		{
			"different content same version",
			stamped,
			CommandFile{Name: "test.md", Content: []byte("# Updated\n"), Version: "0.12.0"},
			true,
			true,
		},
		{
			"downgrade blocked",
			StampedContent([]byte("# Newer\n"), "0.13.0"),
			CommandFile{Name: "test.md", Content: []byte("# Older\n"), Version: "0.12.0"},
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldWriteCommand(tt.existing, tt.cmd, tt.overwrite)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCommandStale(t *testing.T) {
	cmd := CommandFile{
		Name:    "test.md",
		Content: []byte("# Test\n"),
		Version: "0.12.0",
	}
	stamped := StampedContent(cmd.Content, cmd.Version)

	tests := []struct {
		name     string
		existing []byte
		cmd      CommandFile
		expected bool
	}{
		{"same hash", stamped, cmd, false},
		{"user managed", []byte("# Custom\n"), cmd, false},
		{
			"different content newer version",
			stamped,
			CommandFile{Name: "test.md", Content: []byte("# Updated\n"), Version: "0.13.0"},
			true,
		},
		{
			"installed by newer version",
			StampedContent([]byte("# Newer\n"), "0.13.0"),
			CommandFile{Name: "test.md", Content: []byte("# Older\n"), Version: "0.12.0"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommandStale(tt.existing, tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}
