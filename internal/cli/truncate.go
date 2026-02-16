package cli

import "strings"

// TruncateID shortens an ID for display, keeping prefix and suffix
// Examples:
//   - "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ" -> "oxsid_01JB...WXYZ" (verbose=false)
//   - "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ" -> "oxsid_01JBCDEFGHIJKLMNOPQRSTUVWXYZ" (verbose=true)
//   - "Oxa7b3" -> "Oxa7b3" (already short)
//
// Returns full ID if verbose=true or ID is already short (<=16 chars).
func TruncateID(id string, verbose bool) string {
	if verbose || len(id) <= 16 {
		return id
	}

	// for oxsid_ prefix IDs, keep first 12 chars + "..." + last 4 chars
	if strings.HasPrefix(id, "oxsid_") {
		return id[:12] + "..." + id[len(id)-4:]
	}

	// for other prefixed IDs (prj_, org_, tm_, usr_), same strategy
	if strings.Contains(id[:min(6, len(id))], "_") {
		if len(id) > 16 {
			return id[:12] + "..." + id[len(id)-4:]
		}
		return id
	}

	// generic UUIDs or other long IDs: keep first 8 chars + "..."
	if len(id) > 12 {
		return id[:8] + "..."
	}

	return id
}

// TruncateUUID shortens a UUID for display
// Examples:
//   - "a1b2c3d4-e5f6-7890-abcd-ef1234567890" -> "a1b2c3d4..." (verbose=false)
//   - "a1b2c3d4-e5f6-7890-abcd-ef1234567890" -> "a1b2c3d4-e5f6-7890-abcd-ef1234567890" (verbose=true)
//
// Returns full UUID if verbose=true or UUID is already short (<=12 chars).
func TruncateUUID(uuid string, verbose bool) string {
	if verbose || len(uuid) <= 12 {
		return uuid
	}
	return uuid[:8] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
