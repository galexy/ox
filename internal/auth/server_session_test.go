package auth

import (
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerSessionID(t *testing.T) {
	id := NewServerSessionID()

	assert.True(t, strings.HasPrefix(id, "oxsid_"), "expected session ID to start with 'oxsid_', got: %s", id)

	// verify the ULID portion is valid
	ulidPart := strings.TrimPrefix(id, "oxsid_")
	_, err := ulid.Parse(ulidPart)
	assert.NoError(t, err, "expected valid ULID")
}

func TestNewServerSessionID_Uniqueness(t *testing.T) {
	iterations := 1000
	ids := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		id := NewServerSessionID()
		assert.False(t, ids[id], "duplicate session ID generated: %s", id)
		ids[id] = true
	}
}

func TestIsValidServerSessionID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			name:  "valid session ID",
			id:    NewServerSessionID(),
			valid: true,
		},
		{
			name:  "valid session ID with known ULID",
			id:    "oxsid_01JEYQ9Z8X9Y2K3N4P5Q6R7S8T",
			valid: true,
		},
		{
			name:  "missing prefix",
			id:    "01JEYQ9Z8X9Y2K3N4P5Q6R7S8T",
			valid: false,
		},
		{
			name:  "wrong prefix",
			id:    "session_01JEYQ9Z8X9Y2K3N4P5Q6R7S8T",
			valid: false,
		},
		{
			name:  "empty string",
			id:    "",
			valid: false,
		},
		{
			name:  "prefix only",
			id:    "oxsid_",
			valid: false,
		},
		{
			name:  "invalid ULID characters",
			id:    "oxsid_invalid-ulid-value",
			valid: false,
		},
		{
			name:  "ULID too short",
			id:    "oxsid_123",
			valid: false,
		},
		{
			name:  "ULID too long",
			id:    "oxsid_01JEYQ9Z8X9Y2K3N4P5Q6R7S8T99",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidServerSessionID(tt.id)
			assert.Equal(t, tt.valid, result, "IsValidServerSessionID(%q)", tt.id)
		})
	}
}

func TestIsValidServerSessionID_GeneratedIDs(t *testing.T) {
	// verify that all generated IDs are valid
	for i := 0; i < 100; i++ {
		id := NewServerSessionID()
		require.True(t, IsValidServerSessionID(id), "generated session ID is invalid: %s", id)
	}
}

func TestServerSessionFromResponse(t *testing.T) {
	tests := []struct {
		name        string
		serverID    string
		expectValid bool
		expectSame  bool // if true, expects same ID returned
	}{
		{
			name:        "valid server ID returned as-is",
			serverID:    "oxsid_01JEYQ9Z8X9Y2K3N4P5Q6R7S8T",
			expectValid: true,
			expectSame:  true,
		},
		{
			name:        "empty server ID generates new",
			serverID:    "",
			expectValid: true,
			expectSame:  false,
		},
		{
			name:        "invalid server ID generates new",
			serverID:    "invalid",
			expectValid: true,
			expectSame:  false,
		},
		{
			name:        "wrong prefix generates new",
			serverID:    "session_01JEYQ9Z8X9Y2K3N4P5Q6R7S8T",
			expectValid: true,
			expectSame:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ServerSessionFromResponse(tt.serverID)

			assert.True(t, IsValidServerSessionID(result), "result should be valid session ID")

			if tt.expectSame {
				assert.Equal(t, tt.serverID, result, "should return same ID")
			} else if tt.serverID != "" {
				assert.NotEqual(t, tt.serverID, result, "should generate new ID")
			}
		})
	}
}

func BenchmarkNewServerSessionID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewServerSessionID()
	}
}

func BenchmarkIsValidServerSessionID(b *testing.B) {
	id := NewServerSessionID()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidServerSessionID(id)
	}
}
