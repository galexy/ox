// Package repotools provides utilities for working with repository identifiers.
//
// The repo ID format is: "repo_" + standard UUIDv7 string
//
// UUIDv7 is used because it's time-sortable, which helps with concurrent init
// detection where the earlier timestamp wins as canonical. The standard UUID
// format preserves this time-sortability when sorted lexicographically.
//
// Example generated IDs:
//
//	repo_01936d5a-0000-7abc-8def-0123456789ab
//	repo_01936d5a-0001-7abc-8def-0123456789ab
//	repo_01936d5a-0002-7abc-8def-0123456789ab
package repotools

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

const (
	repoIDPrefix = "repo_"
)

// GenerateRepoID generates a new prefixed repo ID using UUIDv7.
// Format: "repo_" + standard UUIDv7 string (e.g., "repo_01936d5a-0000-7abc-8def-0123456789ab")
//
// UUIDv7 is time-sortable, and the standard format preserves this property
// when sorted lexicographically (e.g., ls -la .sageox/.repo_*).
func GenerateRepoID() string {
	uuidV7 := uuid.Must(uuid.NewV7())
	return repoIDPrefix + uuidV7.String()
}

// ParseRepoID parses a repo ID back to its underlying UUID.
// Returns an error if the ID is invalid or doesn't have the correct prefix.
func ParseRepoID(id string) (uuid.UUID, error) {
	if !strings.HasPrefix(id, repoIDPrefix) {
		return uuid.Nil, errors.New("invalid repo ID: missing 'repo_' prefix")
	}

	uuidStr := strings.TrimPrefix(id, repoIDPrefix)
	if len(uuidStr) == 0 {
		return uuid.Nil, errors.New("invalid repo ID: empty UUID portion")
	}

	return uuid.Parse(uuidStr)
}

// IsValidRepoID validates the format of a repo ID.
// Returns true if the ID has the correct prefix and can be parsed as a valid UUID.
func IsValidRepoID(id string) bool {
	_, err := ParseRepoID(id)
	return err == nil
}
