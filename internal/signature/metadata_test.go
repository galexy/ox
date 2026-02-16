package signature

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantMeta  *MetadataBlock
		wantErr   bool
		expectNil bool
	}{
		{
			name: "valid metadata block",
			content: `# SAGEOX.md

Some content here.

<!-- SAGEOX_METADATA
{
  "org": "ghostlayer",
  "team": "platform",
  "project": "sageox",
  "updated_at": "2025-12-09T12:00:00Z",
  "signature": "base64-encoded-signature"
}
-->
`,
			wantMeta: &MetadataBlock{
				Org:       "ghostlayer",
				Team:      "platform",
				Project:   "sageox",
				UpdatedAt: mustParseTime("2025-12-09T12:00:00Z"),
				Signature: "base64-encoded-signature",
			},
			wantErr:   false,
			expectNil: false,
		},
		{
			name: "no metadata block",
			content: `# SAGEOX.md

Some content here without metadata.
`,
			wantMeta:  nil,
			wantErr:   false,
			expectNil: true,
		},
		{
			name: "invalid JSON in metadata",
			content: `# SAGEOX.md

<!-- SAGEOX_METADATA
{invalid json}
-->
`,
			wantMeta:  nil,
			wantErr:   true,
			expectNil: false,
		},
		{
			name: "metadata with extra whitespace",
			content: `# SAGEOX.md

<!-- SAGEOX_METADATA

{
  "org": "ghostlayer",
  "team": "platform",
  "project": "sageox",
  "updated_at": "2025-12-09T12:00:00Z",
  "signature": "sig123"
}

-->
`,
			wantMeta: &MetadataBlock{
				Org:       "ghostlayer",
				Team:      "platform",
				Project:   "sageox",
				UpdatedAt: mustParseTime("2025-12-09T12:00:00Z"),
				Signature: "sig123",
			},
			wantErr:   false,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := ParseMetadata([]byte(tt.content))

			if tt.wantErr {
				assert.Error(t, err, "ParseMetadata() expected error")
				return
			}
			require.NoError(t, err, "ParseMetadata() unexpected error")

			if tt.expectNil {
				assert.Nil(t, meta, "ParseMetadata() expected nil")
				return
			}

			require.NotNil(t, meta, "ParseMetadata() got nil, want metadata")

			assert.Equal(t, tt.wantMeta.Org, meta.Org)
			assert.Equal(t, tt.wantMeta.Team, meta.Team)
			assert.Equal(t, tt.wantMeta.Project, meta.Project)
			assert.True(t, meta.UpdatedAt.Equal(tt.wantMeta.UpdatedAt))
			assert.Equal(t, tt.wantMeta.Signature, meta.Signature)
		})
	}
}

func TestWriteMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		meta    *MetadataBlock
		wantErr bool
	}{
		{
			name:    "append metadata to content without existing metadata",
			content: "# SAGEOX.md\n\nSome content here.\n",
			meta: &MetadataBlock{
				Org:       "ghostlayer",
				Team:      "platform",
				Project:   "sageox",
				UpdatedAt: mustParseTime("2025-12-09T12:00:00Z"),
				Signature: "sig123",
			},
			wantErr: false,
		},
		{
			name: "replace existing metadata",
			content: `# SAGEOX.md

Some content here.

<!-- SAGEOX_METADATA
{
  "org": "old-org",
  "team": "old-team",
  "project": "old-project",
  "updated_at": "2025-01-01T00:00:00Z",
  "signature": "old-sig"
}
-->
`,
			meta: &MetadataBlock{
				Org:       "ghostlayer",
				Team:      "platform",
				Project:   "sageox",
				UpdatedAt: mustParseTime("2025-12-09T12:00:00Z"),
				Signature: "new-sig",
			},
			wantErr: false,
		},
		{
			name:    "nil metadata returns error",
			content: "# SAGEOX.md\n",
			meta:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := WriteMetadata([]byte(tt.content), tt.meta)

			if tt.wantErr {
				assert.Error(t, err, "WriteMetadata() expected error")
				return
			}
			require.NoError(t, err, "WriteMetadata() unexpected error")

			// verify metadata can be parsed back
			parsedMeta, err := ParseMetadata(result)
			require.NoError(t, err, "WriteMetadata() produced invalid metadata")
			require.NotNil(t, parsedMeta, "WriteMetadata() metadata not found in result")

			assert.Equal(t, tt.meta.Org, parsedMeta.Org)
			assert.Equal(t, tt.meta.Team, parsedMeta.Team)
			assert.Equal(t, tt.meta.Project, parsedMeta.Project)
			assert.True(t, parsedMeta.UpdatedAt.Equal(tt.meta.UpdatedAt))
			assert.Equal(t, tt.meta.Signature, parsedMeta.Signature)

			// verify original content is preserved (without old metadata)
			stripped := StripMetadata(result)
			originalStripped := StripMetadata([]byte(tt.content))
			assert.True(t, strings.HasPrefix(string(stripped), strings.TrimSpace(string(originalStripped))),
				"WriteMetadata() original content not preserved")
		})
	}
}

func TestStripMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "strip metadata from content",
			content: `# SAGEOX.md

Some content here.

<!-- SAGEOX_METADATA
{
  "org": "ghostlayer",
  "team": "platform",
  "project": "sageox",
  "updated_at": "2025-12-09T12:00:00Z",
  "signature": "sig123"
}
-->
`,
			want: "# SAGEOX.md\n\nSome content here.\n\n\n",
		},
		{
			name:    "no metadata to strip",
			content: "# SAGEOX.md\n\nSome content here.\n",
			want:    "# SAGEOX.md\n\nSome content here.\n",
		},
		{
			name: "multiple metadata blocks (should strip all)",
			content: `# SAGEOX.md

<!-- SAGEOX_METADATA
{"org":"first"}
-->

Some content.

<!-- SAGEOX_METADATA
{"org":"second"}
-->
`,
			want: "# SAGEOX.md\n\n\n\nSome content.\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripMetadata([]byte(tt.content))
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func TestHasMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "has metadata",
			content: `# SAGEOX.md

<!-- SAGEOX_METADATA
{"org":"ghostlayer"}
-->
`,
			want: true,
		},
		{
			name:    "no metadata",
			content: "# SAGEOX.md\n\nSome content.\n",
			want:    false,
		},
		{
			name:    "similar but not metadata comment",
			content: "# SAGEOX.md\n\n<!-- METADATA -->\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasMetadata([]byte(tt.content))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMetadataBlockJSON(t *testing.T) {
	meta := &MetadataBlock{
		Org:       "ghostlayer",
		Team:      "platform",
		Project:   "sageox",
		UpdatedAt: mustParseTime("2025-12-09T12:00:00Z"),
		Signature: "sig123",
	}

	// marshal to JSON
	data, err := json.Marshal(meta)
	require.NoError(t, err, "failed to marshal metadata")

	// unmarshal back
	var meta2 MetadataBlock
	require.NoError(t, json.Unmarshal(data, &meta2), "failed to unmarshal metadata")

	// verify fields match
	assert.Equal(t, meta.Org, meta2.Org)
	assert.Equal(t, meta.Team, meta2.Team)
	assert.Equal(t, meta.Project, meta2.Project)
	assert.True(t, meta2.UpdatedAt.Equal(meta.UpdatedAt))
	assert.Equal(t, meta.Signature, meta2.Signature)
}

// helper function to parse time for tests
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
