package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		verbose  bool
		expected string
	}{
		{
			name:     "long session ID - normal mode",
			id:       "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ",
			verbose:  false,
			expected: "oxsid_01JBCD...WXYZ",
		},
		{
			name:     "long session ID - verbose mode",
			id:       "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ",
			verbose:  true,
			expected: "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name:     "short agent ID - normal mode",
			id:       "Oxa7b3",
			verbose:  false,
			expected: "Oxa7b3",
		},
		{
			name:     "short agent ID - verbose mode",
			id:       "Oxa7b3",
			verbose:  true,
			expected: "Oxa7b3",
		},
		{
			name:     "project ID - normal mode",
			id:       "prj_01JBCDEFGHIJKLMNOP",
			verbose:  false,
			expected: "prj_01JBCDEF...MNOP",
		},
		{
			name:     "project ID - verbose mode",
			id:       "prj_01JBCDEFGHIJKLMNOP",
			verbose:  true,
			expected: "prj_01JBCDEFGHIJKLMNOP",
		},
		{
			name:     "user ID - normal mode",
			id:       "usr_01234567890123456789",
			verbose:  false,
			expected: "usr_01234567...6789",
		},
		{
			name:     "generic long ID - normal mode",
			id:       "abcdefghijklmnopqrstuvwxyz",
			verbose:  false,
			expected: "abcdefgh...",
		},
		{
			name:     "generic long ID - verbose mode",
			id:       "abcdefghijklmnopqrstuvwxyz",
			verbose:  true,
			expected: "abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:     "already short ID - normal mode",
			id:       "abc123",
			verbose:  false,
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateID(tt.id, tt.verbose)
			assert.Equal(t, tt.expected, result, "TruncateID(%q, %v)", tt.id, tt.verbose)
		})
	}
}

func TestTruncateUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		verbose  bool
		expected string
	}{
		{
			name:     "standard UUID - normal mode",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			verbose:  false,
			expected: "a1b2c3d4...",
		},
		{
			name:     "standard UUID - verbose mode",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			verbose:  true,
			expected: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			name:     "short UUID - normal mode",
			uuid:     "abc123",
			verbose:  false,
			expected: "abc123",
		},
		{
			name:     "short UUID - verbose mode",
			uuid:     "abc123",
			verbose:  true,
			expected: "abc123",
		},
		{
			name:     "UUID without hyphens - normal mode",
			uuid:     "a1b2c3d4e5f67890abcdef1234567890",
			verbose:  false,
			expected: "a1b2c3d4...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateUUID(tt.uuid, tt.verbose)
			assert.Equal(t, tt.expected, result, "TruncateUUID(%q, %v)", tt.uuid, tt.verbose)
		})
	}
}
