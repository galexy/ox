package auth

import (
	"crypto/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Design decision: Server-side session generation with client fallback
// Rationale: Client-generated ULIDs have predictable timestamp component (48-bit);
// server binding enables revocation and signed claims. However, for offline/degraded
// mode, client generation provides graceful fallback - better than no session at all.
// See: docs/plan/drifting-exploring-quill.md for security analysis

const serverSessionPrefix = "oxsid_"

// NewServerSessionID generates a new server session ID in the format: oxsid_<ulid>.
// This is used as fallback when server doesn't provide a session ID (offline mode).
func NewServerSessionID() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	return serverSessionPrefix + id.String()
}

// IsValidServerSessionID validates that the session ID has the correct format
// (starts with "oxsid_" and contains a valid ULID).
// Works for both client-generated and server-generated session IDs.
func IsValidServerSessionID(id string) bool {
	if !strings.HasPrefix(id, serverSessionPrefix) {
		return false
	}

	ulidPart := strings.TrimPrefix(id, serverSessionPrefix)
	if len(ulidPart) == 0 {
		return false
	}

	_, err := ulid.Parse(ulidPart)
	return err == nil
}

// ServerSessionFromResponse validates and returns a server-provided session ID.
// If the server ID is empty or invalid, generates a new client-side session ID
// as fallback for offline/degraded operation.
//
// Design decision: Client fallback for offline
// Rationale: Graceful degradation - offline agents still get sessions;
// server-side preferred when available for security benefits.
func ServerSessionFromResponse(serverSessionID string) string {
	if serverSessionID == "" {
		return NewServerSessionID()
	}
	if IsValidServerSessionID(serverSessionID) {
		return serverSessionID
	}
	// invalid server ID - fall back to client generation
	return NewServerSessionID()
}
