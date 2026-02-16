package repotools

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRepoID(t *testing.T) {
	id := GenerateRepoID()

	assert.True(t, strings.HasPrefix(id, "repo_"), "expected repo ID to have 'repo_' prefix, got: %s", id)
	assert.Greater(t, len(id), len("repo_"), "expected repo ID to have content after prefix, got: %s", id)

	// verify the format is repo_ + standard UUID (36 chars)
	uuidPart := strings.TrimPrefix(id, "repo_")
	assert.Len(t, uuidPart, 36, "expected UUID part to be 36 chars: %s", uuidPart)
}

func TestGenerateRepoID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id := GenerateRepoID()
		assert.False(t, ids[id], "duplicate repo ID generated: %s", id)
		ids[id] = true
	}

	assert.Len(t, ids, count)
}

func TestGenerateRepoID_TimeSortable(t *testing.T) {
	// generate multiple IDs in sequence
	id1 := GenerateRepoID()
	id2 := GenerateRepoID()
	id3 := GenerateRepoID()

	// with standard UUID format, lexicographic sorting preserves time order
	// because UUIDv7 has the timestamp in the most significant bits
	assert.LessOrEqual(t, id1, id2, "expected first ID to be <= second ID for time sorting")
	assert.LessOrEqual(t, id2, id3, "expected second ID to be <= third ID for time sorting")
}

func TestGenerateRepoID_StringSortMatchesTimeSort(t *testing.T) {
	// this is the key test: verify that ls -la .sageox/.repo_* gives chronological order
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = GenerateRepoID()
	}

	// verify IDs are already in sorted order (since they were generated sequentially)
	for i := 1; i < len(ids); i++ {
		assert.LessOrEqual(t, ids[i-1], ids[i],
			"IDs not in chronological string-sort order at index %d: %s > %s", i, ids[i-1], ids[i])
	}
}

func TestParseRepoID_Valid(t *testing.T) {
	id := GenerateRepoID()

	parsedUUID, err := ParseRepoID(id)
	require.NoError(t, err, "failed to parse valid repo ID")

	assert.NotEqual(t, uuid.Nil, parsedUUID, "expected non-nil UUID from parsed repo ID")

	// verify version is 7
	assert.Equal(t, uuid.Version(7), parsedUUID.Version(), "expected UUID version 7")
}

func TestParseRepoID_RoundTrip(t *testing.T) {
	// generate original ID
	originalID := GenerateRepoID()

	// parse to UUID
	parsedUUID, err := ParseRepoID(originalID)
	require.NoError(t, err, "failed to parse repo ID")

	// encode back to repo ID
	reEncodedID := repoIDPrefix + parsedUUID.String()

	// verify round trip produces same ID
	assert.Equal(t, originalID, reEncodedID, "round trip failed")
}

func TestParseRepoID_InvalidPrefix(t *testing.T) {
	invalidIDs := []string{
		"invalid_01936d5a-0000-7abc-8def-0123456789ab",
		"01936d5a-0000-7abc-8def-0123456789ab",
		"repo01936d5a-0000-7abc-8def-0123456789ab",
		"",
		"repo_",
	}

	for _, id := range invalidIDs {
		_, err := ParseRepoID(id)
		assert.Error(t, err, "expected error for invalid repo ID: %s", id)
	}
}

func TestParseRepoID_InvalidUUID(t *testing.T) {
	invalidIDs := []string{
		"repo_not-a-valid-uuid",
		"repo_!!!invalid",
		"repo_@#$%",
		"repo_ spaces ",
		"repo_01936d5a-0000-7abc-8def", // truncated
	}

	for _, id := range invalidIDs {
		_, err := ParseRepoID(id)
		assert.Error(t, err, "expected error for invalid UUID format: %s", id)
	}
}

func TestIsValidRepoID_Valid(t *testing.T) {
	id := GenerateRepoID()
	assert.True(t, IsValidRepoID(id), "expected valid repo ID to return true: %s", id)
}

func TestIsValidRepoID_Invalid(t *testing.T) {
	invalidIDs := []string{
		"invalid_01936d5a-0000-7abc-8def-0123456789ab",
		"01936d5a-0000-7abc-8def-0123456789ab",
		"",
		"repo_",
		"repo_!!!invalid",
		"repo_not-a-uuid",
	}

	for _, id := range invalidIDs {
		assert.False(t, IsValidRepoID(id), "expected invalid repo ID to return false: %s", id)
	}
}

func BenchmarkGenerateRepoID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRepoID()
	}
}

func BenchmarkParseRepoID(b *testing.B) {
	id := GenerateRepoID()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ParseRepoID(id)
	}
}

func BenchmarkIsValidRepoID(b *testing.B) {
	id := GenerateRepoID()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		IsValidRepoID(id)
	}
}
